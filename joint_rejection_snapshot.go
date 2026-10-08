package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
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
	records int, searches []jointSearchCandidate, fills []jointFillCandidate, runtime json.RawMessage) error {
	kind := "source_search_index"
	if rejected.Stage == "LOCAL_SOURCE_FILL" && schema == "gooo/joint-construction/v5" {
		kind = "source_fill_index"
	}
	if schema != "gooo/joint-construction/v3" && schema != "gooo/joint-construction/v4" && schema != "gooo/joint-construction/v5" ||
		(rejected.Stage != "LOCAL_SOURCE_SEARCH" && kind != "source_fill_index") ||
		rejected.Slot == nil || *rejected.Slot < 0 || *rejected.Slot >= len(kinds) || kinds[*rejected.Slot] != kind {
		return fmt.Errorf("joint rejection requires a bound search slot or v5 fill slot")
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
	if records != prefixRecords || len(searches) != prefixSearches || len(fills) != prefixFills {
		return fmt.Errorf("joint rejected observations differ from the evaluated prefix")
	}
	if kind == "source_fill_index" {
		if len(fills) == 0 {
			return fmt.Errorf("missing rejected fill")
		}
		if err := validateRejectedFill(fills[len(fills)-1], rejected); err != nil {
			return err
		}
	} else {
		if len(searches) == 0 {
			return fmt.Errorf("missing rejected search")
		}
		if err := validateRejectedSearch(searches[len(searches)-1], rejected); err != nil {
			return err
		}
	}
	value, err := decodeValue(runtime)
	if err != nil || !zeroJointObservation(value) {
		return fmt.Errorf("joint rejected expression cannot claim native observations")
	}
	return nil
}

func validateRejectedSearch(candidate jointSearchCandidate, rejected *JointRejectionObservation) error {
	a := candidate.Attempt
	if candidate.Schema != "gooo/search-candidate/v1" || candidate.Activity == "" || candidate.Activity != rejected.Activity ||
		a.ID == "" || a.ID != rejected.CandidateID || a.Error == "" || a.Error != rejected.Reason ||
		candidate.InputSHA == "" || candidate.PlanSHA == "" || candidate.SelectedSHA != "" ||
		a.Typed == nil || a.Scored == nil || *a.Scored || a.Accuracy != nil || a.Passed == nil || *a.Passed != 0 ||
		a.Total == nil || *a.Total < 1 || len(a.Cases) != 0 {
		return fmt.Errorf("joint rejected expression claims a score or lacks its failure identity")
	}
	return nil
}

type fillRejection struct {
	ID     string          `json:"candidate_id"`
	Stage  string          `json:"stage"`
	Reason string          `json:"reason"`
	Fills  json.RawMessage `json:"hole_fills"`
}

func validFillRejection(r *fillRejection) bool {
	if r == nil || r.ID == "" || r.Reason == "" || r.Stage != "TYPECHECK" && r.Stage != "TRAINING_EVALUATION" {
		return false
	}
	var fills []struct {
		HoleID     string `json:"hole_id"`
		Expression string `json:"expression"`
	}
	if json.Unmarshal(r.Fills, &fills) != nil || len(fills) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, f := range fills {
		if f.HoleID == "" || f.Expression == "" || seen[f.HoleID] {
			return false
		}
		seen[f.HoleID] = true
	}
	return true
}

func validateRejectedFill(c jointFillCandidate, r *JointRejectionObservation) error {
	zero := func(n *int64) bool { return n != nil && *n == 0 }
	a, errA := decodeValue(c.Fills)
	var b any
	var errB error
	if c.Rejection != nil {
		b, errB = decodeValue(c.Rejection.Fills)
	}
	if c.Schema != "gooo/fill-candidate/v1" || !validFillRejection(c.Rejection) || c.ID != r.CandidateID || c.ID != c.Rejection.ID ||
		c.Activity == "" || c.Activity != r.Activity || c.ActivityID == "" || c.Rejection.Reason != r.Reason ||
		c.InputSHA == "" || c.SelectedSHA == "" || c.PlanSHA == "" || c.Count == nil || *c.Count < 2 || *c.Count > 16 ||
		!zero(c.Passed) || !zero(c.Total) || !zero(c.HoldoutPassed) || !zero(c.HoldoutTotal) ||
		len(c.Cases)+len(c.Values)+len(c.Holdout)+len(c.ValueHoldout) != 0 || errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
		return fmt.Errorf("joint rejected fill claims a score or lacks its assignment identity")
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
