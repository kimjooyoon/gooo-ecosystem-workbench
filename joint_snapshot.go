package workbench

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"slices"
)

// JointObservation separates historical local preparation, whole-program search,
// and subsequent evaluation. Counts describe supplied finite observations.
type JointObservation struct {
	Initial               []ConstructionObservation `json:"initial_local_construction"`
	History               []JointAttemptObservation `json:"program_history"`
	ProgramAttempts       int64                     `json:"program_attempts"`
	RejectedAttempts      int64                     `json:"rejected_attempts"`
	NativeProgramAttempts int64                     `json:"native_program_attempts"`
	NativeFaultAttempts   int64                     `json:"native_fault_attempts"`
	NativeFaults          int64                     `json:"selected_faulted_activities"`
	BlockedActivities     int64                     `json:"selected_blocked_activities"`
	ProgramBudget         int64                     `json:"program_budget"`
	CandidateSpace        string                    `json:"candidate_space"`
	CandidateKinds        []string                  `json:"candidate_kinds,omitempty"`
	MoreCandidates        bool                      `json:"more_candidates"`
	SelectedAttempt       int                       `json:"selected_attempt"`
	LocalPassed           int64                     `json:"local_passed"`
	LocalTotal            int64                     `json:"local_total"`
	FillHoldoutPassed     int64                     `json:"fill_holdout_passed"`
	FillHoldoutTotal      int64                     `json:"fill_holdout_total"`
	CallerPassed          int64                     `json:"caller_passed"`
	CallerTotal           int64                     `json:"caller_total"`
	Decision              string                    `json:"decision"`
	StopReason            string                    `json:"stop_reason"`
	Replayed              bool                      `json:"construction_replayed"`
	NewModelCalls         int64                     `json:"new_model_calls"`
	Inputs                jointInputs               `json:"reported_evaluation_inputs"`
}

type JointAttemptObservation struct {
	NativeOutcomes    *NativeOutcomes            `json:"native_outcomes,omitempty"`
	Rejection         *JointRejectionObservation `json:"rejection,omitempty"`
	LocalPassed       int64                      `json:"local_passed"`
	LocalTotal        int64                      `json:"local_total"`
	FillHoldoutPassed int64                      `json:"fill_holdout_passed"`
	FillHoldoutTotal  int64                      `json:"fill_holdout_total"`
	CallerPassed      int64                      `json:"caller_passed"`
	CallerTotal       int64                      `json:"caller_total"`
}

type jointInputs struct {
	Unique     *int64 `json:"unique_inputs"`
	Duplicates *int64 `json:"duplicate_rows"`
	Consumed   *int64 `json:"construction_inputs"`
	Other      *int64 `json:"other_inputs"`
	Scope      string `json:"scope"`
}

type jointReceipt struct {
	Generated    *bool `json:"generated_now"`
	Construction struct {
		Schema   string          `json:"schema"`
		Stage    string          `json:"stage"`
		Failure  string          `json:"failure"`
		Initial  json.RawMessage `json:"initial"`
		Budget   *int64          `json:"program_budget"`
		Space    string          `json:"candidate_space"`
		Kinds    []string        `json:"candidate_kinds"`
		Selected *int            `json:"selected_attempt"`
		Decision string          `json:"decision"`
		Stop     string          `json:"stop_reason"`
		Attempts []struct {
			Rejection   *JointRejectionObservation `json:"rejection"`
			Masks       []int                      `json:"masks"`
			LocalPassed *int64                     `json:"local_passed"`
			LocalTotal  *int64                     `json:"local_total"`
			Runtime     json.RawMessage            `json:"runtime"`
			Candidates  []struct {
				Attempt struct {
					Passed *int64 `json:"passed"`
					Total  *int64 `json:"total"`
				} `json:"attempt"`
				Cases []struct {
					Actual   json.RawMessage `json:"actual"`
					Expected json.RawMessage `json:"expected"`
					Passed   *bool           `json:"passed"`
				} `json:"cases"`
			} `json:"candidates"`
			SearchCandidates []jointSearchCandidate `json:"search_candidates"`
			FillCandidates   []jointFillCandidate   `json:"fill_candidates"`
		} `json:"attempts"`
	} `json:"construction"`
	Evaluation struct {
		Runtime  json.RawMessage `json:"runtime"`
		Replayed *bool           `json:"construction_replayed"`
		Calls    *int64          `json:"new_model_calls"`
		Inputs   jointInputs     `json:"input_separation"`
	} `json:"evaluation"`
}

func readJointSnapshot(raw []byte, inputSHA string) (Snapshot, error) {
	var r jointReceipt
	if err := json.Unmarshal(raw, &r); err != nil {
		return Snapshot{}, err
	}
	c, e := r.Construction, r.Evaluation
	if !slices.Contains([]string{"gooo/joint-construction/v1", "gooo/joint-construction/v2", "gooo/joint-construction/v3", "gooo/joint-construction/v4", "gooo/joint-construction/v5", "gooo/joint-construction/v6"}, c.Schema) || c.Stage != "COMPLETE" || c.Failure != "" ||
		c.Budget == nil || *c.Budget < 1 || *c.Budget > 64 || c.Selected == nil || *c.Selected < 0 || *c.Selected >= len(c.Attempts) ||
		r.Generated == nil || e.Replayed == nil || *r.Generated == *e.Replayed || e.Calls == nil || *e.Calls != 0 {
		return Snapshot{}, fmt.Errorf("joint construction requires complete finite observations, a selected program and explicit zero-inference evaluation")
	}
	space, ok := new(big.Int).SetString(c.Space, 10)
	count := int64(len(c.Attempts))
	if !ok || space.Sign() < 1 || space.String() != c.Space || count > *c.Budget || space.Cmp(big.NewInt(count)) < 0 {
		return Snapshot{}, fmt.Errorf("joint program attempts disagree with their budget or candidate space")
	}
	s, err := readJointRuntime(e.Runtime)
	if err != nil {
		return Snapshot{}, fmt.Errorf("joint evaluation: %w", err)
	}
	s.InputSHA = inputSHA
	j := &JointObservation{ProgramAttempts: count, ProgramBudget: *c.Budget, CandidateSpace: c.Space,
		CandidateKinds: c.Kinds,
		MoreCandidates: space.Cmp(big.NewInt(count)) > 0, SelectedAttempt: *c.Selected, Decision: c.Decision, StopReason: c.Stop,
		Replayed: *e.Replayed, NewModelCalls: *e.Calls, Inputs: e.Inputs}
	var initial result
	if !present(c.Initial) {
		return Snapshot{}, fmt.Errorf("joint initial preparation is missing")
	}
	if err = json.Unmarshal(append(append([]byte(`{"composition":`), c.Initial...), '}'), &initial); err != nil {
		return Snapshot{}, err
	}
	j.Initial = constructionObservations(initial)
	var initialFillRejections int64
	for _, o := range j.Initial {
		if o.Kind == "source_fill" {
			if !o.Consistent {
				return Snapshot{}, fmt.Errorf("initial fill assignments disagree with their observations")
			}
			initialFillRejections += o.Rejected
		}
	}
	if initialFillRejections > 0 && c.Schema != "gooo/joint-construction/v5" && c.Schema != "gooo/joint-construction/v6" {
		return Snapshot{}, fmt.Errorf("initial fill rejections require v5/v6")
	}
	fillRejected := false
	for index, a := range c.Attempts {
		o := JointAttemptObservation{Rejection: a.Rejection}
		if a.Rejection != nil {
			if err := validateJointRejection(c.Schema, c.Kinds, a.Masks, a.Rejection, len(a.Candidates), a.SearchCandidates, a.FillCandidates, a.Runtime); err != nil {
				return Snapshot{}, err
			}
			j.RejectedAttempts++
			fillRejected = fillRejected || a.Rejection.Stage == "LOCAL_SOURCE_FILL"
		} else {
			native, err := readJointRuntime(a.Runtime)
			if err != nil {
				return Snapshot{}, fmt.Errorf("joint attempt %d caller: %w", index, err)
			}
			o.NativeOutcomes = native.NativeOutcomes
			if o.NativeOutcomes != nil {
				j.NativeFaultAttempts++
			}
			// Caller scores use whole named outputs even when they return records.
			summary, err := summarize(append(append([]byte(`{"runtime":`), a.Runtime...), '}'), "caller", "recorded")
			if err != nil {
				return Snapshot{}, err
			}
			o.CallerPassed, o.CallerTotal = int64(summary.NamedPassed), int64(summary.NamedTotal)
			j.NativeProgramAttempts++
		}
		if len(a.Candidates)+len(a.SearchCandidates)+len(a.FillCandidates) == 0 {
			return Snapshot{}, fmt.Errorf("joint attempt has no local candidates")
		}
		if a.Rejection == nil {
			if err := validateJointKinds(c.Schema, c.Kinds, a.Masks, len(a.Candidates), len(a.SearchCandidates), len(a.FillCandidates)); err != nil {
				return Snapshot{}, err
			}
		}
		for _, candidate := range a.Candidates {
			var matched int64
			for _, row := range candidate.Cases {
				actual, err := decodeValue(row.Actual)
				if err != nil {
					return Snapshot{}, err
				}
				expected, err := decodeValue(row.Expected)
				if err != nil {
					return Snapshot{}, err
				}
				pass := reflect.DeepEqual(actual, expected)
				if row.Passed == nil || *row.Passed != pass {
					return Snapshot{}, fmt.Errorf("joint local case flag differs from actual values")
				}
				if pass {
					matched++
				}
			}
			if candidate.Attempt.Passed == nil || candidate.Attempt.Total == nil || len(candidate.Cases) == 0 ||
				*candidate.Attempt.Passed != matched || *candidate.Attempt.Total != int64(len(candidate.Cases)) {
				return Snapshot{}, fmt.Errorf("joint local candidate counts differ from actual cases")
			}
			o.LocalPassed += matched
			o.LocalTotal += int64(len(candidate.Cases))
		}
		for i, candidate := range a.SearchCandidates {
			if a.Rejection != nil && a.Rejection.Stage == "LOCAL_SOURCE_SEARCH" && i == len(a.SearchCandidates)-1 {
				continue // This expression was checked above and has no local score.
			}
			matched, total, err := recountJointSearch(candidate)
			if err != nil {
				return Snapshot{}, err
			}
			o.LocalPassed += matched
			o.LocalTotal += total
		}
		for i, candidate := range a.FillCandidates {
			if a.Rejection != nil && a.Rejection.Stage == "LOCAL_SOURCE_FILL" && i == len(a.FillCandidates)-1 {
				continue
			}
			local, holdout, err := recountJointFill(candidate)
			if err != nil {
				return Snapshot{}, err
			}
			o.LocalPassed += local.passed
			o.LocalTotal += local.total
			o.FillHoldoutPassed += holdout.passed
			o.FillHoldoutTotal += holdout.total
		}
		if a.LocalPassed == nil || a.LocalTotal == nil || *a.LocalPassed != o.LocalPassed || *a.LocalTotal != o.LocalTotal {
			return Snapshot{}, fmt.Errorf("joint local totals differ from candidate cases")
		}
		j.History = append(j.History, o)
	}
	selected := j.History[j.SelectedAttempt]
	if (c.Schema == "gooo/joint-construction/v6") != (j.NativeFaultAttempts > 0) {
		return Snapshot{}, fmt.Errorf("v6 must describe native fault observations")
	}
	if c.Schema == "gooo/joint-construction/v5" && initialFillRejections == 0 && !fillRejected {
		return Snapshot{}, fmt.Errorf("v5 requires a recorded fill rejection")
	}
	if selected.Rejection != nil || c.Schema == "gooo/joint-construction/v3" && j.RejectedAttempts == 0 {
		return Snapshot{}, fmt.Errorf("joint rejection version or selected executable differs")
	}
	j.LocalPassed, j.LocalTotal = selected.LocalPassed, selected.LocalTotal
	j.FillHoldoutPassed, j.FillHoldoutTotal = selected.FillHoldoutPassed, selected.FillHoldoutTotal
	j.CallerPassed, j.CallerTotal = selected.CallerPassed, selected.CallerTotal
	if selected.NativeOutcomes != nil {
		j.NativeFaults, j.BlockedActivities = selected.NativeOutcomes.FaultedActivities, selected.NativeOutcomes.BlockedActivities
	}
	complete := j.LocalPassed == j.LocalTotal && j.CallerTotal > 0 && j.CallerPassed == j.CallerTotal &&
		j.NativeFaults == 0 && j.BlockedActivities == 0
	wantDecision, wantStop := "PARTIAL_FINITE", "PROGRAM_BUDGET_EXHAUSTED"
	if !j.MoreCandidates {
		wantStop = "DECLARED_SPACE_EXHAUSTED"
	}
	if complete {
		wantDecision, wantStop = "COMPLETE_FINITE", "LOCAL_AND_CALLER_CASES_MATCHED"
	}
	if c.Decision != wantDecision || c.Stop != wantStop || !complete && j.MoreCandidates && count != *c.Budget {
		return Snapshot{}, fmt.Errorf("joint decision or stopping reason disagrees with recorded obligations")
	}
	var runtime result
	if err = json.Unmarshal(append(append([]byte(`{"runtime":`), e.Runtime...), '}'), &runtime); err != nil {
		return Snapshot{}, err
	}
	inputs := e.Inputs
	if inputs.Unique == nil || inputs.Duplicates == nil || inputs.Consumed == nil || inputs.Other == nil ||
		*inputs.Unique < 0 || *inputs.Duplicates < 0 || *inputs.Consumed < 0 || *inputs.Other < 0 ||
		*inputs.Unique != *inputs.Consumed+*inputs.Other || *inputs.Unique+*inputs.Duplicates != int64(len(runtime.Runtime.Traces)) {
		return Snapshot{}, fmt.Errorf("joint evaluation input counts disagree with recorded rows")
	}
	s.Joint = j
	return s, nil
}

func readJointRuntime(raw json.RawMessage) (Snapshot, error) {
	return ReadSnapshot(append(append([]byte(`{"runtime":`), raw...), '}'))
}
