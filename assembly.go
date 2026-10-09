package workbench

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// AssemblyRequest names a root record assembler, its graph entry and caller cases.
// AssemblyActivity defaults to Entry for existing single-activity callers.
type AssemblyRequest struct{ Source, Entry, Cases, AssemblyActivity string }

type AssemblyReport struct {
	Schema       string            `json:"schema"`
	Route        policyAdvice      `json:"route"`
	Preflight    assemblyPreflight `json:"preflight"`
	Policy       Summary           `json:"policy_observation"`
	Observation  Summary           `json:"observation"`
	Next         policyAdvice      `json:"next"`
	FollowUp     Summary           `json:"follow_up_observation"`
	NextContext  string            `json:"next_context"`
	GeneratedSHA string            `json:"generated_sha256"`
	Artifacts    map[string]string `json:"artifacts"`
	Scope        string            `json:"scope"`
}

// Assemble connects preflight, a Gooo routing rule, native assembly and saved replay.
// The caller's source and cases are copied byte for byte into a new output directory.
func Assemble(ctx context.Context, o Options, request AssemblyRequest) (AssemblyReport, error) {
	var report AssemblyReport
	if request.Source == "" || request.Entry == "" || request.Cases == "" {
		return report, fmt.Errorf("assemble requires --source, --entry and --cases; use assemble --help")
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return report, err
	}
	for name, path := range map[string]string{"source.gooo": request.Source, "cases.json": request.Cases} {
		raw, e := os.ReadFile(path)
		if e != nil {
			return report, e
		}
		if e = write(filepath.Join(root, name), raw); e != nil {
			return report, e
		}
	}
	source, err := os.ReadFile(filepath.Join(root, "source.gooo"))
	if err != nil {
		return report, err
	}
	sourceSHA := fmt.Sprintf("sha256:%x", sha256.Sum256(source))
	model, err := prepareModel(o.Model, root)
	if err != nil {
		return report, err
	}
	activity := request.AssemblyActivity
	if activity == "" {
		activity = request.Entry
	}
	args := []string{"body-context", "--activity", activity}
	if model != "" {
		args = append(args, "--model", model)
	}
	raw, err := retainedAssemblyCommand(ctx, o.Compiler, root, "preflight.json", append(args, filepath.Join(root, "source.gooo"))...)
	if err != nil {
		return report, err
	}
	preflight, err := readAssemblyPreflight(raw, sourceSHA, model != "")
	if err != nil {
		return report, err
	}
	route, policy, err := runAssemblyRoute(ctx, o, root, preflight, model != "")
	if err != nil {
		return report, err
	}
	if route.Action != "model" && route.Action != "fixed" {
		return report, fmt.Errorf("%s: %s; original preflight retained in %s", route.Code, route.Message, root)
	}
	if route.Action == "fixed" {
		model = ""
	}
	args = []string{"body-compose", "--source", filepath.Join(root, "source.gooo"), "--entry", request.Entry,
		"--cases", filepath.Join(root, "cases.json"), "--out", filepath.Join(root, "composition")}
	if model != "" {
		args = append(args, "--model", model)
	}
	raw, err = retainedAssemblyCommand(ctx, o.Compiler, root, "assembly.json", args...)
	if err != nil {
		return report, err
	}
	original, err := readAssemblyExecution(raw, preflight, true, route.Action == "model")
	if err != nil {
		return report, err
	}
	if _, entry, e := sourceAssemblyCases(raw, preflight.ActivityID); e != nil || entry != request.Entry {
		if e != nil {
			return report, e
		}
		return report, fmt.Errorf("assembly graph entry differs from the requested activity")
	}
	observation, err := summarize(raw, "source-model-assembly", route.Action)
	if err != nil {
		return report, err
	}
	args = []string{"body-compose", "--source", filepath.Join(root, "source.gooo"), "--cases", filepath.Join(root, "cases.json"),
		"--composition", filepath.Join(root, "composition", "composition.json")}
	raw, err = retainedAssemblyCommand(ctx, o.Compiler, root, "replay.json", args...)
	if err != nil {
		return report, err
	}
	replayed, err := readAssemblyExecution(raw, preflight, false, route.Action == "model")
	if err != nil {
		return report, err
	}
	if original.Composition.GeneratedSHA != replayed.Composition.GeneratedSHA ||
		original.Runtime.Source != replayed.Runtime.Source || !reflect.DeepEqual(original.Runtime.Traces, replayed.Runtime.Traces) {
		return report, fmt.Errorf("saved assembly replay changed source, selected program or observed outputs")
	}
	observation.ReplayVerified = true
	if policy.CompilerSource != observation.CompilerSource {
		return report, fmt.Errorf("routing policy and assembly used different compiler sources")
	}
	next, followUp, err := runAssemblyPolicy(ctx, o, root, "follow-up", "assembly-next", "assemblynext", assemblyNextInput{
		observation.NamedPassed, observation.NamedTotal, observation.FieldsPassed, observation.FieldsTotal,
		observation.SelectionPassed, observation.SelectionTotal})
	if err != nil {
		return report, err
	}
	if followUp.CompilerSource != observation.CompilerSource {
		return report, fmt.Errorf("follow-up policy and assembly used different compiler sources")
	}
	// The short report points to the original graph text in preflight.json.
	preflight.Context.Text = ""
	report = AssemblyReport{Schema: "gooo/ecosystem-source-model-assembly/v1", Route: route, Preflight: preflight, Policy: policy,
		Observation: observation, Next: next, FollowUp: followUp, NextContext: "next-context.json",
		GeneratedSHA: original.Composition.GeneratedSHA,
		Artifacts: map[string]string{"preflight": "preflight.json", "routing": "routing/execution.json", "routing_replay": "routing/replay.json",
			"assembly": "assembly.json", "replay": "replay.json", "generated_go": "composition/generated.go", "saved_composition": "composition/composition.json"},
		Scope: "one source-owned root record assembler with fixed bound consumers and pure helpers; representation readiness, source selection cases and caller native cases are separate; saved replay performs zero fresh inference; no general accuracy claim"}
	if err = saveAssemblyContext(root, report, original); err != nil {
		return report, err
	}
	return report, save(filepath.Join(root, "report.json"), report)
}

func retainedAssemblyCommand(ctx context.Context, compiler, root, name string, args ...string) ([]byte, error) {
	raw, err := command(ctx, compiler, args...)
	if len(raw) > 0 {
		if e := write(filepath.Join(root, name), raw); e != nil {
			return raw, e
		}
	}
	if err != nil {
		_ = write(filepath.Join(root, name+".error"), []byte(err.Error()+"\n"))
	}
	return raw, err
}

func runAssemblyRoute(ctx context.Context, o Options, root string, p assemblyPreflight, requested bool) (policyAdvice, Summary, error) {
	input := map[string]any{"requested": requested, "status": "", "reason": ""}
	if p.Compatibility != nil {
		input["status"], input["reason"] = p.Compatibility.Status, p.Compatibility.Reason
	}
	return runAssemblyPolicy(ctx, o, root, "routing", "assembly-route", "assemblyroute", input)
}

func runAssemblyPolicy(ctx context.Context, o Options, root, directory, recipe, pkg string, input any) (policyAdvice, Summary, error) {
	var advice policyAdvice
	var summary Summary
	dir := filepath.Join(root, directory)
	if err := os.Mkdir(dir, 0755); err != nil {
		return advice, summary, err
	}
	if err := save(filepath.Join(dir, "inputs.json"), map[string]any{"schema": "gooo/body-composition-inputs/v1",
		"inputs": []any{map[string]any{pkg + ":Next": input}}}); err != nil {
		return advice, summary, err
	}
	source, err := assets.ReadFile("recipes/" + recipe + ".gooo")
	if err != nil {
		return advice, summary, err
	}
	if err = write(filepath.Join(dir, recipe+".gooo"), source); err != nil {
		return advice, summary, err
	}
	if err = save(filepath.Join(dir, "gooo.workspace.json"), map[string]any{"schema": "gooo/package-workspace-manifest/v1",
		"entry":    map[string]string{"package_path": pkg, "activity": "Next"},
		"packages": []any{map[string]any{"path": pkg, "sources": []string{recipe + ".gooo"}}}}); err != nil {
		return advice, summary, err
	}
	var originalSHA string
	for _, replay := range []bool{false, true} {
		raw, r, program, e := executeInputPolicy(ctx, o, dir, "", replay)
		if e != nil {
			return advice, summary, e
		}
		values, e := policyAdviceValues(r, 1)
		if e != nil {
			return advice, summary, e
		}
		if !replay {
			advice, originalSHA = values[0], program
			summary, e = summarize(raw, recipe, "fixed")
			if e != nil {
				return advice, summary, e
			}
		} else if advice != values[0] || originalSHA != program {
			return advice, summary, fmt.Errorf("saved Gooo %s policy changed advice or selected program", recipe)
		}
	}
	summary.ReplayVerified = true
	return advice, summary, nil
}
