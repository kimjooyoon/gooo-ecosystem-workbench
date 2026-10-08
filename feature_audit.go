package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"slices"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type FeatureAuditCase struct {
	ID            string                             `json:"id"`
	Family        string                             `json:"family"`
	Choices       []jointdecision.RecordChoice       `json:"choices,omitempty"`
	OriginChoices []jointdecision.RecordOriginChoice `json:"origin_choices,omitempty"`
	Graph         *jointdecision.RecordGraphInput    `json:"graph,omitempty"`
	AcceptedMasks []uint16                           `json:"accepted_masks"`
}

type FeatureAuditInput struct {
	Schema         string             `json:"schema"`
	FeatureVersion string             `json:"feature_version"`
	LabelSource    string             `json:"label_source"`
	Cases          []FeatureAuditCase `json:"cases"`
}

type FeatureCollisionGroup struct {
	FeatureSHA256 string   `json:"feature_sha256"`
	Rows          []string `json:"rows"`
	Votes         [8]int   `json:"acceptable_row_counts_by_mask"`
	Maximum       int      `json:"maximum_compatible_rows"`
}

type FeatureAuditPrediction struct {
	ID        string `json:"id"`
	Mask      uint16 `json:"mask"`
	Satisfied bool   `json:"satisfied"`
}

type FeatureAuditReport struct {
	Schema              string                   `json:"schema"`
	InputSHA256         string                   `json:"input_sha256"`
	FeatureVersion      string                   `json:"feature_version"`
	ProjectionVersion   string                   `json:"projection_module_version"`
	LabelSource         string                   `json:"label_source"`
	Rows                int                      `json:"rows"`
	Families            int                      `json:"families"`
	DistinctFeatures    int                      `json:"distinct_feature_vectors"`
	ConflictingGroups   int                      `json:"conflicting_feature_groups"`
	MaximumCompatible   int                      `json:"maximum_compatible_rows"`
	Groups              []FeatureCollisionGroup  `json:"groups"`
	ModelObserved       bool                     `json:"model_observed"`
	ModelCalls          int                      `json:"model_calls"`
	ModelSatisfied      int                      `json:"model_satisfied_rows"`
	Predictions         []FeatureAuditPrediction `json:"predictions,omitempty"`
	ModelMetadataSHA256 string                   `json:"model_metadata_sha256,omitempty"`
	ModelWeightsSHA256  string                   `json:"model_weights_sha256,omitempty"`
	Policy              FeatureAuditPolicy       `json:"policy"`
	PolicySourceSHA256  string                   `json:"policy_source_sha256"`
	PolicyCompilerSHA   string                   `json:"policy_compiler_source_sha"`
	PolicyReplayed      bool                     `json:"policy_runtime_replayed"`
	Scope               string                   `json:"scope"`
}

type FeatureAuditPolicy struct {
	Code    string `json:"code"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

func readFeatureAuditInput(raw []byte) (FeatureAuditInput, error) {
	var input FeatureAuditInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return input, err
	}
	if input.Schema != "gooo/record-feature-audit-input/v1" || !auditFeatureVersion(input.FeatureVersion) ||
		input.LabelSource == "" || len(input.Cases) == 0 || len(input.Cases) > 4096 {
		return input, fmt.Errorf("a supported feature contract, label source and 1..4096 cases are required")
	}
	seen := make(map[string]bool, len(input.Cases))
	for _, row := range input.Cases {
		if row.ID == "" || row.Family == "" || seen[row.ID] || len(row.AcceptedMasks) == 0 || !auditChoiceShape(row, input.FeatureVersion) {
			return input, fmt.Errorf("unique row IDs, source families, exactly three choices and nonempty accepted masks are required")
		}
		seen[row.ID] = true
		var masks [8]bool
		for _, mask := range row.AcceptedMasks {
			if mask >= 8 || masks[mask] {
				return input, fmt.Errorf("accepted masks must be distinct values in 0..7")
			}
			masks[mask] = true
		}
	}
	return input, nil
}

// Group keys contain every float32 bit, not only a hash or rounded summary.
// Equal features require one deterministic prediction; the largest acceptable
// label count gives that group's empirical upper bound for any such predictor.
func auditFeatureRows(ctx context.Context, input FeatureAuditInput, model *jointdecision.ThreeModel) (FeatureAuditReport, error) {
	r := FeatureAuditReport{Schema: "gooo/record-feature-audit/v1", FeatureVersion: input.FeatureVersion,
		ProjectionVersion: moduleVersion("github.com/kimjooyoon/gooo-decision-runtime"),
		LabelSource:       input.LabelSource, Rows: len(input.Cases), ModelObserved: model != nil,
		Scope: "Empirical row-weighted upper bound for a deterministic chooser using exactly these features and the supplied accepted-mask labels; repeated source families are counted separately; no generalization, label correctness, training or runtime claim."}
	if !auditFeatureVersion(input.FeatureVersion) || model != nil && model.FeatureVersion() != input.FeatureVersion {
		return r, fmt.Errorf("audit projection and model require the same explicit feature contract")
	}
	groups := map[string]*FeatureCollisionGroup{}
	families := map[string]bool{}
	var features [jointdecision.ThreeFeatureDim]float32
	var bits [4 * jointdecision.ThreeFeatureDim]byte
	var workspace jointdecision.ThreeWorkspace
	for _, row := range input.Cases {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		if err := projectAuditRow(row, input.FeatureVersion, &features); err != nil {
			return r, fmt.Errorf("%s: %w", row.ID, err)
		}
		for i, value := range features {
			binary.LittleEndian.PutUint32(bits[4*i:], math.Float32bits(value))
		}
		key := string(bits[:])
		if groups[key] == nil {
			groups[key] = &FeatureCollisionGroup{FeatureSHA256: fmt.Sprintf("%x", sha256.Sum256(bits[:]))}
		}
		group := groups[key]
		group.Rows = append(group.Rows, row.ID)
		for _, mask := range row.AcceptedMasks {
			group.Votes[mask]++
		}
		families[row.Family] = true
		if model != nil {
			var prediction jointdecision.ThreePrediction
			if err := predictAuditFeatures(model, &features, &workspace, &prediction); err != nil {
				return r, err
			}
			matched := slices.Contains(row.AcceptedMasks, prediction.Mask)
			r.ModelCalls++
			if matched {
				r.ModelSatisfied++
			}
			r.Predictions = append(r.Predictions, FeatureAuditPrediction{row.ID, prediction.Mask, matched})
		}
	}
	for _, group := range groups {
		slices.Sort(group.Rows)
		group.Maximum = slices.Max(group.Votes[:])
		r.MaximumCompatible += group.Maximum
		if group.Maximum < len(group.Rows) {
			r.ConflictingGroups++
		}
		r.Groups = append(r.Groups, *group)
	}
	slices.SortFunc(r.Groups, func(a, b FeatureCollisionGroup) int {
		return slices.Compare(a.Rows, b.Rows)
	})
	r.Families, r.DistinctFeatures = len(families), len(groups)
	return r, nil
}

// Go measures exact input collisions; the next action is implemented in Gooo.
func AuditRecordFeatures(ctx context.Context, o Options, raw []byte) (FeatureAuditReport, error) {
	input, err := readFeatureAuditInput(raw)
	if err != nil {
		return FeatureAuditReport{}, err
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return FeatureAuditReport{}, err
	}
	var model *jointdecision.ThreeModel
	modelPath, err := prepareModel(o.Model, root)
	if err != nil {
		return FeatureAuditReport{}, err
	}
	if modelPath != "" {
		model, err = loadAuditModel(modelPath, input.FeatureVersion)
		if err != nil {
			return FeatureAuditReport{}, err
		}
	}
	r, err := auditFeatureRows(ctx, input, model)
	if err != nil {
		return r, err
	}
	r.InputSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
	if model != nil {
		r.ModelMetadataSHA256, r.ModelWeightsSHA256 = model.MetadataSHA256(), model.WeightsSHA256()
	}
	if err = write(filepath.Join(root, "input.json"), raw); err != nil {
		return r, err
	}
	observation := map[string]any{"rows": r.Rows, "maximum": r.MaximumCompatible,
		"model_observed": r.ModelObserved, "model_matched": r.ModelSatisfied}
	cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{
		map[string]any{"inputs": map[string]any{"ObservationEcho": observation}, "expected": map[string]any{"ObservationEcho": observation}},
	}})
	if err != nil {
		return r, err
	}
	rawExecution, execution, err := runRecipe(ctx, o, root, "feature-audit", "assessment", "", cases)
	if err != nil {
		return r, err
	}
	if execution.Runtime.Calls != 0 || execution.Runtime.Passed != 1 || execution.Runtime.Total != 1 {
		return r, fmt.Errorf("feature assessment lost its observation or used inference")
	}
	if err = actualFor(execution, "featureaudit://activity/assess", &r.Policy); err != nil {
		return r, err
	}
	var replay struct {
		Runtime struct {
			Projection bool `json:"projection_replayed"`
			Runtime    bool `json:"runtime_replayed"`
		} `json:"runtime"`
	}
	if err = json.Unmarshal(rawExecution, &replay); err != nil || !replay.Runtime.Projection || !replay.Runtime.Runtime {
		return r, fmt.Errorf("Gooo feature assessment did not replay")
	}
	r.PolicySourceSHA256 = execution.Composition.OriginalSourceSHA
	r.PolicyCompilerSHA, r.PolicyReplayed = execution.Runtime.Source, true
	return r, save(filepath.Join(root, "feature-audit.json"), r)
}
