package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchPolicyDispatchesExhaustedSpaceAndGrammar(t *testing.T) {
	for _, mode := range []string{"cap", "grammar", "mixed-model"} {
		t.Run(mode, func(t *testing.T) {
			o := searchRefinementOptions(t, mode == "mixed-model")
			o.SearchPolicy, o.Source, o.Policy = true, "examples/search-policy/source.gooo", "examples/search-policy/policy.gooo"
			want := "expand-search-space"
			if mode == "mixed-model" {
				o.Source, o.Model = "examples/search-policy/mixed.gooo", "builtin"
			}
			raw, err := os.ReadFile(o.Source)
			if err != nil {
				t.Fatal(err)
			}
			source := strings.Replace(string(raw), `attempts "1"`, `attempts "2"`, 1)
			if mode == "grammar" {
				source = strings.Replace(source, `max_candidates "2"`, `max_candidates "16"`, 1)
				source = strings.Replace(source, `attempts "2"`, `attempts "8"`, 1)
				want = "expand-declared-choices"
			}
			o.Source = filepath.Join(t.TempDir(), "source.gooo")
			if err = os.WriteFile(o.Source, []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			r, err := Refine(context.Background(), o)
			if err != nil || !r.SearchPolicy || !r.Dispatched || r.InitialPlan.Action != want || r.Status != "PASS" || r.EvaluationStatus != "PASS" {
				t.Fatal("search proposal did not revise source", err, r)
			}
			if mode == "mixed-model" && (r.InitialModelCalls != 1 || r.RefinementModelCalls != r.Rounds) {
				t.Fatal("mixed search lost the record model observations", r)
			}
			selected, err := os.ReadFile(filepath.Join(o.Out, r.SelectedSource))
			if err != nil || !strings.Contains(string(selected), `grammar "integer-hole-residual/v1" intent`) {
				t.Fatal("retained source did not select contextual grammar", err)
			}
			unchanged, _ := os.ReadFile(o.Source)
			if string(unchanged) != source {
				t.Fatal("caller source was overwritten")
			}
			final, _ := os.ReadFile(filepath.Join(o.Out, "final-result.json"))
			var replay result
			if json.Unmarshal(final, &replay) != nil || replay.Generated || replay.Runtime.Calls != 0 {
				t.Fatal("saved program called inference again")
			}
		})
	}
}

func TestSearchPolicyRequiresIntegerSourceBeforeLoadingModel(t *testing.T) {
	o := refinementOptions(t)
	o.SearchPolicy, o.Model = true, "/missing/model.json"
	_, err := Refine(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "source-owned integer search") {
		t.Fatal("search policy accepted a record target or loaded its model", err)
	}
	if shouldRefine("expand-search-space", false) || shouldRefine("expand-declared-choices", false) || shouldRefine("observe-new-inputs", true) {
		t.Fatal("unrequested search revisions were dispatched")
	}
}
