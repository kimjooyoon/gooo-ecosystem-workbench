package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
)

const jointCasesSchema = "gooo/body-composition-cases/v1"

type jointCases struct {
	Schema string            `json:"schema"`
	Cases  []json.RawMessage `json:"cases"`
}

type JointFeedbackRow struct {
	Index           int    `json:"evaluation_index"`
	Matched         int64  `json:"matched"`
	Total           int64  `json:"total"`
	Known           bool   `json:"already_retained_or_duplicate"`
	RowSHA256       string `json:"original_row_sha256"`
	CanonicalSHA256 string `json:"canonical_row_sha256"`
	Include         bool   `json:"include"`
	Reason          string `json:"reason"`
	raw             json.RawMessage
}

type JointFeedbackUpdate struct {
	Prepared           bool               `json:"prepared"`
	Consumed           bool               `json:"consumed_by_next_round"`
	StopReason         string             `json:"stop_reason,omitempty"`
	ResultSHA256       string             `json:"originating_result_sha256"`
	EvaluationSHA256   string             `json:"evaluation_file_sha256"`
	PreviousSHA256     string             `json:"previous_cases_sha256"`
	NextSHA256         string             `json:"next_cases_sha256,omitempty"`
	NextCasesFile      string             `json:"next_cases_file,omitempty"`
	PolicyResultSHA256 string             `json:"policy_result_sha256,omitempty"`
	AddedIndices       []int              `json:"added_indices"`
	SelectedIndices    []int              `json:"selected_indices"`
	Rows               []JointFeedbackRow `json:"rows"`
}

func jointDigest(raw []byte) string { return fmt.Sprintf("%x", sha256.Sum256(raw)) }

func jointCanonical(raw []byte) ([]byte, error) {
	value, err := decodeValue(raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func readJointCases(raw []byte, inputOnly bool) (jointCases, error) {
	var doc jointCases
	if len(raw) > 32<<10 {
		return doc, fmt.Errorf("cases exceed 32 KiB")
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return doc, err
	}
	if inputOnly && doc.Schema == "gooo/body-composition-inputs/v1" {
		var input struct {
			Inputs []json.RawMessage `json:"inputs"`
		}
		if err := json.Unmarshal(raw, &input); err != nil {
			return doc, err
		}
		for _, row := range input.Inputs {
			doc.Cases = append(doc.Cases, json.RawMessage(`{"inputs":`+string(row)+`}`))
		}
	} else if doc.Schema != jointCasesSchema {
		return doc, fmt.Errorf("unsupported construction/evaluation cases schema")
	}
	if len(doc.Cases) < 1 || len(doc.Cases) > 128 {
		return doc, fmt.Errorf("cases require 1..128 rows")
	}
	return doc, nil
}

// collectJointFeedback recounts named expectations from original rows. Exact
// JSON numbers, complete multi-output rows and contradictory oracles survive.
func collectJointFeedback(current, evaluation, raw []byte) (jointCases, []JointFeedbackRow, error) {
	doc, err := readJointCases(current, false)
	if err != nil {
		return doc, nil, err
	}
	return collectJointFeedbackRows(doc, evaluation, raw)
}

var emptyGraphHistory = []byte(`{"schema":"gooo/body-composition-cases/v1","cases":[]}` + "\n")

// Only a saved graph's first caller feedback starts without joint caller rows.
// Source-local cases stay in the native contracts, independently of this seed.
func collectInitialGraphFeedback(evaluation, raw []byte) (jointCases, []JointFeedbackRow, error) {
	return collectJointFeedbackRows(jointCases{Schema: jointCasesSchema, Cases: []json.RawMessage{}}, evaluation, raw)
}

func collectJointFeedbackRows(doc jointCases, evaluation, raw []byte) (jointCases, []JointFeedbackRow, error) {
	eval, err := readJointCases(evaluation, true)
	if err != nil {
		return doc, nil, err
	}
	var envelope struct {
		Schema string `json:"schema"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return doc, nil, err
	}
	aliases := map[string]string{}
	if envelope.Schema == packageJointSchema {
		var observed *PackageConstructionObservation
		raw, observed, err = packageJointOutput(raw)
		if err != nil {
			return doc, nil, err
		}
		for _, activity := range observed.Activities {
			aliases[activity.Lowered] = activity.Package + ":" + activity.Activity
		}
	}
	var report struct {
		Composition struct {
			Plan struct {
				Schema     string                      `json:"schema"`
				Activities []struct{ Name, ID string } `json:"activities"`
			} `json:"plan"`
		} `json:"composition"`
		Construction struct {
			Selected struct {
				Plan struct {
					Activities []struct{ Name, ID string } `json:"activities"`
				} `json:"plan"`
			} `json:"selected"`
		} `json:"construction"`
		Evaluation result `json:"evaluation"`
	}
	if err = json.Unmarshal(raw, &report); err != nil {
		return doc, nil, err
	}
	if report.Composition.Plan.Schema == "gooo/body-composition-plan/v1" {
		if len(report.Construction.Selected.Plan.Activities) != 0 {
			return doc, nil, fmt.Errorf("feedback has ambiguous composition and construction")
		}
		report.Construction.Selected.Plan.Activities = report.Composition.Plan.Activities
		if err = json.Unmarshal(raw, &report.Evaluation); err != nil {
			return doc, nil, err
		}
	}
	names := map[string]string{}
	ids := map[string]bool{}
	for _, activity := range report.Construction.Selected.Plan.Activities {
		if name := aliases[activity.Name]; name != "" {
			activity.Name = name
		}
		if activity.Name == "" || activity.ID == "" || names[activity.Name] != "" || ids[activity.ID] {
			return doc, nil, fmt.Errorf("ambiguous evaluation activity identity")
		}
		names[activity.Name], ids[activity.ID] = activity.ID, true
	}
	traces := report.Evaluation.Runtime.Traces
	if len(traces) != len(eval.Cases) {
		return doc, nil, fmt.Errorf("evaluation trace count differs from original rows")
	}
	positions := make([]int, len(traces))
	seenIndices := make([]bool, len(traces))
	for pos, trace := range traces {
		if trace.CaseIndex < 0 || trace.CaseIndex >= len(traces) || seenIndices[trace.CaseIndex] {
			return doc, nil, fmt.Errorf("duplicate or invalid evaluation case index")
		}
		positions[trace.CaseIndex], seenIndices[trace.CaseIndex] = pos, true
	}
	known := map[string]bool{}
	for _, row := range doc.Cases {
		canonical, err := jointCanonical(row)
		if err != nil {
			return doc, nil, err
		}
		known[jointDigest(canonical)] = true
	}
	rows := make([]JointFeedbackRow, len(eval.Cases))
	for index, row := range eval.Cases {
		canonical, err := jointCanonical(row)
		if err != nil {
			return doc, nil, err
		}
		observation := JointFeedbackRow{Index: index, RowSHA256: jointDigest(row), CanonicalSHA256: jointDigest(canonical), raw: row}
		observation.Known = known[observation.CanonicalSHA256]
		known[observation.CanonicalSHA256] = true
		var original struct {
			Expected map[string]json.RawMessage `json:"expected"`
		}
		if err = json.Unmarshal(row, &original); err != nil {
			return doc, nil, err
		}
		deliveries := traces[positions[index]].Deliveries
		byID := map[string]int{}
		for pos, delivery := range deliveries {
			if _, exists := byID[delivery.ID]; exists || !ids[delivery.ID] {
				return doc, nil, fmt.Errorf("duplicate or unknown evaluation activity")
			}
			byID[delivery.ID] = pos
		}
		for name, expected := range original.Expected {
			id := names[name]
			pos, exists := byID[id]
			if id == "" || !exists {
				return doc, nil, fmt.Errorf("evaluation omitted expected activity %q", name)
			}
			want, err := decodeValue(expected)
			if err != nil {
				return doc, nil, err
			}
			recorded, err := decodeValue(deliveries[pos].Expected)
			if err != nil || !reflect.DeepEqual(want, recorded) {
				return doc, nil, fmt.Errorf("evaluation expectation differs from original row %d", index)
			}
			actual, err := decodeValue(deliveries[pos].Actual)
			if err != nil {
				return doc, nil, err
			}
			observation.Total++
			if reflect.DeepEqual(want, actual) {
				observation.Matched++
			}
		}
		rows[index] = observation
	}
	return doc, rows, nil
}

// prepareJointFeedback executes the Gooo row policy once for the whole batch.
// A prepared file is not counted as consumed until a subsequent round succeeds.
func prepareJointFeedback(ctx context.Context, o Options, root, label string, current, evaluation, raw []byte) (*JointFeedbackUpdate, error) {
	doc, rows, err := collectJointFeedback(current, evaluation, raw)
	if err != nil {
		return nil, err
	}
	return prepareCollectedJointFeedback(ctx, o, root, label, current, evaluation, raw, doc, rows)
}

func prepareInitialGraphFeedback(ctx context.Context, o Options, root string, evaluation, raw []byte) (*JointFeedbackUpdate, error) {
	doc, rows, err := collectInitialGraphFeedback(evaluation, raw)
	if err != nil {
		return nil, err
	}
	return prepareCollectedJointFeedback(ctx, o, root, "origin-feedback", emptyGraphHistory, evaluation, raw, doc, rows)
}

func prepareCollectedJointFeedback(ctx context.Context, o Options, root, label string, current, evaluation, raw []byte,
	doc jointCases, rows []JointFeedbackRow) (*JointFeedbackUpdate, error) {
	update := &JointFeedbackUpdate{ResultSHA256: jointDigest(raw), EvaluationSHA256: jointDigest(evaluation), PreviousSHA256: jointDigest(current), Rows: rows, AddedIndices: []int{}}
	cases := make([]any, len(rows))
	for i, row := range rows {
		input := map[string]any{"matched": row.Matched, "total": row.Total, "known": row.Known}
		cases[i] = map[string]any{"inputs": map[string]any{"ObservationEcho": input}, "expected": map[string]any{"ObservationEcho": input}}
	}
	caseBytes, err := json.Marshal(map[string]any{"schema": jointCasesSchema, "cases": cases})
	if err != nil {
		return update, err
	}
	policyRaw, r, err := runRecipe(ctx, o, root, "joint-feedback", label+"-policy", "", caseBytes)
	if err != nil {
		return update, err
	}
	update.PolicyResultSHA256 = jointDigest(policyRaw)
	if r.Runtime.Calls != 0 || r.Runtime.Passed != len(rows) || r.Runtime.Total != len(rows) || len(r.Runtime.Traces) != len(rows) {
		return update, fmt.Errorf("feedback policy lost observations or predicted during execution")
	}
	seen := make([]bool, len(rows))
	for _, trace := range r.Runtime.Traces {
		i := trace.CaseIndex
		if i < 0 || i >= len(rows) || seen[i] {
			return update, fmt.Errorf("invalid feedback policy case index")
		}
		seen[i] = true
		count := 0
		for _, delivery := range trace.Deliveries {
			if delivery.ID != "jointfeedback://activity/decide" {
				continue
			}
			var decision struct {
				Include *bool  `json:"include"`
				Reason  string `json:"reason"`
			}
			if err = json.Unmarshal(delivery.Actual, &decision); err != nil {
				return update, err
			}
			if decision.Include == nil || decision.Reason == "" {
				return update, fmt.Errorf("feedback policy omitted its typed decision")
			}
			rows[i].Include, rows[i].Reason = *decision.Include, decision.Reason
			count++
		}
		if count != 1 {
			return update, fmt.Errorf("missing or duplicate feedback decision")
		}
	}
	for _, row := range rows {
		if row.Include {
			doc.Cases = append(doc.Cases, row.raw)
			update.SelectedIndices = append(update.SelectedIndices, row.Index)
		}
	}
	if len(update.SelectedIndices) == 0 {
		update.StopReason = "no-new-counterexamples"
		return update, nil
	}
	next, err := json.Marshal(doc)
	if err != nil {
		return update, err
	}
	next = append(next, '\n')
	if len(doc.Cases) > 128 || len(next) > 32<<10 {
		update.StopReason = "construction-cases-limit"
		return update, nil
	}
	update.NextCasesFile = label + "-cases.json"
	if err = write(filepath.Join(root, update.NextCasesFile), next); err != nil {
		return update, err
	}
	update.NextSHA256, update.Prepared = jointDigest(next), true
	update.AddedIndices = append(update.AddedIndices, update.SelectedIndices...)
	return update, nil
}
