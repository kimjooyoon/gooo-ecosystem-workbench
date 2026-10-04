package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndependentCounterKeepsExactInt64Values(t *testing.T) {
	raw := []byte(`{"runtime":{"model_calls":0,"finite_passed":0,"finite_total":1,"traces":[{"deliveries":[{"actual":9223372036854775807,"expected":9223372036854775806}]}]}}`)
	s, e := summarize(raw, "stdlib", "deterministic")
	if e != nil || s.NamedPassed != 0 || s.NamedTotal != 1 {
		t.Fatal(s, e)
	}
}
func TestIndependentCounterRejectsFalseReportedPasses(t *testing.T) {
	raw := []byte(`{"runtime":{"model_calls":0,"finite_passed":1,"finite_total":1,"traces":[{"deliveries":[{"actual":{"a":"wrong"},"expected":{"a":"right"}}]}]}}`)
	if _, e := summarize(raw, "diagnostics", "model"); e == nil {
		t.Fatal("false runtime pass accepted")
	}
}
func TestEmbeddedSourceAndFrozenModel(t *testing.T) {
	b, e := assets.ReadFile("recipes/stdlib.gooo")
	if e != nil || strings.Count(string(b), "activity ") != 13 {
		t.Fatal("standard Gooo sources missing", e)
	}
	b, e = assets.ReadFile("models/shared-qat/weights.bin")
	if e != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != "049081cc6dc62f3c4230f72e601c1fd421c514fce9cfdf7bc1eb412ca4c5514b" {
		t.Fatal("frozen model identity changed", e)
	}
}
func TestOutputCannotOverwriteExistingDirectory(t *testing.T) {
	if _, e := newOutput(t.TempDir()); e == nil {
		t.Fatal("existing output accepted")
	}
}

func TestSnapshotRecountsTheRetainedPartialProgram(t *testing.T) {
	raw, err := os.ReadFile("examples/partial-composition.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadSnapshot(raw)
	if err != nil || s.Passed != 17 || s.Total != 21 || s.Rejected != 1 || !strings.Contains(s.Detail, "deferred") {
		t.Fatal(s, err)
	}
	if _, err = ReadSnapshot([]byte(`{"detail":"missing observations"}`)); err == nil {
		t.Fatal("missing counters became zero observations")
	}
}
func TestSnapshotBoundsDetailAndKeepsInputIdentity(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"runtime": map[string]any{
		"model_calls": 0, "finite_passed": 0, "finite_total": 1,
		"traces": []any{map[string]any{"deliveries": []any{map[string]any{
			"actual":   map[string]string{"a": strings.Repeat("x", 2000)},
			"expected": map[string]string{"a": "right"},
		}}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadSnapshot(raw)
	if err != nil || len(s.Detail) > 1024 || s.Passed != 0 || s.Total != 1 ||
		!strings.Contains(s.Detail, `"detail_limited":true`) ||
		s.InputSHA != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal(s, err)
	}
}

func TestSnapshotOrdersFieldGaps(t *testing.T) {
	raw := []byte(`{"runtime":{"model_calls":0,"finite_passed":0,"finite_total":1,"traces":[{"deliveries":[{"actual":{"z":"wrong","a":"wrong"},"expected":{"z":"right","a":"right"}}]}]}}`)
	s, err := ReadSnapshot(raw)
	a, z := strings.Index(s.Detail, `"field":"a"`), strings.Index(s.Detail, `"field":"z"`)
	if err != nil || a < 0 || z < 0 || a >= z {
		t.Fatal(s, err)
	}
}

func TestNativeRecipesScaffoldAndDiagnostics(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual Gooo/native verification")
	}
	root := t.TempDir()
	ctx := context.Background()
	summaries, e := Verify(ctx, Options{Compiler: compiler, Model: "builtin", Out: filepath.Join(root, "verify")})
	if e != nil {
		t.Fatal(e)
	}
	if len(summaries) != 5 {
		t.Fatal(summaries)
	}
	for _, s := range summaries {
		if s.NamedPassed != s.NamedTotal || !s.ReplayVerified {
			t.Fatal(s)
		}
	}
	for _, profile := range []string{"record", "scalar"} {
		p, e := Scaffold(ctx, Options{Compiler: compiler, Model: "builtin", Out: filepath.Join(root, profile)}, profile)
		if e != nil || p.Filename != "main.gooo" || !strings.Contains(p.Source, "activity Identity") {
			t.Fatal(p, e)
		}
	}
	raw, e := Diagnose(ctx, Options{Compiler: compiler, Model: "builtin", Out: filepath.Join(root, "diagnose")}, Snapshot{Passed: 12, Total: 15, Rejected: 1, Detail: "missing reason field"})
	if e != nil || !strings.Contains(string(raw), "repair-and-replay") || !strings.Contains(string(raw), "partial") {
		t.Fatal(string(raw), e)
	}
	_, e = Scaffold(ctx, Options{Compiler: compiler, Out: filepath.Join(root, "unknown")}, "unknown")
	if e == nil {
		t.Fatal("unknown profile created a project")
	}
}
