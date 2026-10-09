package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeAssemblyNextRulesAreUnscoredAndReplayed(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native assembly follow-up")
	}
	for _, test := range []struct {
		input assemblyNextInput
		code  string
	}{
		{assemblyNextInput{4, 4, 12, 12, 12, 12}, "observed-complete"},
		{assemblyNextInput{3, 4, 11, 12, 12, 12}, "partial"},
		{assemblyNextInput{4, 4, 11, 12, 12, 12}, "partial"},
		{assemblyNextInput{4, 4, 12, 12, 11, 12}, "construction-partial"},
		{assemblyNextInput{}, "unobserved"},
		{assemblyNextInput{5, 4, 12, 12, 12, 12}, "inconsistent-observation"},
		{assemblyNextInput{-1, 4, 12, 12, 12, 12}, "inconsistent-observation"},
	} {
		t.Run(test.code+fmt.Sprint(test.input), func(t *testing.T) {
			advice, summary, err := runAssemblyPolicy(context.Background(), Options{Compiler: compiler}, t.TempDir(),
				"follow-up", "assembly-next", "assemblynext", test.input)
			if err != nil || advice.Code != test.code || summary.NamedTotal != 0 || summary.ModelCalls != 0 || !summary.ReplayVerified {
				t.Fatal("Gooo follow-up, scope or saved replay differs", advice, summary, err)
			}
		})
	}
}

func TestNativeAssemblyContextRetainsPartialCallerEvidence(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual partial caller assembly")
	}
	original, err := os.ReadFile("examples/model-assembly/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := bytes.Replace(original, []byte(`"해보자!"`), []byte(`"해보자?"`), 1)
	if bytes.Equal(cases, original) {
		t.Fatal("partial caller fixture was not changed")
	}
	casesPath := filepath.Join(t.TempDir(), "cases.json")
	if err = write(casesPath, cases); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "partial")
	report, err := Assemble(context.Background(), Options{Compiler: compiler, Out: out},
		AssemblyRequest{Source: "examples/model-assembly/source.gooo", Entry: "Describe", Cases: casesPath})
	if err != nil || report.Next.Code != "partial" || report.Next.Action != "add-counterexamples-to-construction" {
		t.Fatal("partial caller result did not produce follow-up", report, err)
	}
	if report.Observation.NamedPassed != 3 || report.Observation.NamedTotal != 4 || report.Observation.FieldsPassed != 11 ||
		report.Observation.SelectionPassed != 12 || report.FollowUp.NamedTotal != 0 || report.FollowUp.ModelCalls != 0 ||
		!report.FollowUp.ReplayVerified || report.FollowUp.CompilerSource != report.Observation.CompilerSource {
		t.Fatal("source, caller or policy observations were combined", report)
	}
	raw, err := os.ReadFile(filepath.Join(out, report.NextContext))
	var c assemblyContext
	if err != nil || json.Unmarshal(raw, &c) != nil || c.Next != report.Next || c.Observation != report.Observation ||
		len(c.Failures) != 1 || c.Failures[0].Pointer != "/runtime/traces/0/deliveries/0" || c.Mismatches != 1 {
		t.Fatal("next context lost original failure or advice", string(raw), err)
	}
	if len(raw) > 8000 || bytes.Contains(raw, []byte(`"text":`)) {
		t.Fatal("next context embedded a full input graph")
	}
	for _, ref := range c.Artifacts {
		data, err := os.ReadFile(filepath.Join(out, ref.Path))
		if err != nil || ref.SHA256 != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) || ref.Bytes != int64(len(data)) {
			t.Fatal("context reference differs from saved bytes", ref, err)
		}
	}
	saved, err := os.ReadFile(filepath.Join(out, "cases.json"))
	if err != nil || !bytes.Equal(saved, cases) || !bytes.Contains(saved, []byte("9007199254740993")) {
		t.Fatal("caller expectations or exact integer changed", err)
	}
}
