package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

type delivery struct {
	ID       string          `json:"activity_id"`
	Actual   json.RawMessage `json:"actual"`
	Expected json.RawMessage `json:"expected"`
	Passed   *bool           `json:"passed"`
	Fault    *struct {
		Kind  string `json:"kind"`
		Right int64  `json:"right"`
		Site  struct {
			Activity   string `json:"activity_id"`
			Expression string `json:"expression_id"`
			Operator   string `json:"operator"`
			Projection string `json:"projection_sha256"`
		} `json:"site"`
	} `json:"fault"`
	Blocked []string `json:"blocked_by"`
}
type outcomes struct{ Matched, Mismatched, Faulted, Blocked, Unobserved int }
type runtimeObservation struct {
	Stage     string    `json:"stage"`
	Source    string    `json:"original_source_sha256"`
	Generated string    `json:"generated_sha256"`
	Passed    int       `json:"finite_passed"`
	Total     int       `json:"finite_total"`
	Calls     int       `json:"model_calls"`
	Replayed  bool      `json:"runtime_replayed"`
	Outcomes  *outcomes `json:"outcomes"`
	Runs      []struct {
		Completed bool   `json:"completed"`
		Exit      *int   `json:"exit_code"`
		Stdout    string `json:"stdout_sha256"`
	} `json:"runs"`
	Traces []struct {
		Index      int        `json:"case_index"`
		Deliveries []delivery `json:"deliveries"`
	} `json:"traces"`
}
type program struct {
	Plan struct {
		Activities []struct{ ID, Name string } `json:"activities"`
	} `json:"plan"`
}
type report struct {
	Generated    bool `json:"generated_now"`
	Construction struct {
		Schema   string          `json:"schema"`
		Decision string          `json:"decision"`
		Stop     string          `json:"stop_reason"`
		Original string          `json:"original_source_sha256"`
		Source   string          `json:"selected_source"`
		Budget   int             `json:"program_budget"`
		Space    string          `json:"candidate_space"`
		Cases    json.RawMessage `json:"construction_cases"`
		Selected program         `json:"selected"`
		Initial  struct {
			Preparations []struct {
				Generation struct {
					Report struct {
						Record *struct {
							Calls int   `json:"model_calls"`
							NS    int64 `json:"predict_ns"`
						} `json:"record_assembly"`
						Fill *struct {
							Calls int `json:"local_model_predictions"`
						} `json:"body_fill"`
					} `json:"report"`
				} `json:"generation"`
			} `json:"preparations"`
		} `json:"initial"`
		Attempts []struct {
			Rejection json.RawMessage    `json:"rejection"`
			Runtime   runtimeObservation `json:"runtime"`
		} `json:"attempts"`
	} `json:"construction"`
	Evaluation struct {
		Runtime  runtimeObservation `json:"runtime"`
		Calls    int                `json:"new_model_calls"`
		Replayed bool               `json:"construction_replayed"`
	} `json:"evaluation"`
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool, detail string) {
	if !ok {
		panic(detail)
	}
}
func read(name string) []byte { b, e := os.ReadFile(name); check(e); return b }
func exact(raw []byte) any {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	check(d.Decode(&v))
	return v
}
func hash(raw []byte) string            { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }
func observed(raw json.RawMessage) bool { return len(raw) > 0 && string(raw) != "null" }

func recount(r runtimeObservation) (outcomes, int) {
	require(r.Stage == "COMPLETE" && r.Replayed && r.Calls == 0 && len(r.Runs) == 2, "native execution incomplete")
	for _, p := range r.Runs {
		require(p.Completed && p.Exit != nil && *p.Exit == 0, "native process failed")
	}
	require(r.Runs[0].Stdout == r.Runs[1].Stdout, "native outputs differ")
	var counts outcomes
	faults := 0
	for i, row := range r.Traces {
		require(row.Index == i, "case order differs")
		for _, d := range row.Deliveries {
			if d.Fault != nil {
				faults++
				f := d.Fault
				require(f.Kind == "ZERO_DIVISOR" && f.Right == 0 && f.Site.Expression != "" && f.Site.Activity != "" && (f.Site.Operator == "/" || f.Site.Operator == "%") && f.Site.Projection == r.Generated, "unbound language fault")
				require(!observed(d.Actual) && len(d.Blocked) == 0, "fault invented a value")
			}
			if len(d.Blocked) > 0 {
				require(!observed(d.Actual), "blocked activity invented a value")
			}
			if !observed(d.Expected) {
				continue
			}
			match := d.Fault == nil && len(d.Blocked) == 0 && reflect.DeepEqual(exact(d.Actual), exact(d.Expected))
			require(d.Passed != nil && *d.Passed == match, "pass flag differs from original value")
			switch {
			case d.Fault != nil:
				counts.Faulted++
			case len(d.Blocked) > 0:
				counts.Blocked++
			case match:
				counts.Matched++
			default:
				counts.Mismatched++
			}
		}
	}
	require(counts.Matched == r.Passed && counts.Matched+counts.Mismatched+counts.Faulted+counts.Blocked == r.Total, "runtime totals differ")
	if r.Outcomes != nil {
		require(*r.Outcomes == counts, "outcome categories differ")
	}
	return counts, faults
}

func observe(root, fixtures, name string) (map[string]any, string) {
	z, err := gzip.NewReader(bytes.NewReader(read(filepath.Join(root, name+".json.gz"))))
	check(err)
	defer z.Close()
	raw, err := io.ReadAll(z)
	check(err)
	var r report
	check(json.Unmarshal(raw, &r))
	snapshot, err := workbench.ReadSnapshot(raw)
	check(err)
	c := r.Construction
	mixed := len(name) >= 5 && name[:5] == "mixed"
	source, construction, evaluation := "source.gooo", "evaluation-cases.json", "holdout-cases.json"
	if mixed {
		source, construction, evaluation = "mixed-model.gooo", "mixed-construction-cases.json", "mixed-evaluation-cases.json"
	}
	require(c.Original == hash(read(filepath.Join(fixtures, source))), "original source changed")
	require(reflect.DeepEqual(exact(c.Cases), exact(read(filepath.Join(fixtures, construction)))), "original caller cases changed")
	require(c.Schema == "gooo/joint-construction/v6" && len(c.Attempts) <= c.Budget, "history version or budget differs")
	rejected, native, faulted := 0, 0, 0
	for _, a := range c.Attempts {
		if observed(a.Rejection) {
			rejected++
			require(a.Runtime.Stage == "", "rejected candidate invented native results")
			continue
		}
		_, faults := recount(a.Runtime)
		native++
		if faults > 0 {
			faulted++
		}
	}
	counts, _ := recount(r.Evaluation.Runtime)
	j := snapshot.Joint
	require(j != nil && j.ProgramAttempts == int64(len(c.Attempts)) && j.RejectedAttempts == int64(rejected) &&
		j.NativeProgramAttempts == int64(native) && j.NativeFaultAttempts == int64(faulted) && snapshot.Passed == int64(counts.Matched) &&
		snapshot.Total == int64(r.Evaluation.Runtime.Total), "workbench counts differ from independent recount")
	require(r.Evaluation.Runtime.Source == hash([]byte(c.Source)), "evaluation source differs from selected program")
	var cases struct {
		Cases []struct {
			Expected map[string]json.RawMessage `json:"expected"`
		} `json:"cases"`
	}
	check(json.Unmarshal(read(filepath.Join(fixtures, evaluation)), &cases))
	require(len(cases.Cases) == len(r.Evaluation.Runtime.Traces), "original final row count differs")
	names := map[string]string{}
	for _, a := range c.Selected.Plan.Activities {
		names[a.ID] = a.Name
	}
	for i, row := range r.Evaluation.Runtime.Traces {
		require(len(row.Deliveries) == len(cases.Cases[i].Expected), "final expected activity count differs")
		for _, d := range row.Deliveries {
			want, ok := cases.Cases[i].Expected[names[d.ID]]
			require(ok, "unexpected activity identity")
			require(reflect.DeepEqual(exact(d.Expected), exact(want)), "original final expectation changed")
		}
	}
	historical, ns := 0, int64(0)
	for _, p := range c.Initial.Preparations {
		if a := p.Generation.Report.Record; a != nil {
			historical += a.Calls
			ns += a.NS
		}
		if f := p.Generation.Report.Fill; f != nil {
			historical += f.Calls
		}
	}
	fresh := 0
	if r.Generated {
		fresh = historical
	}
	require(r.Evaluation.Calls == 0 && (r.Generated != r.Evaluation.Replayed), "replay inferred again or state differs")
	return map[string]any{"run": name, "program_attempts": len(c.Attempts), "rejected_attempts": rejected, "native_combinations": native, "native_fault_combinations": faulted, "candidate_space": c.Space, "decision": c.Decision, "stop_reason": c.Stop, "final_matched": counts.Matched, "final_total": r.Evaluation.Runtime.Total, "fresh_model_predictions": fresh, "historical_prediction_ns": ns, "original_source_sha256": c.Original}, c.Source
}
func main() {
	root, fixtures := "examples/caller-native-failure", "examples/caller-native-failure"
	names := []string{"budget-5", "budget-6", "budget-replay", "mixed-fixed", "mixed-model", "mixed-replay"}
	rows := []map[string]any{}
	sources := map[string]string{}
	for _, name := range names {
		r, s := observe(root, fixtures, name)
		rows = append(rows, r)
		sources[name] = s
	}
	require(sources["budget-6"] == sources["budget-replay"] && sources["mixed-fixed"] == sources["mixed-model"] && sources["mixed-model"] == sources["mixed-replay"], "selected source differs")
	raw, e := json.MarshalIndent(map[string]any{"schema": "gooo/native-arithmetic-study/v1", "observations": rows, "scope": "two related finite programs; unchanged own model orders record choices; source fill order stays deterministic; original expectations and exact integer values retained"}, "", "  ")
	check(e)
	fmt.Println(string(raw))
}
