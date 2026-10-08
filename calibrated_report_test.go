package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalibratedReportRefinesNumericBodyAndConstructsRecord(t *testing.T) {
	for _, mode := range []string{"fixed", "model", "round-limit", "no-alternative"} {
		t.Run(mode, func(t *testing.T) {
			o := refinementOptions(t)
			root := "examples/calibrated-report/"
			o.Source, o.Activity, o.SearchPolicy = root+"source.gooo", "Energy", true
			o.Cases, o.EvaluationCases = root+"feedback-cases.json", root+"evaluation-cases.json"
			o.Policy = "examples/search-policy/policy.gooo"
			if mode == "model" || mode == "round-limit" {
				o.Model = "builtin"
			}
			if mode == "round-limit" {
				o.MaxRounds = 1
			}
			if mode == "no-alternative" {
				raw, err := os.ReadFile(o.Source)
				if err != nil {
					t.Fatal(err)
				}
				source := strings.Replace(string(raw), "    search_alternative \"shared_fit\" grammar \"integer-hole-quadratic/v2\" max_candidates \"16\"\n", "", 1)
				o.Source = filepath.Join(t.TempDir(), "source.gooo")
				if err := os.WriteFile(o.Source, []byte(source), 0644); err != nil {
					t.Fatal(err)
				}
			}
			r, err := Refine(context.Background(), o)
			if err != nil || !r.Dispatched || r.Initial.Passed != 2 || r.Initial.Total != 20 || r.Evaluation == nil {
				t.Fatal("calibration did not retain the initial partial graph", err, r)
			}
			if mode == "fixed" || mode == "model" {
				if r.Status != "PASS" || r.Final.Passed != 20 || r.Final.Total != 20 || r.Evaluation.Passed != 6 || r.Evaluation.Total != 6 || r.Rounds != 2 {
					t.Fatal("declared shared fit did not complete both graph activities", r)
				}
			} else if r.Status != "PROGRESS" || r.Final.Passed != 2 || r.Evaluation.Passed != 0 || r.Rounds != 1 {
				t.Fatal("bounded control hid unresolved outputs", r)
			}
			if o.Model == "builtin" && (r.InitialModelCalls != 1 || r.RefinementModelCalls != r.Rounds) || o.Model == "" && (r.InitialModelCalls != 0 || r.RefinementModelCalls != 0) {
				t.Fatal("record model calls did not match the selected mode", r)
			}
			for _, name := range []string{"final-result.json", "evaluation-result.json"} {
				raw, err := os.ReadFile(filepath.Join(o.Out, name))
				var replay result
				if err != nil || json.Unmarshal(raw, &replay) != nil || replay.Generated || replay.Runtime.Calls != 0 {
					t.Fatal("saved calibration graph invoked new inference", name, err)
				}
			}
		})
	}
}
