package workbench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// Expected outputs and activity outcomes have different denominators.
type NativeOutcomes struct {
	Matched           int64 `json:"matched"`
	Mismatched        int64 `json:"mismatched"`
	Faulted           int64 `json:"faulted"`
	Blocked           int64 `json:"blocked"`
	Unobserved        int64 `json:"unobserved"`
	FaultedActivities int64 `json:"faulted_activities"`
	BlockedActivities int64 `json:"blocked_activities"`
}

type nativeFaultSite struct {
	ExpressionID     string `json:"expression_id"`
	RootActivityID   string `json:"root_activity_id"`
	ActivityID       string `json:"activity_id"`
	Operator         string `json:"operator"`
	Expression       string `json:"expression"`
	ProjectionSHA256 string `json:"projection_sha256"`
	Start            int    `json:"start"`
	End              int    `json:"end"`
}

type nativeFault struct {
	Kind  string          `json:"kind"`
	Site  nativeFaultSite `json:"site"`
	Left  *int64          `json:"left"`
	Right *int64          `json:"right"`
}

type nativeOutcomeDelivery struct {
	ID       string          `json:"activity_id"`
	Actual   json.RawMessage `json:"actual"`
	Expected json.RawMessage `json:"expected"`
	Passed   *bool           `json:"passed"`
	Fault    *nativeFault    `json:"fault"`
	Blocked  []string        `json:"blocked_by"`
	Producer string          `json:"producer_id"`
	Input    json.RawMessage `json:"input"`
	Inputs   []struct {
		Producer string          `json:"producer_id"`
		Value    json.RawMessage `json:"value"`
	} `json:"inputs"`
}

type nativeOutcomeRuntime struct {
	Schema             string                                                              `json:"schema"`
	Stage              string                                                              `json:"stage"`
	Failure            string                                                              `json:"failure"`
	Generated          string                                                              `json:"generated_sha256"`
	Calls              *int64                                                              `json:"model_calls"`
	Passed             *int64                                                              `json:"finite_passed"`
	Total              *int64                                                              `json:"finite_total"`
	Replayed           bool                                                                `json:"runtime_replayed"`
	ProjectionReplayed bool                                                                `json:"projection_replayed"`
	Sites              []nativeFaultSite                                                   `json:"fault_sites"`
	Outcomes           *struct{ Matched, Mismatched, Faulted, Blocked, Unobserved *int64 } `json:"outcomes"`
	Runs               []struct {
		Started     bool   `json:"started"`
		Completed   bool   `json:"completed"`
		Canceled    bool   `json:"canceled"`
		TimedOut    bool   `json:"timed_out"`
		Truncated   bool   `json:"output_truncated"`
		Diagnostics int    `json:"diagnostics_bytes"`
		Exit        *int   `json:"exit_code"`
		Stdout      string `json:"stdout_sha256"`
		Stderr      string `json:"stderr_sha256"`
	} `json:"runs"`
	Traces []struct {
		Index      int                     `json:"case_index"`
		Deliveries []nativeOutcomeDelivery `json:"deliveries"`
	} `json:"traces"`
}

func readNativeOutcomes(raw json.RawMessage, inputSHA string) (Snapshot, bool, error) {
	var r nativeOutcomeRuntime
	if err := json.Unmarshal(raw, &r); err != nil {
		return Snapshot{}, false, err
	}
	hasFault := false
	for _, row := range r.Traces {
		for _, d := range row.Deliveries {
			hasFault = hasFault || d.Fault != nil || len(d.Blocked) > 0
		}
	}
	active := hasFault || r.Schema == "gooo/body-composition-runtime/v3" || r.Outcomes != nil || len(r.Sites) > 0
	if !active {
		return Snapshot{}, false, nil
	}
	if !slices.Contains([]string{"gooo/body-composition-runtime/v1", "gooo/body-composition-runtime/v2", "gooo/body-composition-runtime/v3"}, r.Schema) ||
		hasFault != (r.Schema == "gooo/body-composition-runtime/v3") || r.Outcomes == nil || len(r.Sites) == 0 || !nativeDigest(r.Generated) ||
		r.Stage != "COMPLETE" || r.Failure != "" || !r.Replayed || !r.ProjectionReplayed || r.Calls == nil || *r.Calls != 0 || r.Passed == nil || r.Total == nil {
		return Snapshot{}, true, fmt.Errorf("native arithmetic outcomes require a complete bound runtime")
	}
	if err := validateNativeOutcomeRuns(r); err != nil {
		return Snapshot{}, true, err
	}
	sites, err := nativeOutcomeSites(r)
	if err != nil {
		return Snapshot{}, true, err
	}
	var counts NativeOutcomes
	var gaps []map[string]any
	for i, row := range r.Traces {
		if row.Index != i {
			return Snapshot{}, true, fmt.Errorf("native outcome case order differs")
		}
		seen := map[string]nativeOutcomeDelivery{}
		for _, d := range row.Deliveries {
			if err := recountNativeDelivery(d, seen, sites, &counts); err != nil {
				return Snapshot{}, true, err
			}
			seen[d.ID] = d
			if present(d.Expected) && d.Passed != nil && !*d.Passed {
				gaps = append(gaps, map[string]any{"activity": d.ID, "actual": d.Actual, "expected": d.Expected, "fault": d.Fault, "blocked_by": d.Blocked})
			}
		}
	}
	if err := compareNativeOutcomeCounts(r, counts); err != nil {
		return Snapshot{}, true, err
	}
	if !hasFault {
		return Snapshot{}, false, nil
	} // Keep existing record-field units for successful values.
	detail, _ := json.Marshal(map[string]any{"unit": "activity_outputs", "native_outcomes": counts, "gaps": gaps})
	if len(detail) > 1024 {
		detail, _ = json.Marshal(map[string]any{"unit": "activity_outputs", "native_outcomes": counts, "gaps": len(gaps), "detail_limited": true, "input_sha256": inputSHA})
	}
	return Snapshot{Unit: "activity_outputs", Passed: counts.Matched, Total: *r.Total, InputSHA: inputSHA, NativeOutcomes: &counts, Detail: string(detail)}, true, nil
}

func validateNativeOutcomeRuns(r nativeOutcomeRuntime) error {
	if len(r.Runs) != 2 {
		return fmt.Errorf("native outcomes require two process observations")
	}
	for _, run := range r.Runs {
		if !run.Started || !run.Completed || run.Canceled || run.TimedOut || run.Truncated || run.Exit == nil || *run.Exit != 0 ||
			run.Diagnostics != 0 || run.Stderr != fmt.Sprintf("sha256:%x", sha256.Sum256(nil)) || !nativeDigest(run.Stdout) || run.Stdout != r.Runs[0].Stdout {
			return fmt.Errorf("native outcome execution is incomplete or differs on replay")
		}
	}
	return nil
}

func nativeDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}

func nativeOutcomeSites(r nativeOutcomeRuntime) (map[string]nativeFaultSite, error) {
	sites := map[string]nativeFaultSite{}
	for _, site := range r.Sites {
		id := site.ExpressionID
		if _, duplicate := sites[id]; duplicate {
			return nil, fmt.Errorf("duplicate native fault site")
		}
		site.ExpressionID = ""
		raw, err := json.Marshal(site)
		if err != nil {
			return nil, err
		}
		if id != fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) || site.ProjectionSHA256 != r.Generated ||
			site.Start < 0 || site.End <= site.Start || site.End-site.Start != len(site.Expression) || site.ActivityID == "" || site.RootActivityID == "" ||
			site.Operator != "/" && site.Operator != "%" {
			return nil, fmt.Errorf("native fault site differs from its projection binding")
		}
		site.ExpressionID = id
		sites[id] = site
	}
	return sites, nil
}

func recountNativeDelivery(d nativeOutcomeDelivery, seen map[string]nativeOutcomeDelivery, sites map[string]nativeFaultSite, counts *NativeOutcomes) error {
	if _, duplicate := seen[d.ID]; d.ID == "" || duplicate {
		return fmt.Errorf("native activity identity is missing or duplicated")
	}
	blocked, err := nativeBlockedInputs(d, seen)
	if err != nil {
		return err
	}
	if !slices.Equal(blocked, d.Blocked) {
		return fmt.Errorf("blocked outcome differs from its observed producers")
	}
	if d.Fault != nil {
		f := d.Fault
		site, exists := sites[f.Site.ExpressionID]
		if !exists || !reflect.DeepEqual(site, f.Site) || f.Site.RootActivityID != d.ID || f.Kind != "ZERO_DIVISOR" ||
			f.Left == nil || f.Right == nil || *f.Right != 0 || present(d.Actual) || len(blocked) > 0 {
			return fmt.Errorf("invalid or contradictory native fault")
		}
		counts.FaultedActivities++
	} else if len(blocked) > 0 {
		if present(d.Actual) {
			return fmt.Errorf("blocked activity invented a value")
		}
		counts.BlockedActivities++
	} else if !present(d.Actual) {
		return fmt.Errorf("native activity has no value or explicit outcome")
	}
	if !present(d.Expected) {
		if d.Passed != nil {
			return fmt.Errorf("native activity claimed a score without an expectation")
		}
		return nil
	}
	a, err := decodeValue(d.Actual)
	if err != nil {
		return err
	}
	b, err := decodeValue(d.Expected)
	if err != nil {
		return err
	}
	match := d.Fault == nil && len(blocked) == 0 && reflect.DeepEqual(a, b)
	if d.Passed == nil || *d.Passed != match {
		return fmt.Errorf("native outcome flag differs from its actual value")
	}
	switch {
	case d.Fault != nil:
		counts.Faulted++
	case len(blocked) > 0:
		counts.Blocked++
	case match:
		counts.Matched++
	default:
		counts.Mismatched++
	}
	return nil
}

func nativeBlockedInputs(d nativeOutcomeDelivery, seen map[string]nativeOutcomeDelivery) ([]string, error) {
	var blocked []string
	check := func(id string, value json.RawMessage) error {
		if id == "" {
			return nil
		}
		producer, ok := seen[id]
		if !ok {
			return fmt.Errorf("native input refers to an unobserved producer")
		}
		if producer.Fault != nil || len(producer.Blocked) > 0 {
			if present(value) {
				return fmt.Errorf("failed producer supplied an invented value")
			}
			if !slices.Contains(blocked, id) {
				blocked = append(blocked, id)
			}
			return nil
		}
		a, err := decodeValue(value)
		if err != nil {
			return err
		}
		b, err := decodeValue(producer.Actual)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(a, b) {
			return fmt.Errorf("native input differs from its producer value")
		}
		return nil
	}
	if err := check(d.Producer, d.Input); err != nil {
		return nil, err
	}
	for _, input := range d.Inputs {
		if err := check(input.Producer, input.Value); err != nil {
			return nil, err
		}
	}
	return blocked, nil
}

func compareNativeOutcomeCounts(r nativeOutcomeRuntime, c NativeOutcomes) error {
	values := [5]int64{c.Matched, c.Mismatched, c.Faulted, c.Blocked, c.Unobserved}
	stored := [5]*int64{r.Outcomes.Matched, r.Outcomes.Mismatched, r.Outcomes.Faulted, r.Outcomes.Blocked, r.Outcomes.Unobserved}
	for i, value := range values {
		if stored[i] == nil || *stored[i] != value {
			return fmt.Errorf("native outcome totals differ from original deliveries")
		}
	}
	if *r.Passed != c.Matched || *r.Total != c.Matched+c.Mismatched+c.Faulted+c.Blocked {
		return fmt.Errorf("native finite totals differ from measured outcomes")
	}
	return nil
}
