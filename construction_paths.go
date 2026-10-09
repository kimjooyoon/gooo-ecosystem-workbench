package workbench

import "encoding/json"

// The initial receipt records executed local cases, but does not expose its
// source attempt cap. Keep that budget unknown rather than infer it from space.
func pathObservation(activityID string, raw json.RawMessage) ConstructionObservation {
	o := ConstructionObservation{ActivityID: activityID, Kind: "typed_paths", BudgetSource: "unavailable"}
	var p struct {
		Schema string
		Cases  []pathLocalCase `json:"native_case_results"`
		Search struct {
			Schema    string
			Passed    *int64                    `json:"selected_training_passed"`
			Total     *int64                    `json:"training_cases"`
			Declared  *int64                    `json:"declared_combinations"`
			Untested  *int64                    `json:"unattempted_combinations"`
			Evaluated *int64                    `json:"evaluated_candidates"`
			Rejected  *int64                    `json:"type_rejected_candidates"`
			Attempts  []struct{ Status string } `json:"attempts"`
		} `json:"search"`
	}
	if json.Unmarshal(raw, &p) != nil || p.Schema != "gooo/body-codegen-typed-path-receipt/v1" ||
		p.Search.Schema != "gooo/typed-path-tdd-search/v1" || activityID == "" {
		return o
	}
	s := p.Search
	if s.Passed == nil || s.Total == nil || s.Declared == nil || s.Untested == nil || s.Evaluated == nil || s.Rejected == nil {
		return o
	}
	passed, total, err := recountPathCases(p.Cases)
	o.Observed, o.Matched, o.Total = true, passed, total
	o.Ranked, o.Scored, o.Rejected = *s.Declared, *s.Evaluated, *s.Rejected
	o.SpaceKnown = true
	evaluated, rejected := int64(0), int64(0)
	for _, a := range s.Attempts {
		switch a.Status {
		case "EVALUATED":
			evaluated++
		case "TYPE_REJECTED":
			rejected++
		default:
			return o
		}
	}
	o.Consistent = err == nil && total > 0 && *s.Passed == passed && *s.Total == total &&
		evaluated == *s.Evaluated && rejected == *s.Rejected && *s.Declared > 0 && *s.Declared <= 65536 &&
		*s.Untested >= 0 && *s.Declared == int64(len(s.Attempts))+*s.Untested
	return o
}
