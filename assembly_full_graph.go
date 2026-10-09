package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const graphContextSchema = "gooo/assembly-next-context/v2"

var graphOriginNames = []string{"source.gooo", "cases.json", "plan.json", "profiles.json", "assembly.json", "replay.json",
	"composition/generated.go", "composition/composition.json", "follow-up/graph-next.gooo",
	"follow-up/inputs.json", "follow-up/execution.json", "follow-up/replay.json"}

type graphAssemblyProfiles struct {
	Schema          string `json:"schema"`
	RecordRequested bool   `json:"record_model_requested"`
	MetadataSHA     string `json:"record_model_metadata_sha256,omitempty"`
}

type GraphAssemblyReport struct {
	Schema              string              `json:"schema"`
	Entry               string              `json:"entry_activity"`
	Observation         Summary             `json:"observation"`
	Bodies              []GraphAssemblyBody `json:"bodies"`
	Next                policyAdvice        `json:"next"`
	FollowUp            Summary             `json:"follow_up_observation"`
	GeneratedSHA        string              `json:"generated_sha256"`
	NextContext         string              `json:"next_context"`
	ReplayNewModelCalls int                 `json:"replay_new_model_calls"`
	Scope               string              `json:"scope"`
}

// AssembleGraph retains the native plan and all graph bodies before feeding caller
// failures to construction. Per-step model contexts belong to their checkpoints.
func AssembleGraph(ctx context.Context, o Options, request AssemblyRequest) (report GraphAssemblyReport, err error) {
	if ctx == nil {
		return report, fmt.Errorf("graph assembly requires a context")
	}
	if err = ctx.Err(); err != nil {
		return report, err
	}
	if request.Source == "" || request.Entry == "" || request.Cases == "" || request.AssemblyActivity != "" {
		return report, fmt.Errorf("graph assembly requires --source, --entry and --cases; omit --assembly-activity")
	}
	source, err := os.ReadFile(request.Source)
	if err != nil {
		return report, err
	}
	cases, err := os.ReadFile(request.Cases)
	if err != nil {
		return report, err
	}
	if _, err = readJointCases(cases, true); err != nil {
		return report, err
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return report, err
	}
	for name, raw := range map[string][]byte{"source.gooo": source, "cases.json": cases} {
		if err = write(filepath.Join(root, name), raw); err != nil {
			return report, err
		}
	}
	raw, err := retainedAssemblyCommand(ctx, o.Compiler, root, "plan.json", "body-plan", "--source",
		filepath.Join(root, "source.gooo"), "--entry", request.Entry, "--json")
	if err != nil {
		return report, err
	}
	plan, err := readGraphInspection(raw, "sha256:"+jointDigest(source), request.Entry)
	if err != nil {
		return report, err
	}
	model, err := prepareModel(o.Model, root)
	if err != nil {
		return report, err
	}
	profiles := graphAssemblyProfiles{Schema: "gooo/graph-assembly-profiles/v1", RecordRequested: model != ""}
	if model != "" {
		metadata, e := os.ReadFile(model)
		if e != nil {
			return report, e
		}
		profiles.MetadataSHA = jointDigest(metadata)
	}
	if err = save(filepath.Join(root, "profiles.json"), profiles); err != nil {
		return report, err
	}
	args := []string{"body-compose", "--source", filepath.Join(root, "source.gooo"), "--entry", request.Entry,
		graphCaseFlag(cases), filepath.Join(root, "cases.json"), "--out", filepath.Join(root, "composition")}
	if model != "" {
		args = append(args, "--model", model)
	}
	raw, err = retainedAssemblyCommand(ctx, o.Compiler, root, "assembly.json", args...)
	if err != nil {
		return report, err
	}
	original, bodies, err := readGraphExecution(raw, plan, profiles, true)
	if err != nil {
		return report, err
	}
	observation, err := summarizeGraph(raw, bodies)
	if err != nil {
		return report, err
	}
	replay, err := retainedAssemblyCommand(ctx, o.Compiler, root, "replay.json", "body-compose",
		"--source", filepath.Join(root, "source.gooo"), graphCaseFlag(cases), filepath.Join(root, "cases.json"),
		"--composition", filepath.Join(root, "composition/composition.json"))
	if err != nil {
		return report, err
	}
	replayed, replayBodies, err := readGraphExecution(replay, plan, profiles, false)
	if err != nil {
		return report, err
	}
	if err = sameGraphReplay(raw, replay, original, replayed); err != nil || !reflect.DeepEqual(bodies, replayBodies) {
		if err == nil {
			err = fmt.Errorf("graph replay changed body observations")
		}
		return report, err
	}
	observation.ReplayVerified = true
	next, followUp, err := runGraphAssemblyPolicy(ctx, o, root, "follow-up", observation, bodies)
	if err != nil {
		return report, err
	}
	if followUp.CompilerSource != observation.CompilerSource {
		return report, fmt.Errorf("graph and Gooo follow-up used different compiler sources")
	}
	report = GraphAssemblyReport{Schema: "gooo/ecosystem-graph-assembly/v1", Entry: request.Entry, Observation: observation,
		Bodies: bodies, Next: next, FollowUp: followUp, GeneratedSHA: original.Composition.GeneratedSHA, NextContext: "next-context.json",
		Scope: "native graph plan, multiple bodies, called preparation and caller roots; source-local selection and caller expectations remain separate; optional record model contexts are checkpoint-specific; source-fill is deterministic; saved replay has zero fresh inference; finite observations do not establish general accuracy"}
	c := assemblyContext{Schema: graphContextSchema, Entry: request.Entry, SourceSHA: plan.SourceSHA,
		GeneratedSHA: report.GeneratedSHA, Observation: observation, Next: next, Failures: []assemblyFailure{}, Scope: report.Scope}
	if err = assemblyContextFailures(&c, original); err != nil {
		return report, err
	}
	if c.Mismatches != observation.NamedTotal-observation.NamedPassed {
		return report, fmt.Errorf("graph mismatch recount differs")
	}
	if err = saveAssemblyContextArtifacts(root, report.NextContext, c, graphOriginNames); err != nil {
		return report, err
	}
	return report, save(filepath.Join(root, "report.json"), report)
}

func graphCaseFlag(cases []byte) string {
	var doc struct{ Schema string }
	_ = json.Unmarshal(cases, &doc)
	if doc.Schema == "gooo/body-composition-inputs/v1" {
		return "--inputs"
	}
	return "--cases"
}

func summarizeGraph(raw []byte, bodies []GraphAssemblyBody) (Summary, error) {
	s, err := summarize(raw, "source-graph-assembly", "native-graph")
	if err != nil {
		return s, err
	}
	s.ModelCalls = 0
	for _, body := range bodies {
		s.ModelCalls += body.ModelCalls
	}
	return s, nil
}

func runGraphAssemblyPolicy(ctx context.Context, o Options, root, directory string, observed Summary, bodies []GraphAssemblyBody) (policyAdvice, Summary, error) {
	passed, total, unsupported := 0, 0, 0
	for _, body := range bodies {
		passed += body.SourcePassed
		total += body.SourceTotal
		if body.Kind == "typed_paths" {
			unsupported++
		}
	}
	return runAssemblyPolicy(ctx, o, root, directory, "graph-next", "graphnext", map[string]int{
		"caller_passed": observed.NamedPassed, "caller_total": observed.NamedTotal,
		"source_passed": passed, "source_total": total, "bodies": len(bodies), "unsupported_bodies": unsupported})
}

func sameGraphReplay(originalRaw, replayRaw []byte, original, replay result) error {
	var a, b struct{ Composition json.RawMessage }
	if err := json.Unmarshal(originalRaw, &a); err != nil {
		return err
	}
	if err := json.Unmarshal(replayRaw, &b); err != nil {
		return err
	}
	x, err := jointCanonical(a.Composition)
	if err != nil {
		return err
	}
	y, err := jointCanonical(b.Composition)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(x, y) || original.Runtime.Source != replay.Runtime.Source || !reflect.DeepEqual(original.Runtime.Traces, replay.Runtime.Traces) {
		return fmt.Errorf("saved native graph changed its plan, checkpoints, program or caller outputs")
	}
	return nil
}
