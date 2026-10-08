package workbench

import (
	"encoding/json"
	"fmt"
)

// Rejected attempts have no caller score. Their local totals cover only the
// already scored prefix, not every body in the proposed combination.
type JointRejectionObservation struct {
	Stage       string `json:"stage"`
	Slot        *int   `json:"slot"`
	Activity    string `json:"activity"`
	CandidateID string `json:"candidate_id"`
	Reason      string `json:"reason"`
}

func validateJointRejection(schema string, kinds []string, masks []int, rejected *JointRejectionObservation,
	records int, searches []jointSearchCandidate, fills int, runtime json.RawMessage) error {
	if schema != "gooo/joint-construction/v3" && schema != "gooo/joint-construction/v4" || rejected.Stage != "LOCAL_SOURCE_SEARCH" ||
		rejected.Slot == nil || *rejected.Slot < 0 || *rejected.Slot >= len(kinds) || kinds[*rejected.Slot] != "source_search_index" {
		return fmt.Errorf("joint rejection requires a v3/v4 source-search slot")
	}
	allRecords, allSearches, prefixRecords, prefixSearches := 0, 0, 0, 0
	allFills, prefixFills := 0, 0
	for i, kind := range kinds {
		if kind == "record_mask" {
			allRecords++
			if i <= *rejected.Slot {
				prefixRecords++
			}
		} else if kind == "source_search_index" {
			allSearches++
			if i <= *rejected.Slot {
				prefixSearches++
			}
		} else if kind == "source_fill_index" {
			allFills++
			if i <= *rejected.Slot {
				prefixFills++
			}
		}
	}
	if err := validateJointKinds(schema, kinds, masks, allRecords, allSearches, allFills); err != nil {
		return err
	}
	if records != prefixRecords || len(searches) != prefixSearches || len(searches) == 0 || fills != prefixFills {
		return fmt.Errorf("joint rejected observations differ from the evaluated prefix")
	}
	candidate := searches[len(searches)-1]
	a := candidate.Attempt
	if candidate.Schema != "gooo/search-candidate/v1" || candidate.Activity == "" || candidate.Activity != rejected.Activity ||
		a.ID == "" || a.ID != rejected.CandidateID || a.Error == "" || a.Error != rejected.Reason ||
		candidate.InputSHA == "" || candidate.PlanSHA == "" || candidate.SelectedSHA != "" ||
		a.Typed == nil || a.Scored == nil || *a.Scored || a.Accuracy != nil || a.Passed == nil || *a.Passed != 0 ||
		a.Total == nil || *a.Total < 1 || len(a.Cases) != 0 {
		return fmt.Errorf("joint rejected expression claims a score or lacks its failure identity")
	}
	value, err := decodeValue(runtime)
	if err != nil || !zeroJointObservation(value) {
		return fmt.Errorf("joint rejected expression cannot claim native observations")
	}
	return nil
}

func zeroJointObservation(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case string:
		return v == ""
	case json.Number:
		return v.String() == "0"
	case []any:
		return len(v) == 0
	case map[string]any:
		for _, field := range v {
			if !zeroJointObservation(field) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
