package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
)

func jointInput(s Snapshot) map[string]any {
	j := s.Joint
	return map[string]any{"local_passed": j.LocalPassed, "local_total": j.LocalTotal,
		"caller_passed": j.CallerPassed, "caller_total": j.CallerTotal,
		"evaluation_passed": s.Passed, "evaluation_total": s.Total,
		"program_attempts": j.ProgramAttempts, "program_budget": j.ProgramBudget,
		"program_faults": j.NativeFaults, "program_blocked": j.BlockedActivities,
		"evaluation_faults": nativeActivityCount(s, false), "evaluation_blocked": nativeActivityCount(s, true),
		"more_candidates": j.MoreCandidates, "other_inputs": *j.Inputs.Other}
}

func nativeActivityCount(s Snapshot, blocked bool) int64 {
	if s.NativeOutcomes == nil {
		return 0
	}
	if blocked {
		return s.NativeOutcomes.BlockedActivities
	}
	return s.NativeOutcomes.FaultedActivities
}

func diagnoseJoint(ctx context.Context, o Options, root string, s Snapshot) (json.RawMessage, error) {
	if err := save(filepath.Join(root, "observation.json"), s); err != nil {
		return nil, err
	}
	input := jointInput(s)
	cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{
		map[string]any{"inputs": map[string]any{"ObservationEcho": input}, "expected": map[string]any{"ObservationEcho": input}},
	}})
	if err != nil {
		return nil, err
	}
	_, r, err := runRecipe(ctx, o, root, "joint-next", "joint-next", "", cases)
	if err != nil {
		return nil, err
	}
	if r.Runtime.Calls != 0 || r.Runtime.Passed != 1 || r.Runtime.Total != 1 {
		return nil, fmt.Errorf("joint diagnostic lost its observation or predicted during execution")
	}
	var next struct {
		Code    string `json:"code"`
		Action  string `json:"action"`
		Message string `json:"message"`
		Budget  int64  `json:"next_program_budget"`
	}
	if err = actualFor(r, "jointnext://activity/next", &next); err != nil {
		return nil, err
	}
	if next.Code == "" || next.Action == "" || next.Message == "" {
		return nil, fmt.Errorf("joint diagnostic omitted its typed recommendation")
	}
	source, err := assets.ReadFile("recipes/joint-next.gooo")
	if err != nil {
		return nil, err
	}
	value := map[string]any{"schema": "gooo/joint-diagnostic/v1", "code": next.Code, "action": next.Action, "message": next.Message, "next_program_budget": next.Budget,
		"input_sha256": s.InputSHA, "source_sha256": fmt.Sprintf("%x", sha256.Sum256(source)), "compiler_source": r.Runtime.Source,
		"evaluation":         map[string]any{"unit": s.Unit, "passed": s.Passed, "total": s.Total, "detail": s.Detail},
		"joint_construction": s.Joint, "new_model_calls": 0, "model_requested": o.Model != "", "model_used": false,
		"scope": "Gooo advice from recounted receipt values; original program execution, source identity and root-input membership are not independently reverified by diagnose; no proposed action is executed; diagnostic rules are deterministic"}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, write(filepath.Join(root, "diagnostic.json"), append(raw, '\n'))
}
