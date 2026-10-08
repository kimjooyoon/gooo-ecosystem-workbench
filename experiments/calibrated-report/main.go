// calibrated-report joins numeric source refinement with compact-model records.
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
	"time"

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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	rows := map[string]wb.RefinementReport{}
	for _, mode := range []string{"fixed", "model", "round-limit", "no-alternative"} {
		dir := "examples/calibrated-report/"
		o := wb.RefineOptions{Options: wb.Options{Compiler: *compiler, Out: filepath.Join(*out, mode)},
			Source: dir + "source.gooo", Activity: "Energy", Cases: dir + "feedback-cases.json",
			EvaluationCases: dir + "evaluation-cases.json", Policy: "examples/search-policy/policy.gooo",
			SearchPolicy: true, MaxAttempts: 8, MaxRounds: 4}
		if mode == "model" || mode == "round-limit" {
			o.Model = "builtin"
		}
		if mode == "round-limit" {
			o.MaxRounds = 1
		}
		if mode == "no-alternative" {
			raw, err := os.ReadFile(o.Source)
			if err != nil {
				return err
			}
			source := strings.Replace(string(raw), "    search_alternative \"shared_fit\" grammar \"integer-hole-quadratic/v2\" max_candidates \"16\"\n", "", 1)
			o.Source = filepath.Join(*out, "no-alternative.gooo")
			if err := os.WriteFile(o.Source, []byte(source), 0644); err != nil {
				return err
			}
		}
		r, err := wb.Refine(ctx, o)
		if err != nil {
			return fmt.Errorf("%s: %w", mode, err)
		}
		rows[mode] = r
		fmt.Printf("%s: %s %d/%d -> %d/%d, rounds=%d, model calls=%d\n", mode, r.Status,
			r.Initial.Passed, r.Initial.Total, r.Final.Passed, r.Final.Total, r.Rounds, r.InitialModelCalls+r.RefinementModelCalls)
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
	raw, err := json.MarshalIndent(map[string]any{"schema": "gooo/calibrated-report-study/v1",
		"compiler": stamp(compilerInfo), "experiment": stamp(selfInfo), "runs": rows,
		"scope": "One synthetic calibration graph in four modes, ten adaptive root inputs and three separate evaluation inputs, two expected activity outputs per input. Numeric grammar is deterministic; the existing compact model ranks three record fields. No training; model-training exposure unknown."}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "summary.json"), append(raw, '\n'), 0644)
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
