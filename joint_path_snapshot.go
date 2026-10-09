package workbench

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

type pathLocalCase struct {
	Input, Actual, Expected *int64
	Passed                  *bool
}

type jointPathCandidate struct {
	Schema, Activity string
	ActivityID       string            `json:"activity_id"`
	InputSHA         string            `json:"input_source_sha256"`
	SelectedSHA      string            `json:"selected_source_sha256"`
	DocumentSHA      string            `json:"document_sha256"`
	PlanSHA          string            `json:"plan_sha256"`
	Mask             *int              `json:"mask"`
	Choices          map[string]string `json:"choices"`
	Set              struct {
		DocumentSHA string `json:"document_sha256"`
		PlanSHA     string `json:"plan_sha256"`
		Budget      *int   `json:"attempt_budget"`
		Declared    *int   `json:"declared_combinations"`
		Masks       []int  `json:"masks"`
		Order       string `json:"order"`
	} `json:"candidate_set"`
	Stage, Failure string
	Cases          []pathLocalCase `json:"case_results"`
	Passed         *int64          `json:"local_passed"`
	Total          *int64          `json:"local_total"`
}

func pathDigest(value string, prefix bool) bool {
	if prefix {
		if !strings.HasPrefix(value, "sha256:") {
			return false
		}
		value = strings.TrimPrefix(value, "sha256:")
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == strings.ToLower(value)
}

func validatePathIdentity(c jointPathCandidate) error {
	s := c.Set
	if c.Schema != "gooo/path-candidate/v1" || c.Activity == "" || c.ActivityID == "" ||
		!pathDigest(c.InputSHA, true) || !pathDigest(c.DocumentSHA, true) || !pathDigest(c.PlanSHA, false) ||
		s.DocumentSHA != c.DocumentSHA || s.PlanSHA != c.PlanSHA || c.Mask == nil ||
		len(c.Choices) < 1 || len(c.Choices) > 16 || s.Declared == nil || *s.Declared != 1<<len(c.Choices) ||
		s.Budget == nil || *s.Budget < 1 || *s.Budget > 64 || len(s.Masks) != min(*s.Budget, *s.Declared) ||
		s.Order != "initial_selection_then_fallback_distance_then_numeric_mask" || !slices.Contains(s.Masks, *c.Mask) {
		return fmt.Errorf("typed path candidate lacks its bounded source palette")
	}
	seen := make(map[int]bool, len(s.Masks))
	for _, mask := range s.Masks {
		if mask < 0 || mask >= *s.Declared || seen[mask] {
			return fmt.Errorf("typed path palette has an invalid mask")
		}
		seen[mask] = true
	}
	for id, label := range c.Choices {
		if id == "" || label == "" {
			return fmt.Errorf("typed path choice identity is empty")
		}
	}
	return nil
}

func recountJointPath(c jointPathCandidate) (int64, int64, error) {
	if err := validatePathIdentity(c); err != nil {
		return 0, 0, err
	}
	if c.Stage != "COMPLETE" || c.Failure != "" || !pathDigest(c.SelectedSHA, true) ||
		c.Passed == nil || c.Total == nil || len(c.Cases) == 0 {
		return 0, 0, fmt.Errorf("typed path candidate requires complete local cases")
	}
	passed, total, err := recountPathCases(c.Cases)
	if err != nil {
		return 0, 0, err
	}
	if *c.Passed != passed || *c.Total != total {
		return 0, 0, fmt.Errorf("typed path local counts differ from cases")
	}
	return passed, total, nil
}

func recountPathCases(cases []pathLocalCase) (int64, int64, error) {
	var passed int64
	for _, row := range cases {
		if row.Input == nil || row.Actual == nil || row.Expected == nil || row.Passed == nil || *row.Passed != (*row.Actual == *row.Expected) {
			return 0, 0, fmt.Errorf("typed path local flag differs from exact integer values")
		}
		if *row.Passed {
			passed++
		}
	}
	return passed, int64(len(cases)), nil
}

func validateRejectedPath(c jointPathCandidate, r *JointRejectionObservation) error {
	if err := validatePathIdentity(c); err != nil {
		return err
	}
	if c.Activity != r.Activity || r.CandidateID != fmt.Sprintf("path_%04x", *c.Mask) || c.Failure == "" ||
		c.Failure != r.Reason || c.Stage != "TYPE_CHECK" && c.Stage != "LOCAL_CASES" || c.SelectedSHA != "" ||
		len(c.Cases) != 0 || c.Passed == nil || *c.Passed != 0 || c.Total == nil || *c.Total != 0 {
		return fmt.Errorf("rejected typed path claims a score or lacks its failure identity")
	}
	return nil
}

func validatePathSelectors(kinds []string, masks []int, candidates []jointPathCandidate, rejected *JointRejectionObservation) error {
	n := 0
	for i, kind := range kinds {
		if rejected != nil && rejected.Slot != nil && i > *rejected.Slot {
			break
		}
		if kind != "typed_path_mask" {
			continue
		}
		if i >= len(masks) || n >= len(candidates) || candidates[n].Mask == nil || *candidates[n].Mask != masks[i] {
			return fmt.Errorf("typed path selector differs from its candidate")
		}
		n++
	}
	if n != len(candidates) {
		return fmt.Errorf("typed path candidate count differs from selectors")
	}
	return nil
}
