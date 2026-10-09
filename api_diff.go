package workbench

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
)

type APIDifference struct {
	Schema            string         `json:"schema"`
	Decision          string         `json:"decision"`
	BeforeDigest      string         `json:"before_interface_digest"`
	AfterDigest       string         `json:"after_interface_digest"`
	SourceChanged     bool           `json:"source_identity_changed"`
	ChangeCount       int            `json:"change_count"`
	ClassifiedChanges int            `json:"classified_changes"`
	Changes           []APIChange    `json:"changes"`
	Unchanged         *APIAssessment `json:"unchanged_assessment,omitempty"`
	Observation       Summary        `json:"observation"`
	ReplayBatches     int            `json:"replay_batches"`
	Scope             string         `json:"scope"`
}

// CompareAPI obtains source-resolved declarations from the compiler. Go joins
// stable identities; Gooo owns change classification and next operations.
func CompareAPI(ctx context.Context, o Options, beforePath, afterPath string) (APIDifference, error) {
	var report APIDifference
	if beforePath == "" || afterPath == "" {
		return report, fmt.Errorf("api-diff requires --before and --after workspace manifests")
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return report, err
	}
	views := make([]APIInterface, 2)
	for i, path := range []string{beforePath, afterPath} {
		raw, err := command(ctx, o.Compiler, "package", "interface", "--json", path)
		if len(raw) > 0 {
			if e := write(filepath.Join(root, []string{"before-interface.json", "after-interface.json"}[i]), raw); e != nil {
				return report, e
			}
		}
		if err != nil {
			return report, err
		}
		views[i], err = readAPIInterface(raw)
		if err != nil {
			return report, err
		}
	}
	model, err := prepareModel(o.Model, root)
	if err != nil {
		return report, err
	}
	changes := apiChangeFacts(views[0], views[1])
	if err = save(filepath.Join(root, "changes.json"), changes); err != nil {
		return report, err
	}
	inputs := changes
	if len(inputs) == 0 {
		inputs = []APIChange{{Kind: "none"}}
	}
	assessments, summary, batches, err := assessAPIChanges(ctx, o, root, model, inputs)
	if err != nil {
		return report, err
	}
	report = APIDifference{Schema: "gooo/api-declaration-difference/v1", Decision: "OBSERVED",
		BeforeDigest: views[0].Digest, AfterDigest: views[1].Digest, SourceChanged: views[0].ImageDigest != views[1].ImageDigest,
		ChangeCount: len(changes), Changes: changes, Observation: summary, ReplayBatches: batches,
		Scope: "Declared workspace changes and Gooo-authored next operations. Usage includes all declared activity signatures in either snapshot; external consumers and runtime behavior are unobserved. Selection cases are finite policy examples; actual changes have no supplied correctness labels.",
	}
	if len(changes) == 0 {
		report.Unchanged = &assessments[0]
	}
	for index := range changes {
		changes[index].Assessment = &assessments[index]
		if assessments[index].Code != "unclassified-change" {
			report.ClassifiedChanges++
		}
	}
	return report, save(filepath.Join(root, "api-diff.json"), report)
}

func assessAPIChanges(ctx context.Context, o Options, root, model string, changes []APIChange) ([]APIAssessment, Summary, int, error) {
	var assessments []APIAssessment
	var summary Summary
	const batchSize = 128 // Compiler input-only execution's existing row limit.
	var generated string
	batches := 0
	for start := 0; start < len(changes); start += batchSize {
		batch := changes[start:min(start+batchSize, len(changes))]
		index := start / batchSize
		raw, built, err := executeAPIBatch(ctx, o, root, model, batch, index, index != 0)
		if err != nil {
			return nil, summary, batches, err
		}
		values, err := apiAssessments(built, len(batch))
		if err != nil {
			return nil, summary, batches, err
		}
		if index == 0 {
			summary, err = summarize(raw, "api-changes", map[bool]string{true: "model", false: "deterministic"}[model != ""])
			if err != nil || summary.SelectionPassed != 51 || summary.SelectionTotal != 51 || summary.NamedTotal != 0 {
				return nil, summary, batches, fmt.Errorf("API policy selection incomplete: %+v: %v", summary, err)
			}
			generated = built.Composition.GeneratedSHA
			_, replay, err := executeAPIBatch(ctx, o, root, "", batch, 0, true)
			if err != nil {
				return nil, summary, batches, err
			}
			again, err := apiAssessments(replay, len(batch))
			if err != nil || !reflect.DeepEqual(values, again) || generated != replay.Composition.GeneratedSHA {
				return nil, summary, batches, fmt.Errorf("saved API policy replay differs: %v", err)
			}
		} else if generated != built.Composition.GeneratedSHA {
			return nil, summary, batches, fmt.Errorf("API policy changed between batches")
		}
		assessments = append(assessments, values...)
		batches++
	}
	summary.ReplayVerified = true
	return assessments, summary, batches, nil
}
