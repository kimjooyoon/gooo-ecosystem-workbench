package workbench

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeAPIDifferenceUsesGoooDirectionRules(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual compiler and native Gooo rules")
	}
	root := t.TempDir()
	before := writeAPIWorkspace(t, filepath.Join(root, "before"), "optional", "required")
	after := writeAPIWorkspace(t, filepath.Join(root, "after"), "required", "optional")
	report, err := CompareAPI(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "comparison")}, before, after)
	if err != nil {
		t.Fatal(err)
	}
	if report.ChangeCount != 2 || report.ClassifiedChanges != 2 || !report.Observation.ReplayVerified || report.Observation.ModelCalls != 0 {
		t.Fatal(report)
	}
	seen := map[string]bool{}
	for _, event := range report.Changes {
		seen[event.Assessment.Code] = true
	}
	if !seen["input-tightened"] || !seen["output-weakened"] {
		t.Fatal(report.Changes)
	}
	for _, name := range []string{"before-interface.json", "after-interface.json", "api-diff.json", "execution.json", "replay-000.json"} {
		if _, err := os.Stat(filepath.Join(root, "comparison", name)); err != nil {
			t.Fatal(err)
		}
	}
}

func writeAPIWorkspace(t *testing.T, root, inputPresence, outputPresence string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	source := `package api
namespace api
entity Request id "api://request" fields {
 field count id "api://request/count" type integer INPUT one
}

entity Response id "api://response" fields {
 field count id "api://response/count" type integer OUTPUT one
}
activity Run(Request) -> Response
`
	source = strings.ReplaceAll(strings.ReplaceAll(source, "INPUT", inputPresence), "OUTPUT", outputPresence)
	if err := os.WriteFile(filepath.Join(root, "api.gooo"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema":"gooo/package-workspace-manifest/v1","entry":{"package_path":"api","activity":"Run"},"packages":[{"path":"api","sources":["api.gooo"]}]}`
	path := filepath.Join(root, "gooo.workspace.json")
	if err := os.WriteFile(path, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNativeAPIUnchangedSurfaceKeepsChangedSourceUnknown(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual declaration observation")
	}
	root := t.TempDir()
	before := writeAPIWorkspace(t, filepath.Join(root, "before"), "optional", "required")
	after := writeAPIWorkspace(t, filepath.Join(root, "after"), "optional", "required")
	path := filepath.Join(filepath.Dir(after), "api.gooo")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, append(raw, []byte("\n// source evidence changed\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := CompareAPI(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "result")}, before, after)
	if err != nil || report.ChangeCount != 0 || report.ClassifiedChanges != 0 || !report.SourceChanged || report.Unchanged == nil ||
		report.Unchanged.Code != "declarations-unchanged" || report.Unchanged.Action != "observe-runtime-behavior" || report.Observation.NamedTotal != 0 {
		t.Fatal(report, err)
	}
}
