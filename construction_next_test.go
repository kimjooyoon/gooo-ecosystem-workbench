package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const constructionFixture = `{"composition":{"preparations":[{"generation":{"report":{
"activity_id":"example/helper","record_assembly":{"passed":2,"total":3,"ranking":[0,1,2,3],
"attempts":[{"mask":0}],"control":{"decisions":[{"input":{"budget":2,"scored":1}}]}}}}}]},
"runtime":{"model_calls":0,"finite_passed":1,"finite_total":2,"traces":[{"case_index":0,"deliveries":[
{"activity_id":"main","actual":0,"expected":1}]},{"case_index":1,"deliveries":[{"activity_id":"main","actual":1,"expected":1}]}]}}`

func TestSnapshotSourceBudgetNeedsNoSeparatePolicy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		budget     int64
		control    string
		consistent bool
	}{
		{"no policy", 2, `"control":{}`, true},
		{"matching policy", 2, `"control":{"decisions":[{"input":{"budget":2,"scored":1}}]}`, true},
		{"conflicting policy", 2, `"control":{"decisions":[{"input":{"budget":3,"scored":1}}]}`, false},
		{"explicit zero", 0, `"control":{}`, true},
		{"more than candidates", 16, `"control":{}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(constructionFixture, `"record_assembly":{`, fmt.Sprintf(`"record_assembly":{"attempt_budget":%d,`, tc.budget), 1)
			raw = strings.Replace(raw, `"control":{"decisions":[{"input":{"budget":2,"scored":1}}]}`, tc.control, 1)
			s, err := ReadSnapshot([]byte(raw))
			if err != nil || len(s.Construction) != 1 {
				t.Fatal(s, err)
			}
			o := s.Construction[0]
			if !o.BudgetKnown || o.Budget != tc.budget || o.BudgetSource != "source_contract" || o.Consistent != tc.consistent {
				t.Fatal("source budget lost or overwritten by policy", o)
			}
		})
	}
}

func TestSnapshotRetainsConstructionBudgetSeparatelyFromCandidates(t *testing.T) {
	s, err := ReadSnapshot([]byte(constructionFixture))
	if err != nil || len(s.Construction) != 1 {
		t.Fatal(s, err)
	}
	o := s.Construction[0]
	if o.ActivityID != "example/helper" || o.Matched != 2 || o.Total != 3 || o.Scored != 1 || o.Ranked != 4 || o.Budget != 2 || !o.BudgetKnown || !o.Observed || !o.Consistent || o.BudgetSource != "policy_observation" {
		t.Fatal("budget was inferred from ranking length or construction values were lost", o)
	}
	for name, replacement := range map[string]string{
		"budget absent":    `"control":{}`,
		"entry only":       `"control":{"entry":{"input":{"budget":2,"scored":1}}}`,
		"budget disagrees": `"control":{"entry":{"input":{"budget":3,"scored":1}},"decisions":[{"input":{"budget":2,"scored":1}}]}`,
		"future attempt":   `"control":{"decisions":[{"input":{"budget":2,"scored":2}}]}`,
		"missing count":    `"control":{"decisions":[{"input":{"scored":1}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			raw := strings.Replace(constructionFixture, `"control":{"decisions":[{"input":{"budget":2,"scored":1}}]}`, replacement, 1)
			s, err := ReadSnapshot([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			o := s.Construction[0]
			if name == "budget absent" && o.BudgetSource != "unavailable" || name == "entry only" && o.BudgetSource != "policy_observation" {
				t.Fatal("legacy budget source was not retained", o)
			}
			if name == "budget absent" && (o.BudgetKnown || !o.Consistent) || name == "entry only" && (!o.BudgetKnown || !o.Consistent) || name != "budget absent" && name != "entry only" && o.Consistent {
				t.Fatal("budget observation lost its availability or consistency", o)
			}
		})
	}
}

func TestSnapshotConstructionAttemptHistoryKeepsTypeFailures(t *testing.T) {
	raw := strings.Replace(constructionFixture, `"attempts":[{"mask":0}]`, `"attempts":[{"mask":0},{"mask":1,"status":"TYPECHECK_FAILED","reason":"unused variable"}]`, 1)
	s, err := ReadSnapshot([]byte(raw))
	if err != nil || s.Construction[0].Scored != 2 || s.Rejected != 1 || !s.Construction[0].Consistent {
		t.Fatal(s, err)
	}
	for _, replacement := range []string{`"ranking":[0,0,2,3]`, `"ranking":[1,0,2,3]`, `"ranking":[]`} {
		s, err := ReadSnapshot([]byte(strings.Replace(raw, `"ranking":[0,1,2,3]`, replacement, 1)))
		if err != nil || s.Construction[0].Consistent {
			t.Fatal(s, err)
		}
	}
}

func TestNativeSourceBudgetDecisions(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual Gooo/native verification")
	}
	for _, tc := range []struct{ name, budget, control, action string }{
		{"source only", "2", `"control":{}`, "resume-candidates"},
		{"zero budget", "0", `"control":{}`, "replay-construction"},
		{"policy conflict", "2", `"control":{"decisions":[{"input":{"budget":3,"scored":1}}]}`, "replay-construction"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(constructionFixture, `"record_assembly":{`, `"record_assembly":{"attempt_budget":`+tc.budget+`,`, 1)
			raw = strings.Replace(raw, `"control":{"decisions":[{"input":{"budget":2,"scored":1}}]}`, tc.control, 1)
			s, err := ReadSnapshot([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			plans, err := constructionNextSteps(context.Background(), Options{Compiler: compiler}, t.TempDir(), s)
			if err != nil || len(plans) != 1 || plans[0].Action != tc.action || plans[0].Observation.BudgetSource != "source_contract" {
				t.Fatal("Gooo did not use the source budget", plans, err)
			}
		})
	}
}

func TestNativeConstructionNextStepCasesAndIntegration(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual Gooo/native verification")
	}
	root := t.TempDir()
	o := Options{Compiler: compiler}
	raw, _, err := runRecipe(context.Background(), o, root, "next-steps", "finite", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := summarize(raw, "next-steps", "deterministic")
	if err != nil || summary.NamedPassed != 30 || summary.NamedTotal != 30 || summary.FieldsPassed != 195 || summary.FieldsTotal != 195 || summary.ModelCalls != 0 {
		t.Fatal("Gooo next-step finite cases did not all match", summary, err)
	}
	for _, model := range []string{"", "builtin"} {
		t.Run("model="+model, func(t *testing.T) {
			s, err := ReadSnapshot([]byte(constructionFixture))
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "diagnose")
			raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Model: model, Out: out}, s)
			if err != nil {
				t.Fatal(err)
			}
			var diagnosis struct {
				Code  string                 `json:"code"`
				Steps []ConstructionNextStep `json:"construction_next_steps"`
			}
			if err = json.Unmarshal(raw, &diagnosis); err != nil || diagnosis.Code != "partial" || len(diagnosis.Steps) != 1 || diagnosis.Steps[0].Action != "resume-candidates" || diagnosis.Steps[0].Observation.ActivityID != "example/helper" {
				t.Fatal("package diagnostic did not expose Gooo continuation plan", string(raw), err)
			}
			if _, err = os.Stat(filepath.Join(out, "construction-next-steps.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
