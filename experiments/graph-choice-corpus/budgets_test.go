package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedModelConstructionAndPartialBudgets(t *testing.T) {
	compiler := "6c4a2b14f6f70c8efca4e08df4ac5e6ce572f9ff"
	for _, mode := range []string{"all-data", "held-out", "deterministic"} {
		files := readNativeEvidence(t, "../../publication/graph-chooser-20261008/native-"+mode+".tar.gz")
		calls := 1
		if mode == "deterministic" {
			calls = 0
		}
		for _, family := range families {
			fold := family.name
			if mode == "all-data" {
				fold = "all-data-demonstration"
			}
			for requested := range uint16(8) {
				id := fmt.Sprintf("%s-mixed-0-request%d", family.name, requested)
				source := files[id+"/construction/original.gooo"]
				for _, replay := range []bool{false, true} {
					name := "/result.json"
					if replay {
						name = "/replay.json"
					}
					if err := checkNativeModel(files[id+name], source, family, requested, compiler, !replay, calls); err != nil {
						t.Fatal(mode, id, name, err)
					}
					if calls == 1 {
						var n nativeExport
						if err := json.Unmarshal(files[id+name], &n); err != nil {
							t.Fatal(err)
						}
						m := n.Composition.Steps[0].Generation.Report.Assembly.Model
						for file, identity := range map[string]string{"model.json": m.MetadataSHA, "weights.bin": m.WeightsSHA} {
							raw, err := os.ReadFile(filepath.Join("../../models/graph-chooser-20261008", fold, "qat_ternary", file))
							if err != nil || strings.TrimPrefix(digest(raw), "sha256:") != identity {
								t.Fatal("native model differs from published trained artifact", err)
							}
						}
					}
				}
				for _, budget := range []int{1, 2, 4, 8} {
					prefix := fmt.Sprintf("%s/budget-%d/", id, budget)
					if _, err := checkBudget(files[prefix+"generation.json"], files[prefix+"source.gooo"], family, requested, budget, calls); err != nil {
						t.Fatal(mode, prefix, err)
					}
				}
			}
		}
	}
}

func TestBudgetRejectsMissingCasesAndIncorrectCompleteness(t *testing.T) {
	files := readNativeEvidence(t, "../../publication/graph-chooser-20261008/native-held-out.tar.gz")
	prefix := "filenames-mixed-0-request0/budget-1/"
	mutations := map[string]func(*finiteExport){
		"missing-call": func(n *finiteExport) { n.Report.Assembly.Calls = nil },
		"source":       func(n *finiteExport) { n.Report.Assembly.SourceSHA = "different" },
		"budget":       func(n *finiteExport) { n.Report.Assembly.Budget = 8 },
		"proposal":     func(n *finiteExport) { n.Report.Assembly.Prediction.Mask ^= 1 },
		"context":      func(n *finiteExport) { n.Report.Assembly.Context = nil },
		"actual":       func(n *finiteExport) { n.Report.Assembly.Cases[0].Actual = json.RawMessage(`{}`) },
		"expected":     func(n *finiteExport) { n.Report.Assembly.Cases[0].Expected = json.RawMessage(`{}`) },
		"fields":       func(n *finiteExport) { n.Report.Assembly.FieldsPass++ },
		"status":       func(n *finiteExport) { n.Report.Assembly.Status = "COMPLETE_FINITE" },
		"ranking":      func(n *finiteExport) { n.Report.Assembly.Ranking[1] = n.Report.Assembly.Ranking[0] },
		"attempt":      func(n *finiteExport) { n.Report.Assembly.Attempts[0].Mask ^= 1 },
		"omit-case-and-adjust-total": func(n *finiteExport) {
			a := &n.Report.Assembly
			a.Cases = a.Cases[1:]
			a.Total--
			a.FieldsTotal -= 3
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var n finiteExport
			if err := json.Unmarshal(files[prefix+"generation.json"], &n); err != nil {
				t.Fatal(err)
			}
			mutate(&n)
			raw, _ := json.Marshal(n)
			if _, err := checkBudget(raw, files[prefix+"source.gooo"], families[0], 0, 1, 1); err == nil {
				t.Fatal("incomplete observation accepted")
			}
		})
	}
	for _, mutate := range []func(*nativeExport){
		func(n *nativeExport) { n.Composition.Steps[0].Generation.Report.Assembly.Context = nil },
		func(n *nativeExport) { n.Composition.Steps[0].Generation.Report.Assembly.Context.Feature = "legacy" },
		func(n *nativeExport) { n.Composition.Steps[0].Generation.Report.Assembly.Prediction.Mask = 8 },
	} {
		id := "filenames-mixed-0-request0/"
		var n nativeExport
		if err := json.Unmarshal(files[id+"result.json"], &n); err != nil {
			t.Fatal(err)
		}
		mutate(&n)
		raw, _ := json.Marshal(n)
		if err := checkNativeModel(raw, files[id+"construction/original.gooo"], families[0], 0, "6c4a2b14f6f70c8efca4e08df4ac5e6ce572f9ff", true, 1); err == nil {
			t.Fatal("missing graph proposal accepted")
		}
	}
}
