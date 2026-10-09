package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func constructGraphFromAssembly(ctx context.Context, o Options, request AssemblyConstructionRequest,
	files map[string][]byte, origin assemblyContext) (r AssemblyConstruction, runError error) {
	r = AssemblyConstruction{Schema: "gooo/assembly-construction/v2", ModelRequested: o.Model != "",
		Scope: "saved native graph replayed before feedback; Gooo selects complete original failing caller rows; source-local examples stay in native contracts and are never inverted into caller inputs; model omission starts each construction deterministically and never inherits the origin model; bounded rounds count repeated attempts; optional final holdout is not fed back; training exposure is unknown"}
	p, err := readGraphInspection(files["plan.json"], origin.SourceSHA, origin.Entry)
	if err != nil {
		return r, err
	}
	var profiles graphAssemblyProfiles
	if err = json.Unmarshal(files["profiles.json"], &profiles); err != nil {
		return r, err
	}
	original, bodies, err := readGraphExecution(files["assembly.json"], p, profiles, true)
	if err != nil {
		return r, err
	}
	observed, err := summarizeGraph(files["assembly.json"], bodies)
	if err != nil {
		return r, err
	}
	observed.ReplayVerified = true
	if observed != origin.Observation || original.Composition.GeneratedSHA != origin.GeneratedSHA {
		return r, fmt.Errorf("saved graph context differs from original observation")
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
	if err = write(filepath.Join(root, "caller-history.json"), emptyGraphHistory); err != nil {
		return r, err
	}
	if holdout != nil {
		if err = write(filepath.Join(root, "holdout-cases.json"), holdout); err != nil {
			return r, err
		}
	}
	raw, err := retainedAssemblyCommand(ctx, o.Compiler, root, "origin-replay.json", "body-compose",
		"--source", filepath.Join(root, "origin/source.gooo"), graphCaseFlag(files["cases.json"]), filepath.Join(root, "origin/cases.json"),
		"--composition", filepath.Join(root, "origin/composition/composition.json"))
	if err != nil {
		return r, err
	}
	replayed, _, err := readGraphExecution(raw, p, profiles, false)
	if err != nil {
		return r, err
	}
	if err = sameGraphReplay(files["assembly.json"], raw, original, replayed); err != nil {
		return r, err
	}
	r.ReplayVerified = true
	advice, policy, err := runGraphAssemblyPolicy(ctx, o, root, "next", observed, bodies)
	if err != nil {
		return r, err
	}
	if policy.CompilerSource != observed.CompilerSource {
		return r, fmt.Errorf("saved graph and next policy used different compiler sources")
	}
	r.StopReason = advice.Action
	if advice.Action == "add-counterexamples-to-construction" {
		r.Feedback, err = prepareInitialGraphFeedback(ctx, o, root, files["cases.json"], files["assembly.json"])
		if err != nil {
			return r, err
		}
		if r.Feedback.Prepared {
			joint := JointRequest{Source: filepath.Join(root, "origin/source.gooo"), Entry: origin.Entry,
				ConstructionCases: filepath.Join(root, r.Feedback.NextCasesFile), EvaluationCases: filepath.Join(root, "origin/cases.json"),
				MaxProgramBudget: request.MaxProgramBudget, MaxRounds: request.MaxRounds}
			if holdout != nil {
				joint.HoldoutCases = filepath.Join(root, "holdout-cases.json")
			}
			loop, e := ConstructJoint(ctx, Options{Compiler: o.Compiler, Model: o.Model, Out: filepath.Join(root, "construction")}, joint)
			r.Loop = &loop
			r.Feedback.Consumed = loop.InitialCasesConsumed
			r.StopReason = loop.StopReason
			if r.StopReason == "" && e != nil {
				r.StopReason = "construction-failed"
			}
			return r, e
		}
		r.StopReason = r.Feedback.StopReason
	}
	if holdout != nil {
		raw, err = retainedAssemblyCommand(ctx, o.Compiler, root, "origin-holdout.json", "body-compose",
			"--source", filepath.Join(root, "origin/source.gooo"), graphCaseFlag(holdout), filepath.Join(root, "holdout-cases.json"),
			"--composition", filepath.Join(root, "origin/composition/composition.json"))
		if err != nil {
			return r, err
		}
		_, heldBodies, e := readGraphExecution(raw, p, profiles, false)
		if e != nil {
			return r, e
		}
		s, e := summarizeGraph(raw, heldBodies)
		if e != nil {
			return r, e
		}
		s.ReplayVerified = true
		r.Holdout = &AssemblyHoldout{Observation: s, NewModelCalls: 0}
	}
	return r, nil
}
