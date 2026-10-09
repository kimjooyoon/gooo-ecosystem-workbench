package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssemblyPreflightRequiresExplicitZeroAndSourceBinding(t *testing.T) {
	raw := []byte(`{"schema":"gooo/record-assembly-input-export/v1","original_source_sha256":"sha256:` + strings.Repeat("a", 64) + `","contract_sha256":"sha256:` + strings.Repeat("b", 64) + `","activity_id":"gooo://example/action","model_predictions":0,"candidate_tests":0}`)
	if _, err := readAssemblyPreflight(raw, "sha256:"+strings.Repeat("a", 64), false); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(map[string]any){
		func(v map[string]any) { delete(v, "model_predictions") },
		func(v map[string]any) { delete(v, "candidate_tests") },
		func(v map[string]any) { v["candidate_tests"] = 1 },
		func(v map[string]any) { v["model_predictions"] = -1 },
		func(v map[string]any) { v["original_source_sha256"] = "sha256:" + strings.Repeat("c", 64) },
		func(v map[string]any) { v["activity_id"] = "" },
	} {
		var value map[string]any
		_ = json.Unmarshal(raw, &value)
		mutation(value)
		changed, _ := json.Marshal(value)
		if _, err := readAssemblyPreflight(changed, "sha256:"+strings.Repeat("a", 64), false); err == nil {
			t.Fatal("accepted absent, nonzero or unbound preflight", string(changed))
		}
	}
	if _, err := readAssemblyPreflight(raw, "sha256:"+strings.Repeat("a", 64), true); err == nil {
		t.Fatal("requested model acquired compatibility from an absent field")
	}
}

func TestNativeAssemblyRoutesAndSavedReplay(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native source/model assembly")
	}
	for _, test := range []struct {
		source, model, route, code string
		calls                      int
	}{
		{"source.gooo", "", "fixed", "no-model", 0},
		{"source.gooo", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json", "model", "source-encoded", 1},
		{"english.gooo", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json", "model", "source-encoded", 1},
		{"one-choice.gooo", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json", "fixed", "representation-declined", 0},
	} {
		t.Run(test.source+"-"+test.route, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "assembly")
			report, err := Assemble(context.Background(), Options{Compiler: compiler, Model: test.model, Out: out},
				AssemblyRequest{Source: "examples/model-assembly/" + test.source, Entry: "Describe", Cases: "examples/model-assembly/cases.json"})
			if err != nil {
				t.Fatal(err)
			}
			if report.Route.Action != test.route || report.Route.Code != test.code || report.Observation.ModelCalls != test.calls ||
				!report.Observation.ReplayVerified || report.Observation.NamedPassed != 4 || report.Observation.NamedTotal != 4 ||
				report.Observation.FieldsPassed != 12 || report.Observation.FieldsTotal != 12 {
				t.Fatalf("route or finite observation differs: %+v", report)
			}
			if report.Policy.ModelCalls != 0 || report.Policy.NamedTotal != 0 || !report.Policy.ReplayVerified {
				t.Fatal("route policy acquired inference or an input accuracy score", report.Policy)
			}
			if report.Next.Code != "observed-complete" || report.Next.Action != "observe-new-inputs" ||
				report.FollowUp.ModelCalls != 0 || report.FollowUp.NamedTotal != 0 || !report.FollowUp.ReplayVerified {
				t.Fatal("complete caller result lost its Gooo follow-up", report)
			}
			if report.Preflight.Context.Text != "" {
				t.Fatal("short report embedded the original input graph")
			}
			raw, err := os.ReadFile(filepath.Join(out, "assembly.json"))
			if err != nil || !strings.Contains(string(raw), "9007199254740993") {
				t.Fatal("original execution lost the int64 value", err)
			}
			if _, err := Assemble(context.Background(), Options{Compiler: compiler, Out: out},
				AssemblyRequest{Source: "examples/model-assembly/source.gooo", Entry: "Describe", Cases: "examples/model-assembly/cases.json"}); err == nil {
				t.Fatal("existing output was reused")
			}
		})
	}
}

func TestNativeAssemblyStopsForInvalidModelAndUnknownRoute(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native preflight")
	}
	out := filepath.Join(t.TempDir(), "invalid-model")
	_, err := Assemble(context.Background(), Options{Compiler: compiler, Model: filepath.Join(out, "missing.json"), Out: out},
		AssemblyRequest{Source: "examples/model-assembly/source.gooo", Entry: "Describe", Cases: "examples/model-assembly/cases.json"})
	if err == nil {
		t.Fatal("invalid model silently became deterministic assembly")
	}
	if _, err := os.Stat(filepath.Join(out, "assembly.json")); !os.IsNotExist(err) {
		t.Fatal("invalid model reached assembly")
	}
	if raw, err := os.ReadFile(filepath.Join(out, "preflight.json")); err != nil || !strings.Contains(string(raw), "FAIL_CLOSED") {
		t.Fatal("original failed preflight was lost", err)
	}
	var p assemblyPreflight
	_ = json.Unmarshal([]byte(`{"model_compatibility":{"status":"FUTURE_STATUS","reason":"UNRECOGNIZED"}}`), &p)
	root := t.TempDir()
	advice, summary, err := runAssemblyRoute(context.Background(), Options{Compiler: compiler}, root, p, true)
	if err != nil || advice.Action != "stop" || summary.NamedTotal != 0 || summary.ModelCalls != 0 {
		t.Fatal("unknown preflight became a runnable route", advice, summary, err)
	}
}

func TestAssemblyExecutionKeepsPresentRuntimeCounters(t *testing.T) {
	sourceSHA := "sha256:" + strings.Repeat("a", 64)
	p := assemblyPreflight{SourceSHA: sourceSHA, ActivityID: "gooo://example/action"}
	raw := []byte(`{"generated_now":true,"composition":{"original_source_sha256":"` + sourceSHA + `","generated_sha256":"sha256:` + strings.Repeat("b", 64) + `","steps":[{"generation":{"report":{"activity_id":"gooo://example/action","record_assembly":{"model_calls":0,"fields_passed":0,"fields_total":0}}}}]},"runtime":{"stage":"COMPLETE","model_calls":0,"finite_passed":0,"finite_total":0,"projection_replayed":true,"runtime_replayed":true}}`)
	if _, err := readAssemblyExecution(raw, p, true, false); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"model_calls", "finite_passed", "finite_total", "projection_replayed", "runtime_replayed"} {
		var value map[string]any
		_ = json.Unmarshal(raw, &value)
		delete(value["runtime"].(map[string]any), field)
		changed, _ := json.Marshal(value)
		if _, err := readAssemblyExecution(changed, p, true, false); err == nil {
			t.Fatal("missing execution observation became zero or complete", field)
		}
	}
	if _, err := readAssemblyExecution(raw, p, false, false); err == nil {
		t.Fatal("fresh generation presented as saved replay")
	}
	for _, field := range []string{"fields_passed", "fields_total"} {
		changed := strings.Replace(string(raw), `,"`+field+`":0`, "", 1)
		if _, err := readAssemblyExecution([]byte(changed), p, true, false); err == nil {
			t.Fatal("absent selection observation became zero", field)
		}
	}
}
