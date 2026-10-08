package workbench

type constructionFill struct {
	Selected      string     `json:"selected_candidate_id"`
	Passed        *int64     `json:"test_cases_passed"`
	Total         *int64     `json:"test_cases_total"`
	Cases         []fillCase `json:"selected_case_results"`
	Values        []fillCase `json:"selected_value_case_results"`
	HoldoutPassed *int64     `json:"holdout_cases_passed"`
	HoldoutTotal  *int64     `json:"holdout_cases_total"`
	Holdout       []fillCase `json:"holdout_case_results"`
	ValueHoldout  []fillCase `json:"selected_value_holdout_case_results"`
	Scores        []struct {
		ID     string `json:"id"`
		Typed  *bool  `json:"typecheck_passed"`
		Passed *int64 `json:"test_cases_passed"`
		Total  *int64 `json:"test_cases_total"`
	} `json:"candidate_scores"`
	Generation *struct {
		Omitted *int64 `json:"assignments_omitted"`
	} `json:"candidate_generation"`
}

func fillObservation(activity string, f *constructionFill) ConstructionObservation {
	n := len(f.Scores)
	o := ConstructionObservation{ActivityID: activity, Kind: "source_fill", Scored: int64(n), Ranked: int64(n),
		Budget: int64(n), BudgetKnown: true, BudgetSource: "source_contract", SpaceKnown: true, Consistent: activity != ""}
	candidate := jointFillCandidate{Schema: "gooo/fill-candidate/v1", ID: f.Selected, Count: &n,
		Passed: f.Passed, Total: f.Total, Cases: f.Cases, Values: f.Values, HoldoutPassed: f.HoldoutPassed,
		HoldoutTotal: f.HoldoutTotal, Holdout: f.Holdout, ValueHoldout: f.ValueHoldout}
	local, _, err := recountJointFill(candidate)
	o.Observed = err == nil
	o.Matched, o.Total = local.passed, local.total
	if err != nil {
		o.Consistent = false
	}
	seen, selected := map[string]bool{}, false
	for _, score := range f.Scores {
		if score.ID == "" || seen[score.ID] || score.Typed == nil || !*score.Typed || score.Passed == nil || score.Total == nil ||
			*score.Passed < 0 || *score.Passed > *score.Total || *score.Total != o.Total {
			o.Consistent = false
		}
		seen[score.ID] = true
		if score.ID == f.Selected {
			selected = true
			if score.Passed == nil || *score.Passed != o.Matched {
				o.Consistent = false
			}
		}
	}
	if !selected {
		o.Consistent = false
	}
	if f.Generation != nil {
		if f.Generation.Omitted == nil || *f.Generation.Omitted < 0 {
			o.Consistent, o.SpaceKnown = false, false
		} else {
			o.Omitted = *f.Generation.Omitted
		}
	}
	return o
}
