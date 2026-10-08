package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Read the compiler's source-derived plan before model loading or candidate
// evaluation, so the caller's bound also covers the initial construction.
func prepareRefinement(ctx context.Context, o RefineOptions, root string) (source, cases, model, activityID string, err error) {
	inputs := map[string]string{"source.gooo": o.Source, "feedback-cases.json": o.Cases, "policy.gooo": o.Policy}
	if o.EvaluationCases != "" {
		inputs["evaluation-cases.json"] = o.EvaluationCases
	}
	for name, path := range inputs {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			err = readErr
			return
		}
		if err = write(filepath.Join(root, name), raw); err != nil {
			return
		}
	}
	source, cases = filepath.Join(root, "source.gooo"), filepath.Join(root, "feedback-cases.json")
	raw, err := command(ctx, o.Compiler, "body-context", "--activity", o.Activity, "--include-plan", source)
	if err != nil {
		return
	}
	if err = write(filepath.Join(root, "source-plan.json"), raw); err != nil {
		return
	}
	var plan struct {
		Schema           string `json:"schema"`
		ActivityID       string `json:"activity_id"`
		ModelPredictions int    `json:"model_predictions"`
		CandidateTests   int    `json:"candidate_tests"`
		ExpandedPlan     *struct {
			MaxAttempts int `json:"max_attempts"`
		} `json:"expanded_plan"`
	}
	if err = json.Unmarshal(raw, &plan); err != nil {
		return
	}
	if plan.Schema != "gooo/record-assembly-input-export/v1" || plan.ActivityID == "" || plan.ExpandedPlan == nil || plan.ModelPredictions != 0 || plan.CandidateTests != 0 {
		err = fmt.Errorf("refine requires a source-derived record assembly plan without inference or candidate outcomes")
		return
	}
	if plan.ExpandedPlan.MaxAttempts < 1 || plan.ExpandedPlan.MaxAttempts > o.MaxAttempts {
		err = fmt.Errorf("source attempt budget %d exceeds requested range 1..%d", plan.ExpandedPlan.MaxAttempts, o.MaxAttempts)
		return
	}
	activityID = plan.ActivityID
	model, err = prepareModel(o.Model, root)
	return
}

func executeRefinement(ctx context.Context, o RefineOptions, root, model, selected string, report *RefinementReport) (string, error) {
	report.StopReason = "no-source-revision-proposal"
	switch report.InitialPlan.Action {
	case "raise-attempt-budget", "add-runtime-cases-to-construction":
		report.Dispatched = true
		refinementDir := filepath.Join(root, "source-refinement")
		args := []string{"body-refine", "--source", filepath.Join(root, "source.gooo"), "--activity", o.Activity, "--feedback-cases", filepath.Join(root, "feedback-cases.json"), "--policy", filepath.Join(root, "policy.gooo"), "--max-attempts", strconv.Itoa(o.MaxAttempts), "--max-rounds", strconv.Itoa(o.MaxRounds), "--out", refinementDir}
		if model != "" {
			args = append(args, "--model", model)
		}
		raw, runErr := command(ctx, o.Compiler, args...)
		if len(raw) > 0 {
			if err := write(filepath.Join(root, "refinement-result.json"), raw); err != nil {
				return "", err
			}
		}
		if runErr != nil {
			return "", runErr
		}
		var refinement struct {
			Schema        string            `json:"schema"`
			Status        string            `json:"status"`
			StopReason    string            `json:"stop_reason"`
			SelectedRound int               `json:"selected_round"`
			Rounds        []json.RawMessage `json:"rounds"`
		}
		if err := json.Unmarshal(raw, &refinement); err != nil {
			return "", err
		}
		if refinement.Schema != "gooo/body-refinement/v1" || refinement.SelectedRound < 0 || refinement.SelectedRound >= len(refinement.Rounds) {
			return "", fmt.Errorf("refinement did not retain a source round")
		}
		report.RefinementStatus, report.StopReason, report.Rounds = refinement.Status, refinement.StopReason, len(refinement.Rounds)
		for _, round := range refinement.Rounds {
			c, e := summarize(round, "refine", "round")
			if e != nil {
				return "", e
			}
			report.RefinementModelCalls += c.ModelCalls
		}
		selected = filepath.Join(refinementDir, "selected")
	}
	return selected, nil
}
