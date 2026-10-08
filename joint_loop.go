package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type JointRequest struct {
	Source, ConstructionCases, EvaluationCases, HoldoutCases, Entry string
	MaxProgramBudget                                                int64
	MaxRounds                                                       int
}

type JointRound struct {
	Budget                int64                `json:"program_budget"`
	Attempts              int64                `json:"program_attempts"`
	EvaluationPassed      int64                `json:"evaluation_passed"`
	EvaluationTotal       int64                `json:"evaluation_total"`
	EvaluationUnit        string               `json:"evaluation_unit"`
	Action                string               `json:"action"`
	ProposedBudget        int64                `json:"proposed_program_budget"`
	ConstructionElapsedNS int64                `json:"construction_elapsed_ns"`
	Result                string               `json:"result"`
	ConstructionFile      string               `json:"construction_cases_file"`
	Feedback              *JointFeedbackUpdate `json:"feedback,omitempty"`
}

type JointLoop struct {
	Schema           string       `json:"schema"`
	StopReason       string       `json:"stop_reason"`
	Rounds           []JointRound `json:"rounds"`
	FinalDirectory   string       `json:"final_directory"`
	Scope            string       `json:"scope"`
	FinalEvaluation  *Snapshot    `json:"final_evaluation,omitempty"`
	HoldoutElapsedNS int64        `json:"holdout_elapsed_ns,omitempty"`
	Failure          string       `json:"failure,omitempty"`
}

// ConstructJoint dispatches Gooo's budget proposals against frozen source and
// expectations. Each rerun starts fresh; selected-code replay is a separate route.
func ConstructJoint(ctx context.Context, o Options, request JointRequest) (loop JointLoop, runError error) {
	loop = JointLoop{Schema: "gooo/joint-construction-loop/v2", Scope: "bounded Gooo budget and counterexample proposals; adaptive evaluation influences selection and is not held out; original expectations are preserved; every round restarts and repeated attempts count separately; optional holdout runs only after selection stops, with zero new inference; input separation is not training independence; no source changes or model training"}
	if ctx == nil {
		return loop, fmt.Errorf("joint construction requires a context")
	}
	if request.MaxProgramBudget < 1 || request.MaxProgramBudget > 64 || request.MaxRounds < 1 || request.MaxRounds > 16 {
		return loop, fmt.Errorf("joint construction requires a program budget of 1..64 and 1..16 rounds")
	}
	// Read all inputs before creating output, and use these bytes in every round.
	inputs := map[string][]byte{}
	files := map[string]string{"source.gooo": request.Source, "construction-cases.json": request.ConstructionCases, "evaluation-cases.json": request.EvaluationCases}
	if request.HoldoutCases != "" {
		files["holdout-cases.json"] = request.HoldoutCases
	}
	for name, filename := range files {
		b, err := os.ReadFile(filename)
		if err != nil {
			return loop, err
		}
		inputs[name] = b
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return loop, err
	}
	defer func() {
		if runError != nil {
			loop.Failure = runError.Error()
			runError = errors.Join(runError, save(filepath.Join(root, "joint-loop.json"), loop))
		}
	}()
	for name, b := range inputs {
		if err = write(filepath.Join(root, name), b); err != nil {
			return loop, err
		}
	}
	model, err := prepareModel(o.Model, root)
	if err != nil {
		return loop, err
	}
	budget := int64(1)
	currentCases, currentFile := inputs["construction-cases.json"], "construction-cases.json"
	var pending *JointFeedbackUpdate
	for round := 0; round < request.MaxRounds; round++ {
		if err = ctx.Err(); err != nil {
			return loop, err
		}
		dir := fmt.Sprintf("round-%d", round)
		args := []string{"body-construct", "--source", filepath.Join(root, "source.gooo"), "--construction-cases", filepath.Join(root, currentFile),
			"--cases", filepath.Join(root, "evaluation-cases.json"), "--attempts", strconv.FormatInt(budget, 10), "--out", filepath.Join(root, dir)}
		if request.Entry != "" {
			args = append(args, "--entry", request.Entry)
		}
		if model != "" {
			args = append(args, "--model", model)
		}
		started := time.Now()
		raw, runErr := command(ctx, o.Compiler, args...)
		elapsed := time.Since(started).Nanoseconds()
		resultName := dir + ".json"
		if err = write(filepath.Join(root, resultName), raw); err != nil {
			return loop, err
		}
		if runErr != nil {
			return loop, runErr
		}
		s, err := ReadSnapshot(raw)
		if err != nil {
			return loop, err
		}
		if s.Joint == nil {
			return loop, fmt.Errorf("compiler omitted joint construction")
		}
		if pending != nil {
			pending.Consumed = true
			pending = nil
		}
		diagnostic, err := Diagnose(ctx, Options{Compiler: o.Compiler, Out: filepath.Join(root, dir+"-diagnosis")}, s)
		if err != nil {
			return loop, err
		}
		var plan struct {
			Action string `json:"action"`
			Next   int64  `json:"next_program_budget"`
		}
		if err = json.Unmarshal(diagnostic, &plan); err != nil {
			return loop, err
		}
		loop.Rounds = append(loop.Rounds, JointRound{Budget: budget, Attempts: s.Joint.ProgramAttempts, EvaluationPassed: s.Passed, EvaluationTotal: s.Total,
			EvaluationUnit: s.Unit, Action: plan.Action, ProposedBudget: plan.Next, ConstructionElapsedNS: elapsed, Result: resultName, ConstructionFile: currentFile})
		loop.FinalDirectory = dir
		loop.StopReason = plan.Action
		if plan.Action == "add-counterexamples-to-construction" {
			if plan.Next != budget {
				return loop, fmt.Errorf("Gooo counterexample proposal changed the program budget")
			}
			feedback, err := prepareJointFeedback(ctx, o, root, dir+"-feedback", currentCases, inputs["evaluation-cases.json"], raw)
			if err != nil {
				return loop, err
			}
			loop.Rounds[len(loop.Rounds)-1].Feedback = feedback
			if !feedback.Prepared {
				loop.StopReason = feedback.StopReason
				break
			}
			if round == request.MaxRounds-1 {
				loop.StopReason = "round-limit"
				break
			}
			currentFile = feedback.NextCasesFile
			currentCases, err = os.ReadFile(filepath.Join(root, currentFile))
			if err != nil {
				return loop, err
			}
			pending = feedback
			continue
		}
		if plan.Action != "rerun-with-larger-program-budget" {
			break
		}
		if plan.Next <= budget || plan.Next > 64 {
			return loop, fmt.Errorf("Gooo proposed a non-advancing or unsupported program budget")
		}
		if plan.Next > request.MaxProgramBudget {
			loop.StopReason = "program-budget-limit"
			break
		}
		if round == request.MaxRounds-1 {
			loop.StopReason = "round-limit"
			break
		}
		budget = plan.Next
	}
	// Persist selection before holdout, including when the optional replay fails.
	if err = save(filepath.Join(root, "joint-loop.json"), loop); err != nil {
		return loop, err
	}
	if request.HoldoutCases != "" {
		if err = ctx.Err(); err != nil {
			return loop, err
		}
		started := time.Now()
		raw, runErr := command(ctx, o.Compiler, "body-construct", "--source", filepath.Join(root, "source.gooo"),
			"--construction", filepath.Join(root, loop.FinalDirectory, "construction.json"), "--cases", filepath.Join(root, "holdout-cases.json"))
		loop.HoldoutElapsedNS = time.Since(started).Nanoseconds()
		if err = write(filepath.Join(root, "holdout-result.json"), raw); err != nil {
			return loop, err
		}
		if runErr != nil {
			return loop, runErr
		}
		final, err := ReadSnapshot(raw)
		if err != nil {
			return loop, err
		}
		if final.Joint == nil || !final.Joint.Replayed || final.Joint.NewModelCalls != 0 {
			return loop, fmt.Errorf("final holdout must replay saved construction without inference")
		}
		loop.FinalEvaluation = &final
	}
	err = save(filepath.Join(root, "joint-loop.json"), loop)
	return loop, err
}
