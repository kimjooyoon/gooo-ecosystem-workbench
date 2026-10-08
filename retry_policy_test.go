package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
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
