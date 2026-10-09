package workbench

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type AssemblyConstructionRequest struct {
	Assembly, HoldoutCases string
	MaxProgramBudget       int64
	MaxRounds              int
}

type AssemblyConstruction struct {
	Schema           string               `json:"schema"`
	StopReason       string               `json:"stop_reason"`
	OriginContextSHA string               `json:"origin_context_sha256"`
	Origin           Summary              `json:"origin_observation"`
	ReplayVerified   bool                 `json:"origin_replay_verified"`
	ModelRequested   bool                 `json:"new_construction_model_requested"`
	Feedback         *JointFeedbackUpdate `json:"feedback,omitempty"`
	Loop             *JointLoop           `json:"construction_loop,omitempty"`
	Holdout          *AssemblyHoldout     `json:"origin_holdout,omitempty"`
	Failure          string               `json:"failure,omitempty"`
	Scope            string               `json:"scope"`
}

type AssemblyHoldout struct {
	Observation   Summary `json:"observation"`
	NewModelCalls int     `json:"new_model_calls"`
}

// ConstructFromAssembly connects source-owned assembly to the existing joint loop.
// It does not resume an old ranking: each construction round is a fresh observation.
func ConstructFromAssembly(ctx context.Context, o Options, request AssemblyConstructionRequest) (r AssemblyConstruction, runError error) {
	r = AssemblyConstruction{Schema: "gooo/assembly-construction/v1", ModelRequested: o.Model != "",
		Scope: "original assembly retained and replayed; source cases translated without changing expectations; Gooo selects original caller counterexamples; existing bounded joint construction restarts each round; model omission is deterministic and does not inherit the origin model; adaptive cases influence selection; optional holdout runs after selection; training exposure is unknown"}
	if ctx == nil {
		return r, fmt.Errorf("assembly construction requires a context")
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if request.MaxProgramBudget < 1 || request.MaxProgramBudget > 64 || request.MaxRounds < 1 || request.MaxRounds > 16 {
		return r, fmt.Errorf("assembly construction requires a program budget of 1..64 and 1..16 rounds")
	}
	files, origin, err := readAssemblyOrigin(request.Assembly)
	if err != nil {
		return r, err
	}
	preflight, err := readAssemblyPreflight(files["preflight.json"], origin.SourceSHA, false)
	if err != nil {
		return r, err
	}
	if preflight.ActivityID != origin.ActivityID {
		return r, fmt.Errorf("assembly context activity differs from preflight")
	}
	model := origin.Observation.ModelCalls == 1
	if model {
		preflight, err = readAssemblyPreflight(files["preflight.json"], origin.SourceSHA, true)
		if err != nil {
			return r, err
		}
	}
	original, err := readAssemblyExecution(files["assembly.json"], preflight, true, model)
	if err != nil {
		return r, err
	}
	observed, err := summarize(files["assembly.json"], "source-model-assembly", origin.Observation.Mode)
	if err != nil {
		return r, err
	}
	observed.ReplayVerified = true
	if observed != origin.Observation || original.Composition.GeneratedSHA != origin.GeneratedSHA {
		return r, fmt.Errorf("assembly context observation differs from original native result")
	}
	current, entry, err := sourceAssemblyCases(files["assembly.json"], origin.ActivityID)
	if err != nil {
		return r, err
	}
	var holdout []byte
	if request.HoldoutCases != "" {
		holdout, err = os.ReadFile(request.HoldoutCases)
		if err != nil {
			return r, err
		}
		if _, err = readJointCases(holdout, true); err != nil {
			return r, err
		}
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return r, err
	}
	defer func() {
		if runError != nil {
			r.Failure = runError.Error()
		}
		runError = errors.Join(runError, save(filepath.Join(root, "assembly-construction.json"), r))
	}()
	r.OriginContextSHA, r.Origin = "sha256:"+jointDigest(files["next-context.json"]), observed
	for name, raw := range files {
		path := filepath.Join(root, "origin", name)
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return r, err
		}
		if err = write(path, raw); err != nil {
			return r, err
		}
	}
	if err = write(filepath.Join(root, "source-cases.json"), current); err != nil {
		return r, err
	}
	if holdout != nil {
		if err = write(filepath.Join(root, "holdout-cases.json"), holdout); err != nil {
			return r, err
		}
	}
	raw, err := retainedAssemblyCommand(ctx, o.Compiler, root, "origin-replay.json", "body-compose",
		"--source", filepath.Join(root, "origin/source.gooo"), "--cases", filepath.Join(root, "origin/cases.json"),
		"--composition", filepath.Join(root, "origin/composition/composition.json"))
	if err != nil {
		return r, err
	}
	replayed, err := readAssemblyExecution(raw, preflight, false, model)
	if err != nil {
		return r, err
	}
	if original.Composition.GeneratedSHA != replayed.Composition.GeneratedSHA ||
		!reflect.DeepEqual(original.Runtime.Traces, replayed.Runtime.Traces) {
		return r, fmt.Errorf("retained assembly caller observation does not replay")
	}
	r.ReplayVerified = true
	advice, _, err := runAssemblyPolicy(ctx, o, root, "next", "assembly-next", "assemblynext", assemblyNextInput{
		observed.NamedPassed, observed.NamedTotal, observed.FieldsPassed, observed.FieldsTotal,
		observed.SelectionPassed, observed.SelectionTotal})
	if err != nil {
		return r, err
	}
	r.StopReason = advice.Action
	if advice.Action != "add-counterexamples-to-construction" {
		if holdout != nil {
			r.Holdout, err = replayAssemblyHoldout(ctx, o, root, preflight, model)
		}
		return r, err
	}
	r.Feedback, err = prepareJointFeedback(ctx, o, root, "origin-feedback", current, files["cases.json"], files["assembly.json"])
	if err != nil {
		return r, err
	}
	if !r.Feedback.Prepared {
		r.StopReason = r.Feedback.StopReason
		if holdout != nil {
			r.Holdout, err = replayAssemblyHoldout(ctx, o, root, preflight, model)
		}
		return r, err
	}
	joint := JointRequest{Source: filepath.Join(root, "origin/source.gooo"), Entry: entry,
		ConstructionCases: filepath.Join(root, r.Feedback.NextCasesFile), EvaluationCases: filepath.Join(root, "origin/cases.json"),
		MaxProgramBudget: request.MaxProgramBudget, MaxRounds: request.MaxRounds}
	if holdout != nil {
		joint.HoldoutCases = filepath.Join(root, "holdout-cases.json")
	}
	loop, err := ConstructJoint(ctx, Options{Compiler: o.Compiler, Model: o.Model, Out: filepath.Join(root, "construction")}, joint)
	r.Loop = &loop
	r.Feedback.Consumed = loop.InitialCasesConsumed
	if loop.StopReason != "" {
		r.StopReason = loop.StopReason
	} else if err != nil {
		r.StopReason = "construction-failed"
	}
	return r, err
}

func replayAssemblyHoldout(ctx context.Context, o Options, root string, p assemblyPreflight, model bool) (*AssemblyHoldout, error) {
	raw, err := retainedAssemblyCommand(ctx, o.Compiler, root, "origin-holdout.json", "body-compose",
		"--source", filepath.Join(root, "origin/source.gooo"), "--cases", filepath.Join(root, "holdout-cases.json"),
		"--composition", filepath.Join(root, "origin/composition/composition.json"))
	if err != nil {
		return nil, err
	}
	if _, err = readAssemblyExecution(raw, p, false, model); err != nil {
		return nil, err
	}
	summary, err := summarize(raw, "retained-assembly-holdout", "saved-replay")
	if err != nil {
		return nil, err
	}
	summary.ReplayVerified = true
	return &AssemblyHoldout{Observation: summary, NewModelCalls: 0}, nil
}
