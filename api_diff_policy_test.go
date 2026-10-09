package workbench

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNativeAPIPolicyAllChangeKindsAndBatchedModelReplay(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native policy comparison")
	}
	type row struct {
		fact APIChange
		want APIAssessment
	}
	var rows []row
	add := func(kinds []string, code, action string, priority int64) {
		for _, kind := range kinds {
			rows = append(rows, row{APIChange{Kind: kind}, APIAssessment{code, action, priority}})
		}
	}
	add([]string{"package-added", "declaration-added"}, "surface-added", "exercise-new-declaration", 1)
	add([]string{"package-removed", "declaration-removed", "field-removed"}, "surface-removed", "rebuild-callers-and-run-examples", 2)
	add([]string{"shape", "field-type", "field-cardinality", "inputs", "output", "declaration-kind"}, "type-contract-changed", "rebuild-callers-and-run-examples", 2)
	add([]string{"field-name", "declaration-name", "package-name", "field-aliases"}, "source-names-changed", "recompile-name-references", 2)
	add([]string{"imports", "entry", "package-namespace"}, "package-wiring-changed", "exercise-package-entry", 2)
	add([]string{"field-order"}, "field-order-changed", "check-positional-construction", 2)
	add([]string{"unknown-extension"}, "unclassified-change", "add-policy-case", 2)
	add([]string{"none"}, "declarations-unchanged", "observe-runtime-behavior", 0)
	rows = append(rows,
		row{APIChange{Kind: "field-presence", InputUsed: true, BeforeRequired: true}, APIAssessment{"input-relaxed", "exercise-omitted-input", 1}},
		row{APIChange{Kind: "field-presence", OutputUsed: true, AfterRequired: true}, APIAssessment{"output-strengthened", "check-output-producers", 1}},
		row{APIChange{Kind: "field-presence", InputUsed: true, OutputUsed: true, BeforeRequired: true}, APIAssessment{"shared-presence-changed", "exercise-callers-and-producers", 2}},
		row{APIChange{Kind: "field-presence"}, APIAssessment{"usage-unobserved", "add-consumer-examples", 2}},
		row{APIChange{Kind: "field-added", InputUsed: true, AfterRequired: true}, APIAssessment{"required-input-added", "supply-required-input", 2}},
		row{APIChange{Kind: "field-added", OutputUsed: true, AfterRequired: true}, APIAssessment{"field-added", "check-record-construction", 1}},
	)
	// Repetition crosses the compiler's row limit; it is batching evidence,
	// not additional distinct policy examples or generalization evidence.
	facts := make([]APIChange, 135)
	want := make([]APIAssessment, 135)
	for i := range facts {
		facts[i] = rows[i%len(rows)].fact
		want[i] = rows[i%len(rows)].want
	}
	var previous []APIAssessment
	for _, model := range []string{"", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json"} {
		root := filepath.Join(t.TempDir(), "observation")
		if err := os.Mkdir(root, 0755); err != nil {
			t.Fatal(err)
		}
		values, summary, batches, err := assessAPIChanges(context.Background(), Options{Compiler: compiler}, root, model, facts)
		if err != nil || !reflect.DeepEqual(values, want) || batches != 2 || !summary.ReplayVerified || summary.NamedTotal != 0 || summary.SelectionPassed != 51 {
			t.Fatal("policy classification/replay", err, summary, batches)
		}
		if (model == "" && summary.ModelCalls != 0) || (model != "" && summary.ModelCalls < 1) {
			t.Fatal("model mode unobserved", summary)
		}
		if previous != nil && !reflect.DeepEqual(previous, values) {
			t.Fatal("model changed declaration judgments")
		}
		previous = values
	}
}
