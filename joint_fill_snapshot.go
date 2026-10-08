package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type fillCase struct {
	Actual   json.RawMessage `json:"actual"`
	Expected json.RawMessage `json:"expected"`
	Passed   *bool           `json:"passed"`
}

type jointFillCandidate struct {
	Rejection     *fillRejection  `json:"rejection"`
	Activity      string          `json:"activity"`
	ActivityID    string          `json:"activity_id"`
	InputSHA      string          `json:"input_source_sha256"`
	SelectedSHA   string          `json:"selected_source_sha256"`
	PlanSHA       string          `json:"plan_sha256"`
	Fills         json.RawMessage `json:"hole_fills"`
	Schema        string          `json:"schema"`
	ID            string          `json:"candidate_id"`
	Count         *int            `json:"candidate_count"`
	Passed        *int64          `json:"test_cases_passed"`
	Total         *int64          `json:"test_cases_total"`
	Cases         []fillCase      `json:"case_results"`
	Values        []fillCase      `json:"value_case_results"`
	HoldoutPassed *int64          `json:"holdout_cases_passed"`
	HoldoutTotal  *int64          `json:"holdout_cases_total"`
	Holdout       []fillCase      `json:"holdout_case_results"`
	ValueHoldout  []fillCase      `json:"value_holdout_results"`
}

type fillCount struct{ passed, total int64 }

func recountFillCases(rows []fillCase, passed, total *int64) (fillCount, error) {
	c := fillCount{total: int64(len(rows))}
	for _, row := range rows {
		actual, err := decodeValue(row.Actual)
		if err != nil || actual == nil {
			return c, fmt.Errorf("fill case requires an actual value")
		}
		expected, err := decodeValue(row.Expected)
		if err != nil || expected == nil {
			return c, fmt.Errorf("fill case requires an expectation")
		}
		match := reflect.DeepEqual(actual, expected)
		if row.Passed == nil || *row.Passed != match {
			return c, fmt.Errorf("fill case flag differs from its values")
		}
		if match {
			c.passed++
		}
	}
	if passed == nil || total == nil || *passed != c.passed || *total != c.total {
		return c, fmt.Errorf("fill counts differ from their original values")
	}
	return c, nil
}

func recountJointFill(c jointFillCandidate) (fillCount, fillCount, error) {
	var zero fillCount
	if c.Rejection != nil || c.Schema != "gooo/fill-candidate/v1" || c.ID == "" || c.Count == nil || *c.Count < 2 || *c.Count > 16 ||
		(len(c.Cases) == 0) == (len(c.Values) == 0) || len(c.Cases) > 0 && len(c.ValueHoldout) > 0 || len(c.Values) > 0 && len(c.Holdout) > 0 {
		return zero, zero, fmt.Errorf("fill assignment requires one local case format and 2..16 candidates")
	}
	rows, holdouts := c.Cases, c.Holdout
	if len(c.Values) > 0 {
		rows, holdouts = c.Values, c.ValueHoldout
	}
	local, err := recountFillCases(rows, c.Passed, c.Total)
	if err != nil {
		return zero, zero, err
	}
	holdout, err := recountFillCases(holdouts, c.HoldoutPassed, c.HoldoutTotal)
	return local, holdout, err
}
