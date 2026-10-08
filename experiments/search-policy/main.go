// search-policy compares source-declared transitions with bounded controls.
package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	wb "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	compiler := flag.String("compiler", "gooo", "Gooo compiler executable")
	out := flag.String("out", "", "new evidence directory")
	flag.Parse()
	if *out == "" {
		return fmt.Errorf("provide a new output directory")
	}
	if err := os.Mkdir(*out, 0755); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rows := map[string]wb.RefinementReport{}
	for _, mode := range []string{"legacy", "alternatives", "mixed", "mixed-model", "round-limit", "no-alternatives"} {
		o, err := options(*compiler, *out, mode)
		if err != nil {
			return err
		}
		r, err := wb.Refine(ctx, o)
		if err != nil {
			return fmt.Errorf("%s: %w", mode, err)
		}
		rows[mode] = r
		fmt.Printf("%s: %s %d/%d -> %d/%d, rounds=%d\n", mode, r.Status, r.Initial.Passed, r.Initial.Total, r.Final.Passed, r.Final.Total, r.Rounds)
	}
	path, err := exec.LookPath(*compiler)
	if err != nil {
		return err
	}
	compilerInfo, err := buildinfo.ReadFile(path)
	if err != nil {
		return err
	}
	selfInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("experiment build metadata unavailable")
	}
	raw, err := json.MarshalIndent(map[string]any{"schema": "gooo/search-policy-study/v1", "compiler": stamp(compilerInfo),
		"experiment": stamp(selfInfo), "runs": rows,
		"scope": "Six finite native runs: legacy policy, source-declared search transitions, mixed fixed/model ordering, round limit and absent alternatives. Two adaptive feedback inputs and two post-selection evaluation inputs per run. No training; model training exposure unknown."}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "summary.json"), append(raw, '\n'), 0644)
}

func options(compiler, root, mode string) (wb.RefineOptions, error) {
	dir := "examples/search-policy/"
	o := wb.RefineOptions{Options: wb.Options{Compiler: compiler, Out: filepath.Join(root, mode)}, Activity: "Add",
		Source: dir + "source.gooo", Cases: dir + "feedback-cases.json", EvaluationCases: dir + "evaluation-cases.json",
		Policy: dir + "policy.gooo", SearchPolicy: true, MaxAttempts: 8, MaxRounds: 6}
	if mode == "legacy" {
		o.SearchPolicy, o.Policy = false, "examples/source-refinement/policy.gooo"
	}
	if strings.HasPrefix(mode, "mixed") {
		o.Source = dir + "mixed.gooo"
		o.Cases, o.EvaluationCases = "examples/search-refinement/mixed-feedback-cases.json", "examples/search-refinement/mixed-evaluation-cases.json"
		if mode == "mixed-model" {
			o.Model = "builtin"
		}
	}
	if mode == "round-limit" {
		o.MaxRounds = 2
	}
	if mode == "no-alternatives" {
		raw, err := os.ReadFile(o.Source)
		if err != nil {
			return o, err
		}
		var lines []string
		for _, line := range strings.Split(string(raw), "\n") {
			if !strings.Contains(line, "search_alternative") {
				lines = append(lines, line)
			}
		}
		o.Source = filepath.Join(root, "no-alternatives.gooo")
		if err := os.WriteFile(o.Source, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return o, err
		}
	}
	return o, nil
}

func stamp(info *debug.BuildInfo) map[string]string {
	result := map[string]string{"go_version": info.GoVersion}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" {
			result[setting.Key] = setting.Value
		}
	}
	return result
}
