package workbench

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type RefineOptions struct {
	Options
	Source, Activity, Cases, Policy, EvaluationCases string
	MaxAttempts, MaxRounds                           int
}

type RefinementReport struct {
	Schema               string               `json:"schema"`
	Status               string               `json:"status"`
	Failure              string               `json:"failure,omitempty"`
	ActivityID           string               `json:"activity_id"`
	Initial              Snapshot             `json:"initial"`
	Final                Snapshot             `json:"final"`
	InitialPlan          ConstructionNextStep `json:"initial_plan"`
	FinalPlan            ConstructionNextStep `json:"final_plan"`
	Dispatched           bool                 `json:"dispatched"`
	StopReason           string               `json:"stop_reason"`
	RefinementStatus     string               `json:"refinement_status"`
	EvaluationStatus     string               `json:"evaluation_status"`
	Evaluation           *Snapshot            `json:"evaluation,omitempty"`
	InitialModelCalls    int                  `json:"initial_model_calls"`
	RefinementModelCalls int                  `json:"refinement_model_calls"`
	Rounds               int                  `json:"refinement_rounds"`
	SelectedSource       string               `json:"selected_source"`
	Scope                string               `json:"scope"`
}

// Refine dispatches source-owned next steps to the compiler's Gooo feedback
// policy. All inputs are copied; source revisions belong to new output artifacts.
func Refine(ctx context.Context, o RefineOptions) (report RefinementReport, err error) {
	report = RefinementReport{Schema: "gooo/workbench-refinement/v1", Status: "PROGRESS", RefinementStatus: "NOT_RUN", EvaluationStatus: "UNKNOWN",
		Scope: "Gooo next-step dispatch and bounded source revisions under an explicit Gooo policy; feedback is adaptive; optional final evaluation is withheld until selection; no model training or invented expectations"}
	if o.Source == "" || o.Activity == "" || o.Cases == "" || o.Policy == "" || o.MaxAttempts < 1 || o.MaxAttempts > 64 || o.MaxRounds < 1 || o.MaxRounds > 8 {
		return report, fmt.Errorf("refine requires source, activity, cases, policy, 1..64 max-attempts and 1..8 max-rounds")
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return report, err
	}
	defer func() {
		if err != nil {
			report.Status, report.Failure = "ERROR", err.Error()
		}
		if saveErr := save(filepath.Join(root, "refinement-dispatch.json"), report); err == nil {
			err = saveErr
		}
	}()
	source, cases, model, activityID, err := prepareRefinement(ctx, o, root)
	if err != nil {
		return report, err
	}
	report.ActivityID = activityID
	selected := filepath.Join(root, "initial-composition")
	args := []string{"body-compose", "--source", source, "--cases", cases, "--out", selected}
	if model != "" {
		args = append(args, "--model", model)
	}
	initial, err := command(ctx, o.Compiler, args...)
	if err != nil {
		return report, err
	}
	if err = write(filepath.Join(root, "initial-result.json"), initial); err != nil {
		return report, err
	}
	report.Initial, err = ReadSnapshot(initial)
	if err != nil {
		return report, err
	}
	counts, err := summarize(initial, "refine", "initial")
	if err != nil {
		return report, err
	}
	report.InitialModelCalls = counts.ModelCalls
	report.InitialPlan, err = refinementPlan(ctx, o.Options, filepath.Join(root, "initial-plan"), report.Initial, report.ActivityID)
	if err != nil {
		return report, err
	}
	selected, err = executeRefinement(ctx, o, root, model, selected, &report)
	if err != nil {
		return report, err
	}
	final, err := command(ctx, o.Compiler, "body-compose", "--source", filepath.Join(selected, "original.gooo"), "--cases", cases, "--composition", filepath.Join(selected, "composition.json"))
	if err != nil {
		return report, err
	}
	if err = write(filepath.Join(root, "final-result.json"), final); err != nil {
		return report, err
	}
	report.Final, err = ReadSnapshot(final)
	if err != nil {
		return report, err
	}
	report.SelectedSource, err = filepath.Rel(root, filepath.Join(selected, "original.gooo"))
	if err != nil {
		return report, err
	}
	report.FinalPlan, err = refinementPlan(ctx, o.Options, filepath.Join(root, "final-plan"), report.Final, report.ActivityID)
	if err != nil {
		return report, err
	}
	if report.Final.Total > 0 && report.Final.Passed == report.Final.Total {
		report.Status = "PASS"
	}
	if o.EvaluationCases != "" {
		evaluation, runErr := command(ctx, o.Compiler, "body-compose", "--source", filepath.Join(selected, "original.gooo"), "--cases", filepath.Join(root, "evaluation-cases.json"), "--composition", filepath.Join(selected, "composition.json"))
		if runErr != nil {
			return report, runErr
		}
		if err = write(filepath.Join(root, "evaluation-result.json"), evaluation); err != nil {
			return report, err
		}
		snapshot, readErr := ReadSnapshot(evaluation)
		if readErr != nil {
			return report, readErr
		}
		report.Evaluation, report.EvaluationStatus = &snapshot, "PROGRESS"
		if snapshot.Total > 0 && snapshot.Passed == snapshot.Total {
			report.EvaluationStatus = "PASS"
		} else {
			report.Status = "PROGRESS"
		}
	}
	return report, nil
}

func refinementPlan(ctx context.Context, o Options, root string, s Snapshot, id string) (ConstructionNextStep, error) {
	if err := os.Mkdir(root, 0755); err != nil {
		return ConstructionNextStep{}, err
	}
	plans, err := constructionNextSteps(ctx, o, root, s)
	if err != nil {
		return ConstructionNextStep{}, err
	}
	for _, plan := range plans {
		if plan.Observation.ActivityID == id {
			return plan, nil
		}
	}
	return ConstructionNextStep{}, fmt.Errorf("missing Gooo next step for %s", id)
}
