package workbench

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
)

// JointObservation separates historical local preparation, whole-program search,
// and subsequent evaluation. Counts describe supplied finite observations.
type JointObservation struct {
	Initial         []ConstructionObservation `json:"initial_local_construction"`
	History         []JointAttemptObservation `json:"program_history"`
	ProgramAttempts int64                     `json:"program_attempts"`
	ProgramBudget   int64                     `json:"program_budget"`
	CandidateSpace  string                    `json:"candidate_space"`
	CandidateKinds  []string                  `json:"candidate_kinds,omitempty"`
	MoreCandidates  bool                      `json:"more_candidates"`
	SelectedAttempt int                       `json:"selected_attempt"`
	LocalPassed     int64                     `json:"local_passed"`
	LocalTotal      int64                     `json:"local_total"`
	CallerPassed    int64                     `json:"caller_passed"`
	CallerTotal     int64                     `json:"caller_total"`
	Decision        string                    `json:"decision"`
	StopReason      string                    `json:"stop_reason"`
	Replayed        bool                      `json:"construction_replayed"`
	NewModelCalls   int64                     `json:"new_model_calls"`
	Inputs          jointInputs               `json:"reported_evaluation_inputs"`
}

type JointAttemptObservation struct {
	LocalPassed  int64 `json:"local_passed"`
	LocalTotal   int64 `json:"local_total"`
	CallerPassed int64 `json:"caller_passed"`
	CallerTotal  int64 `json:"caller_total"`
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
			Masks       []int           `json:"masks"`
			LocalPassed *int64          `json:"local_passed"`
			LocalTotal  *int64          `json:"local_total"`
			Runtime     json.RawMessage `json:"runtime"`
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
	if c.Schema != "gooo/joint-construction/v1" && c.Schema != "gooo/joint-construction/v2" || c.Stage != "COMPLETE" || c.Failure != "" ||
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
	for index, a := range c.Attempts {
		_, err := readJointRuntime(a.Runtime)
		if err != nil {
			return Snapshot{}, fmt.Errorf("joint attempt %d caller: %w", index, err)
		}
		// Caller scores use whole named outputs even when they return records.
		summary, err := summarize(append(append([]byte(`{"runtime":`), a.Runtime...), '}'), "caller", "recorded")
		if err != nil {
			return Snapshot{}, err
		}
		o := JointAttemptObservation{CallerPassed: int64(summary.NamedPassed), CallerTotal: int64(summary.NamedTotal)}
		if len(a.Candidates)+len(a.SearchCandidates) == 0 {
			return Snapshot{}, fmt.Errorf("joint attempt has no local candidates")
		}
		if err := validateJointKinds(c.Schema, c.Kinds, a.Masks, len(a.Candidates), len(a.SearchCandidates)); err != nil {
			return Snapshot{}, err
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
		for _, candidate := range a.SearchCandidates {
			matched, total, err := recountJointSearch(candidate)
			if err != nil {
				return Snapshot{}, err
			}
			o.LocalPassed += matched
			o.LocalTotal += total
		}
		if a.LocalPassed == nil || a.LocalTotal == nil || *a.LocalPassed != o.LocalPassed || *a.LocalTotal != o.LocalTotal {
			return Snapshot{}, fmt.Errorf("joint local totals differ from candidate cases")
		}
		j.History = append(j.History, o)
	}
	selected := j.History[j.SelectedAttempt]
	j.LocalPassed, j.LocalTotal = selected.LocalPassed, selected.LocalTotal
	j.CallerPassed, j.CallerTotal = selected.CallerPassed, selected.CallerTotal
	complete := j.LocalPassed == j.LocalTotal && j.CallerTotal > 0 && j.CallerPassed == j.CallerTotal
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
