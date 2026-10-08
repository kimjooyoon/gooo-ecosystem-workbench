// search-refinement records bounded source search and mixed model construction.
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
		return fmt.Errorf("provide a new out directory")
	}
	if err := os.Mkdir(*out, 0755); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rows := map[string]wb.RefinementReport{}
	for _, mode := range []string{"search", "mixed", "mixed-model", "missing-expression", "candidate-cap"} {
		o, err := options(*compiler, *out, mode)
		if err != nil {
			return err
		}
		r, err := wb.Refine(ctx, o)
		if err != nil {
			return fmt.Errorf("%s: %w", mode, err)
		}
		rows[mode] = r
		fmt.Printf("%s: %s %d/%d -> %d/%d; %s\n", mode, r.Status, r.Initial.Passed, r.Initial.Total, r.Final.Passed, r.Final.Total, r.FinalPlan.Action)
	}
	compilerPath, err := exec.LookPath(*compiler)
	if err != nil {
		return err
	}
	compilerInfo, err := buildinfo.ReadFile(compilerPath)
	if err != nil {
		return err
	}
	selfInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("experiment build metadata unavailable")
	}
	raw, err := json.MarshalIndent(map[string]any{
		"schema": "gooo/search-refinement-study/v1", "compiler": stamp(compilerInfo), "experiment": stamp(selfInfo), "runs": rows,
		"scope": "Five finite source-construction runs. Two adaptive feedback inputs and two post-selection evaluation inputs per run. Integer search is deterministic; the own compact model orders three record choices in the mixed-model arm. No training or general accuracy claim; model training exposure is unknown.",
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "summary.json"), append(raw, '\n'), 0644)
}

func options(compiler, root, mode string) (wb.RefineOptions, error) {
	dir := "examples/search-refinement/"
	o := wb.RefineOptions{Options: wb.Options{Compiler: compiler, Out: filepath.Join(root, mode)}, Activity: "Add",
		Source: dir + "source.gooo", Cases: dir + "feedback-cases.json", EvaluationCases: dir + "evaluation-cases.json",
		Policy: "examples/source-refinement/policy.gooo", MaxAttempts: 8, MaxRounds: 4}
	if strings.HasPrefix(mode, "mixed") {
		o.Source, o.Cases, o.EvaluationCases = dir+"mixed.gooo", dir+"mixed-feedback-cases.json", dir+"mixed-evaluation-cases.json"
		if mode == "mixed-model" {
			o.Model = "builtin"
		}
	}
	if mode == "missing-expression" || mode == "candidate-cap" {
		raw, err := os.ReadFile(o.Source)
		if err != nil {
			return o, err
		}
		source := strings.Replace(string(raw), "return __GOOO", "return input + __GOOO", 1)
		if mode == "candidate-cap" {
			source = strings.Replace(source, `max_candidates "16"`, `max_candidates "2"`, 1)
		}
		o.Source = filepath.Join(root, mode+".gooo")
		if err := os.WriteFile(o.Source, []byte(source), 0644); err != nil {
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
