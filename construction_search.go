package workbench

// These are recorded observations; the Gooo recipe selects the next action.
type constructionSearch struct {
	Schema         string `json:"schema"`
	AttemptBudget  *int64 `json:"attempt_budget"`
	Passed         *int64 `json:"training_passed"`
	Total          *int64 `json:"training_total"`
	CandidateCount *int64 `json:"candidate_count"`
	Attempted      *int64 `json:"attempted_candidates"`
	Evaluated      *int64 `json:"evaluated_candidates"`
	Untested       *int64 `json:"untested_candidates"`
	Attempts       []struct {
		ID               string `json:"candidate_id"`
		TypecheckPassed  bool   `json:"typecheck_passed"`
		ScoringCompleted bool   `json:"scoring_completed"`
		Error            string `json:"error"`
	} `json:"attempts"`
	Generation *struct {
		Enumerated *int64 `json:"candidates_enumerated"`
		Retained   *int64 `json:"candidates_retained"`
		Omitted    *int64 `json:"candidates_omitted"`
	} `json:"candidate_generation"`
}

func searchObservation(activityID string, s *constructionSearch) ConstructionObservation {
	o := ConstructionObservation{ActivityID: activityID, Kind: "integer_search", Scored: int64(len(s.Attempts)),
		BudgetSource: "unavailable", Consistent: activityID != "" && s.Schema == "gooo/body-codegen-ir-search-plan/v1"}
	if s.AttemptBudget != nil {
		o.Budget, o.BudgetKnown, o.BudgetSource = *s.AttemptBudget, true, "search_plan"
	}
	o.Observed = s.Passed != nil && s.Total != nil
	if o.Observed {
		o.Matched, o.Total = *s.Passed, *s.Total
	}
	if s.CandidateCount != nil {
		o.Ranked = *s.CandidateCount
	}
	seen, evaluated := map[string]bool{}, int64(0)
	for _, attempt := range s.Attempts {
		if attempt.ID == "" || seen[attempt.ID] || attempt.ScoringCompleted && !attempt.TypecheckPassed {
			o.Consistent = false
		}
		seen[attempt.ID] = true
		if attempt.ScoringCompleted {
			evaluated++
		}
	}
	if s.CandidateCount == nil || s.Attempted == nil || s.Evaluated == nil || s.Untested == nil {
		o.Consistent = false
	} else if *s.Attempted != o.Scored || *s.Evaluated != evaluated || *s.Untested != o.Ranked-o.Scored {
		o.Consistent = false
	}
	if g := s.Generation; g != nil && g.Enumerated != nil && g.Retained != nil && g.Omitted != nil {
		o.Omitted, o.SpaceKnown = *g.Omitted, true
		if *g.Omitted < 0 || *g.Retained != o.Ranked || *g.Enumerated < *g.Retained || *g.Enumerated-*g.Retained != *g.Omitted {
			o.Consistent = false
		}
	}
	return o
}
