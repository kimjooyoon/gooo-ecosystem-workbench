package workbench

import (
	"encoding/json"
	"strings"
	"testing"
)

const searchObservationFixture = `{"schema":"gooo/body-codegen-ir-search-plan/v1",
"attempt_budget":1,"training_passed":0,"training_total":2,
"candidate_count":2,"attempted_candidates":1,"evaluated_candidates":1,"untested_candidates":1,
"attempts":[{"candidate_id":"input","typecheck_passed":true,"scoring_completed":true}],
"candidate_generation":{"candidates_enumerated":5,"candidates_retained":2,"candidates_omitted":3}}`

func TestSearchObservationSeparatesSourceBudgetRetainedSpaceAndOutcomes(t *testing.T) {
	var search constructionSearch
	if err := json.Unmarshal([]byte(searchObservationFixture), &search); err != nil {
		t.Fatal(err)
	}
	o := searchObservation("offset://activity/add", &search)
	if !o.Consistent || !o.Observed || !o.BudgetKnown || !o.SpaceKnown || o.Budget != 1 || o.Scored != 1 || o.Ranked != 2 || o.Omitted != 3 || o.Matched != 0 || o.Total != 2 || o.BudgetSource != "search_plan" {
		t.Fatal("search observations were collapsed", o)
	}
	for _, tc := range []struct {
		old, next                   string
		consistent, known, observed bool
	}{
		{`"attempt_budget":1,`, "", true, false, true},
		{`"training_passed":0,`, "", true, true, false},
		{`"attempted_candidates":1`, `"attempted_candidates":0`, false, true, true},
		{`"evaluated_candidates":1`, `"evaluated_candidates":0`, false, true, true},
		{`"untested_candidates":1`, `"untested_candidates":2`, false, true, true},
		{`"candidates_omitted":3`, `"candidates_omitted":0`, false, true, true},
		{`"typecheck_passed":true`, `"typecheck_passed":false`, false, true, true},
		{`"candidate_id":"input"`, `"candidate_id":""`, false, true, true},
	} {
		var modified constructionSearch
		if err := json.Unmarshal([]byte(strings.Replace(searchObservationFixture, tc.old, tc.next, 1)), &modified); err != nil {
			t.Fatal(err)
		}
		o := searchObservation("offset://activity/add", &modified)
		if o.Consistent != tc.consistent || o.BudgetKnown != tc.known || o.Observed != tc.observed {
			t.Fatal("missing or inconsistent search field was inferred", tc.old, o)
		}
	}
}

func TestSnapshotIncludesSearchTypeRejectionsAndPreparations(t *testing.T) {
	var fixture map[string]any
	if err := json.Unmarshal([]byte(constructionFixture), &fixture); err != nil {
		t.Fatal(err)
	}
	step := fixture["composition"].(map[string]any)["preparations"].([]any)[0].(map[string]any)
	report := step["generation"].(map[string]any)["report"].(map[string]any)
	delete(report, "record_assembly")
	raw := strings.Replace(searchObservationFixture, `"evaluated_candidates":1`, `"evaluated_candidates":0`, 1)
	raw = strings.Replace(raw, `"typecheck_passed":true,"scoring_completed":true`, `"typecheck_passed":false,"scoring_completed":false,"error":"unbound variable"`, 1)
	report["body_search"] = json.RawMessage(raw)
	encoded, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadSnapshot(encoded)
	if err != nil || len(s.Construction) != 1 || s.Construction[0].Kind != "integer_search" || s.Rejected != 1 || !strings.Contains(s.Detail, "unbound variable") || !s.Construction[0].Consistent {
		t.Fatal("search preparation or rejected attempt disappeared", err, s)
	}
}
