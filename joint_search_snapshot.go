package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type jointSearchCandidate struct {
	Schema      string `json:"schema"`
	Activity    string `json:"activity"`
	InputSHA    string `json:"input_source_sha256"`
	SelectedSHA string `json:"selected_source_sha256"`
	PlanSHA     string `json:"plan_sha256"`
	Attempt     struct {
		ID       string   `json:"candidate_id"`
		Error    string   `json:"error"`
		Accuracy *float64 `json:"accuracy_percent"`
		Typed    *bool    `json:"typecheck_passed"`
		Scored   *bool    `json:"scoring_completed"`
		Passed   *int64   `json:"test_cases_passed"`
		Total    *int64   `json:"test_cases_total"`
		Cases    []struct {
			Actual   json.RawMessage `json:"actual"`
			Expected json.RawMessage `json:"expected"`
			Passed   *bool           `json:"passed"`
		} `json:"case_results"`
	} `json:"attempt"`
}

func validateJointKinds(schema string, kinds []string, masks []int, records, searches, fills int) error {
	fillSchema := schema == "gooo/joint-construction/v4" || schema == "gooo/joint-construction/v5"
	if schema == "gooo/joint-construction/v1" {
		if len(kinds) != 0 || searches != 0 || fills != 0 {
			return fmt.Errorf("v1 joint construction cannot contain search candidates")
		}
		return nil
	}
	if len(kinds) < 1 || len(kinds) > 16 || len(masks) != len(kinds) {
		return fmt.Errorf("joint candidate kinds must describe every selector")
	}
	nr, ns, nf := 0, 0, 0
	for i, kind := range kinds {
		if masks[i] < 0 {
			return fmt.Errorf("joint candidate selector must be nonnegative")
		}
		switch kind {
		case "record_mask":
			nr++
		case "source_search_index":
			ns++
			if masks[i] > 15 {
				return fmt.Errorf("joint search selector exceeds the expression limit")
			}
		case "source_fill_index":
			nf++
			if !fillSchema || masks[i] > 15 {
				return fmt.Errorf("joint fill selector requires v4/v5 and a bounded assignment")
			}
		default:
			return fmt.Errorf("unknown joint candidate kind")
		}
	}
	if nr != records || ns != searches || nf != fills ||
		fillSchema && nf < 1 || !fillSchema && ns < 1 {
		return fmt.Errorf("joint candidate kinds disagree with observations")
	}
	return nil
}

func recountJointSearch(candidate jointSearchCandidate) (int64, int64, error) {
	a := candidate.Attempt
	if candidate.Schema != "gooo/search-candidate/v1" || a.Error != "" || a.Typed == nil || !*a.Typed || a.Scored == nil || !*a.Scored || len(a.Cases) == 0 || a.Passed == nil || a.Total == nil {
		return 0, 0, fmt.Errorf("joint search candidate requires typed, scored local cases")
	}
	var passed int64
	for _, row := range a.Cases {
		actual, err := decodeValue(row.Actual)
		if err != nil {
			return 0, 0, err
		}
		expected, err := decodeValue(row.Expected)
		if err != nil {
			return 0, 0, err
		}
		match := reflect.DeepEqual(actual, expected)
		if row.Passed == nil || *row.Passed != match {
			return 0, 0, fmt.Errorf("joint search local flag differs from actual values")
		}
		if match {
			passed++
		}
	}
	total := int64(len(a.Cases))
	if *a.Passed != passed || *a.Total != total {
		return 0, 0, fmt.Errorf("joint search local counts differ from actual values")
	}
	return passed, total, nil
}
