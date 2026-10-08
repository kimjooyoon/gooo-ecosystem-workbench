package workbench

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompilerFailureDetailKeepsStructuredReasons(t *testing.T) {
	for _, tc := range []struct {
		stderr, stdout, want string
	}{
		{"syntax error\n", `{"error":"other"}`, "syntax error"},
		{"", `{"error":"invalid source"}`, "invalid source"},
		{"", `{"failure":"round limit is invalid"}`, "round limit is invalid"},
		{"", `{"status":"PASS","source":"source text"}`, ""},
		{"", `{"error":{"nested":"value"}}`, ""},
		{"", "truncated response", ""},
	} {
		if got := compilerFailureDetail(tc.stderr, []byte(tc.stdout)); got != tc.want {
			t.Fatalf("failure detail = %q, want %q", got, tc.want)
		}
	}
}

func TestNativeCompilerJSONFailureIncludesSourceDiagnostic(t *testing.T) {
	o := refinementOptions(t)
	raw, err := os.ReadFile("examples/calibrated-report/source.gooo")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "invalid.gooo")
	invalid := strings.Replace(string(raw), "shared_fit", "shared-fit", 1)
	if err := os.WriteFile(source, []byte(invalid), 0644); err != nil {
		t.Fatal(err)
	}
	output, err := command(context.Background(), o.Compiler, "body-context", "--activity", "Energy", "--include-plan", source)
	if err == nil || !strings.Contains(err.Error(), "search alternative requires a unique ID") || !strings.Contains(string(output), "parse.unexpected-token") {
		t.Fatal("compiler JSON diagnostic was lost", err, string(output))
	}
}
