package workbench

import (
	"encoding/json"
	"testing"
)

func TestJointTypedPathSnapshotOriginalNative(t *testing.T) {
	for _, tc := range []struct {
		name            string
		attempts, local int64
		replay          bool
	}{
		{"typed-fixed", 2, 1, false}, {"typed-fixed-replay", 2, 1, true},
		{"typed-mixed", 15, 2, false}, {"typed-mixed-replay", 15, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ReadSnapshot(jointFixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			j := s.Joint
			if j == nil || j.ProgramAttempts != tc.attempts || j.LocalPassed != tc.local || j.LocalTotal != tc.local ||
				j.Replayed != tc.replay || j.NewModelCalls != 0 || j.CallerPassed != 1 || j.CallerTotal != 1 ||
				s.Passed != 3 || s.Total != 3 || *j.Inputs.Other != 3 || j.NativeFaultAttempts != 0 {
				t.Fatal("typed path stages were mixed", s)
			}
			found := false
			for _, o := range j.Initial {
				if o.Kind == "typed_paths" {
					found = true
					if !o.Consistent || o.Matched != 1 || o.Total != 1 || o.Scored != 1 || o.Ranked != 2 || o.BudgetKnown {
						t.Fatal("historical local preparation or unknown source cap lost", o)
					}
				}
			}
			if !found {
				t.Fatal("missing initial typed preparation")
			}
		})
	}
}

func TestJointTypedPathsKeepRejectedAndNativeFaultStages(t *testing.T) {
	for _, tc := range []struct {
		name                                            string
		attempts, rejected, nativeFaults, local, caller int64
	}{
		{"typed-rejection", 4, 1, 0, 1, 1}, {"typed-fault", 2, 0, 1, 1, 0},
	} {
		s, err := ReadSnapshot(jointFixture(t, tc.name))
		if err != nil {
			t.Fatal(tc.name, err)
		}
		j := s.Joint
		if j == nil || j.ProgramAttempts != tc.attempts || j.RejectedAttempts != tc.rejected ||
			j.NativeFaultAttempts != tc.nativeFaults || j.LocalTotal != tc.local || j.CallerPassed != tc.caller || j.Decision != "PARTIAL_FINITE" {
			t.Fatal(tc.name, s)
		}
		if tc.rejected > 0 {
			last := j.History[len(j.History)-1]
			if last.Rejection == nil || last.CallerTotal != 0 || last.LocalTotal != 0 || last.NativeOutcomes != nil {
				t.Fatal("rejected typed combination acquired an execution score", last)
			}
		}
	}
}

func TestJointRejectedTypedPathBindsMixedPrefix(t *testing.T) {
	var path, search, fill jointReceipt
	for _, row := range []struct {
		target *jointReceipt
		name   string
	}{
		{&path, "typed-rejection"}, {&search, "mixed-search"}, {&fill, "../caller-source-fill/mixed-fixed"},
	} {
		if err := json.Unmarshal(jointFixture(t, row.name), row.target); err != nil {
			t.Fatal(err)
		}
	}
	a := path.Construction.Attempts[3]
	r := *a.Rejection
	slot := 3
	r.Slot = &slot
	// Synthetic transport check: an existing record/search/fill prefix precedes
	// the actual rejected path. It does not claim a new native program execution.
	kinds := []string{"record_mask", "source_search_index", "source_fill_index", "typed_path_mask", "source_fill_index"}
	masks := []int{0, 0, 0, 3, 0}
	searches := search.Construction.Attempts[0].SearchCandidates
	fills := fill.Construction.Attempts[0].FillCandidates
	if len(searches) != 1 || len(fills) != 1 {
		t.Fatal("fixture prefix changed")
	}
	check := func(records int, s []jointSearchCandidate, f []jointFillCandidate) error {
		return validateJointRejectionWithPaths("gooo/joint-construction/v7", kinds, masks, &r, records, s, f, a.PathCandidates, a.Runtime)
	}
	if err := check(1, searches, fills); err != nil {
		t.Fatal(err)
	}
	if check(0, searches, fills) == nil || check(1, nil, fills) == nil || check(1, searches, append(fills, fills[0])) == nil {
		t.Fatal("prefix omission or unevaluated suffix accepted")
	}
	changed := a.PathCandidates[0]
	changed.Total = new(int64)
	*changed.Total = 1
	if validateJointRejectionWithPaths("gooo/joint-construction/v7", kinds, masks, &r, 1, searches, fills, []jointPathCandidate{changed}, a.Runtime) == nil {
		t.Fatal("rejected local score accepted")
	}
}

func TestJointTypedInitialCountsAreRecounted(t *testing.T) {
	v, err := decodeValue(jointFixture(t, "typed-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	c := v.(map[string]any)["construction"].(map[string]any)
	p := c["initial"].(map[string]any)["preparations"].([]any)[0].(map[string]any)["generation"].(map[string]any)["report"].(map[string]any)["body_paths"].(map[string]any)
	p["native_case_results"].([]any)[0].(map[string]any)["actual"] = json.Number("9007199254740993")
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReadSnapshot(raw); err == nil {
		t.Fatal("invented initial local count accepted")
	}
}

func TestJointTypedPathSnapshotRejectsInconsistentRows(t *testing.T) {
	for name, change := range map[string]func(map[string]any, map[string]any, map[string]any){
		"schema downgrade":        func(c, a, p map[string]any) { c["schema"] = "gooo/joint-construction/v6" },
		"missing paths":           func(c, a, p map[string]any) { delete(a, "path_candidates") },
		"selector mismatch":       func(c, a, p map[string]any) { p["mask"] = 1 },
		"outside palette":         func(c, a, p map[string]any) { p["mask"] = 2 },
		"duplicate masks":         func(c, a, p map[string]any) { p["candidate_set"].(map[string]any)["masks"] = []int{0, 0} },
		"document binding":        func(c, a, p map[string]any) { p["candidate_set"].(map[string]any)["document_sha256"] = "different" },
		"plan binding":            func(c, a, p map[string]any) { p["candidate_set"].(map[string]any)["plan_sha256"] = "different" },
		"missing input digest":    func(c, a, p map[string]any) { delete(p, "input_source_sha256") },
		"missing selected digest": func(c, a, p map[string]any) { delete(p, "selected_source_sha256") },
		"empty choice":            func(c, a, p map[string]any) { p["choices"] = map[string]string{"sign": ""} },
		"space count":             func(c, a, p map[string]any) { p["candidate_set"].(map[string]any)["declared_combinations"] = 4 },
		"budget count":            func(c, a, p map[string]any) { p["candidate_set"].(map[string]any)["attempt_budget"] = 1 },
		"wrong order":             func(c, a, p map[string]any) { p["candidate_set"].(map[string]any)["order"] = "random" },
		"local counts":            func(c, a, p map[string]any) { p["local_passed"] = 0 },
		"case flag":               func(c, a, p map[string]any) { p["case_results"].([]any)[0].(map[string]any)["passed"] = false },
		"exact integer": func(c, a, p map[string]any) {
			p["case_results"].([]any)[0].(map[string]any)["actual"] = json.Number("9007199254740993")
		},
		"noninteger": func(c, a, p map[string]any) {
			p["case_results"].([]any)[0].(map[string]any)["input"] = json.Number("0.5")
		},
	} {
		t.Run(name, func(t *testing.T) {
			value, err := decodeValue(jointFixture(t, "typed-fixed"))
			if err != nil {
				t.Fatal(err)
			}
			c := value.(map[string]any)["construction"].(map[string]any)
			a := c["attempts"].([]any)[0].(map[string]any)
			p := a["path_candidates"].([]any)[0].(map[string]any)
			change(c, a, p)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("inconsistent typed observation accepted")
			}
		})
	}
}
