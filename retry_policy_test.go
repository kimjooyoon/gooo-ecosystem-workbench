package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This oracle uses arbitrary-precision arithmetic before capping. The Gooo
// implementation instead guards the int64 multiplication before executing it.
func retryOracle(done, transient bool, used, limit, previous, cap int64) map[string]any {
	plan := map[string]any{"retry": false, "delay_ms": int64(0), "reason": "invalid-input"}
	if used < 0 || limit < 0 || previous < 0 || cap < 0 {
		return plan
	}
	if done {
		plan["reason"] = "completed"
	} else if !transient {
		plan["reason"] = "permanent-failure"
	} else if used >= limit {
		plan["reason"] = "attempt-limit"
	} else {
		delay := new(big.Int).Lsh(big.NewInt(previous), 1)
		if delay.Sign() == 0 {
			delay.SetInt64(1)
		}
		if delay.Cmp(big.NewInt(cap)) > 0 {
			delay.SetInt64(cap)
		}
		plan["retry"], plan["delay_ms"], plan["reason"] = true, delay.Int64(), "retry"
	}
	return plan
}

func TestNativeRetryPolicyKeepsBranchesAndIntegerBoundaries(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	// Every case has used=2, which differs from the five source selection tuples.
	var cases []map[string]any
	for _, done := range []bool{false, true} {
		for _, transient := range []bool{false, true} {
			for _, limit := range []int64{-1, 0, 2, 3, math.MaxInt64} {
				for _, pair := range [][2]int64{{-1, 100}, {10, -1}, {0, 0}, {0, 100}, {10, 19}, {10, 20}, {10, 21}, {math.MaxInt64 / 2, math.MaxInt64}, {math.MaxInt64/2 + 1, math.MaxInt64}, {math.MaxInt64, math.MaxInt64}} {
					inputs := map[string]any{"PlanRetry.input0": done, "PlanRetry.input1": transient, "PlanRetry.input2": int64(2), "PlanRetry.input3": limit, "PlanRetry.input4": pair[0], "PlanRetry.input5": pair[1]}
					cases = append(cases, map[string]any{"inputs": inputs, "expected": map[string]any{"PlanRetry": retryOracle(done, transient, 2, limit, pair[0], pair[1])}})
				}
			}
		}
	}
	// Keep all 200 cases, split across the compiler's 32 KiB suite-file bound.
	var casePaths []string
	caseRoot := t.TempDir()
	const batchSize = 50
	for start := 0; start < len(cases); start += batchSize {
		raw, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases[start : start+batchSize]})
		path := filepath.Join(caseRoot, fmt.Sprintf("batch-%d.json", start/batchSize))
		if err != nil || len(raw) > 32768 {
			t.Fatal("invalid finite evaluation batch", len(raw), err)
		}
		if err := write(path, raw); err != nil {
			t.Fatal(err)
		}
		casePaths = append(casePaths, path)
	}
	for _, mode := range []string{"fixed", "model"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			composition := filepath.Join(root, "composition")
			args := []string{"body-compose", "--source", "examples/retry-policy/source.gooo", "--cases", casePaths[0], "--out", composition}
			if mode == "model" {
				model, err := prepareModel("builtin", root)
				if err != nil {
					t.Fatal(err)
				}
				args = append(args, "--model", model)
			}
			output, err := command(context.Background(), compiler, args...)
			if err != nil {
				t.Fatal(err)
			}
			summary, err := summarize(output, "retry-policy", mode)
			wantCalls := 0
			if mode == "model" {
				wantCalls = 1
			}
			if err != nil || summary.NamedPassed != batchSize || summary.NamedTotal != batchSize || summary.FieldsPassed != 3*batchSize || summary.FieldsTotal != 3*batchSize || summary.ModelCalls != wantCalls || summary.SelectionPassed != 15 || summary.SelectionTotal != 15 {
				t.Fatal("Gooo retry policy differs from the finite independent oracle", summary, err)
			}
			var built result
			if err := json.Unmarshal(output, &built); err != nil || len(built.Composition.Steps) != 1 {
				t.Fatal("retry construction did not retain its activity", err)
			}
			assembly := built.Composition.Steps[0].Generation.Report.Assembly
			wantAttempts := 8
			if mode == "model" {
				wantAttempts = 1
			}
			if assembly == nil || len(assembly.Attempts) != wantAttempts {
				t.Fatal("frozen retry fixture changed candidate order", assembly)
			}
			for _, attempt := range assembly.Attempts {
				if attempt.Mask == nil || attempt.Status != "" || attempt.Reason != "" {
					t.Fatal("unused candidate locals caused a rejection", attempt)
				}
			}
			passed := 0
			for _, casesPath := range casePaths {
				replay, err := command(context.Background(), compiler, "body-compose", "--source", filepath.Join(composition, "original.gooo"), "--composition", filepath.Join(composition, "composition.json"), "--cases", casesPath)
				var saved result
				if err != nil || json.Unmarshal(replay, &saved) != nil || saved.Generated || saved.Runtime.Calls != 0 {
					t.Fatal("saved retry policy invoked new inference", err)
				}
				observed, err := summarize(replay, "retry-policy", mode)
				if err != nil || observed.NamedPassed != batchSize || observed.NamedTotal != batchSize || observed.FieldsPassed != 3*batchSize || observed.FieldsTotal != 3*batchSize {
					t.Fatal("saved retry policy differs from finite independent oracle", casesPath, observed, err)
				}
				passed += observed.NamedPassed
			}
			t.Logf("%s: %d/%d distinct plans, %d/%d fields, construction calls %d, replay calls 0", mode, passed, len(cases), 3*passed, 3*len(cases), summary.ModelCalls)
		})
	}
}

func TestNativeRetryPolicyRetainsAnUnconnectedBaseline(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	source, err := os.ReadFile("examples/retry-policy/source.gooo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(source), `attempts "8"`) != 1 {
		t.Fatal("retry source no longer declares the eight-candidate budget")
	}
	root := t.TempDir()
	sourcePath, out := filepath.Join(root, "source.gooo"), filepath.Join(root, "composition")
	if err := write(sourcePath, []byte(strings.Replace(string(source), `attempts "8"`, `attempts "1"`, 1))); err != nil {
		t.Fatal(err)
	}
	var selected string
	for _, replay := range []bool{false, true} {
		args := []string{"body-compose", "--source", sourcePath, "--cases", "examples/retry-policy/cases.json"}
		if replay {
			args = append(args, "--composition", filepath.Join(out, "composition.json"))
		} else {
			args = append(args, "--out", out)
		}
		raw, err := command(context.Background(), compiler, args...)
		var observed result
		if err != nil || json.Unmarshal(raw, &observed) != nil || len(observed.Composition.Steps) != 1 {
			t.Fatal("unread locals prevented partial construction", err)
		}
		a := observed.Composition.Steps[0].Generation.Report.Assembly
		if a == nil || a.CasePassed == nil || *a.CasePassed != 0 || a.CaseTotal == nil || *a.CaseTotal != 5 ||
			a.Passed != 8 || a.Total != 15 || len(a.Attempts) != 1 || a.Calls != 0 {
			t.Fatal("unconnected baseline lost its measured partial result", a)
		}
		attempt := a.Attempts[0]
		if attempt.Mask == nil || *attempt.Mask != 0 || attempt.Status != "" || attempt.Reason != "" {
			t.Fatal("unconnected baseline was rejected", attempt)
		}
		if observed.Generated == replay || observed.Runtime.Calls != 0 || observed.Runtime.Passed != 0 || observed.Runtime.Total != 12 {
			t.Fatal("partial result acquired success or new inference", observed.Runtime)
		}
		if observed.Composition.GeneratedSHA == "" || (replay && selected != observed.Composition.GeneratedSHA) {
			t.Fatal("saved baseline changed its generated program")
		}
		selected = observed.Composition.GeneratedSHA
	}
}

func TestRetryOracleExamples(t *testing.T) {
	for _, tc := range []struct {
		previous, cap, want int64
	}{{0, 0, 0}, {0, 100, 1}, {10, 19, 19}, {10, 20, 20}, {math.MaxInt64, math.MaxInt64, math.MaxInt64}} {
		got := retryOracle(false, true, 1, 2, tc.previous, tc.cap)
		if got["delay_ms"] != tc.want || got["retry"] != true {
			t.Fatal(fmt.Sprint(tc), got)
		}
	}
}
