// source-budget observes source-owned limits through native Gooo construction
// and the Gooo next-step program, with an optional separately measured model arm.
package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	wb "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

type observation struct {
	Mode         string                     `json:"mode"`
	Budget       int                        `json:"declared_budget"`
	Construction wb.ConstructionObservation `json:"construction"`
	NativePassed int64                      `json:"native_passed"`
	NativeTotal  int64                      `json:"native_total"`
	NativeUnit   string                     `json:"native_unit"`
	Action       string                     `json:"action"`
	ResultSHA    string                     `json:"result_sha256"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	compiler := flag.String("compiler", "gooo", "Gooo compiler executable")
	model := flag.String("model", "", "optional compatible record assembly model")
	out := flag.String("out", "", "new evidence directory")
	flag.Parse()
	if *out == "" {
		return fmt.Errorf("provide a new out directory")
	}
	if err := os.Mkdir(*out, 0755); err != nil {
		return err
	}
	template, err := os.ReadFile("examples/source-budget/assembly.gooo")
	if err != nil {
		return err
	}
	if strings.Count(string(template), `attempts "8"`) != 1 {
		return fmt.Errorf("source must declare exactly one attempts value")
	}
	cases, err := os.ReadFile("examples/source-budget/cases.json")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*out, "cases.json"), cases, 0644); err != nil {
		return err
	}
	paths := []string{""}
	if *model != "" {
		paths = append(paths, *model)
	}
	var rows []observation
	for _, modelPath := range paths {
		mode := "deterministic"
		if modelPath != "" {
			mode = "model"
		}
		for _, budget := range []int{1, 3, 8, 16} {
			dir := filepath.Join(*out, fmt.Sprintf("%s-%d", mode, budget))
			if err = os.Mkdir(dir, 0755); err != nil {
				return err
			}
			source := filepath.Join(dir, "source.gooo")
			if err = os.WriteFile(source, []byte(strings.Replace(string(template), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1)), 0644); err != nil {
				return err
			}
			args := []string{"body-compose", "--source", source, "--cases", filepath.Join(*out, "cases.json"), "--out", filepath.Join(dir, "composition")}
			if modelPath != "" {
				args = append(args, "--model", modelPath)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			raw, runErr := exec.CommandContext(ctx, *compiler, args...).Output()
			cancel()
			if runErr != nil {
				return fmt.Errorf("%s budget %d: %w", mode, budget, runErr)
			}
			if err = os.WriteFile(filepath.Join(dir, "result.json"), raw, 0644); err != nil {
				return err
			}
			s, err := wb.ReadSnapshot(raw)
			if err != nil {
				return err
			}
			if len(s.Construction) != 1 || !s.Construction[0].BudgetKnown || s.Construction[0].Budget != int64(budget) || s.Construction[0].Ranked != 8 {
				return fmt.Errorf("source budget observation missing in %s budget %d", mode, budget)
			}
			ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
			diagnosis, runErr := wb.Diagnose(ctx, wb.Options{Compiler: *compiler, Out: filepath.Join(dir, "diagnosis")}, s)
			cancel()
			if runErr != nil {
				return runErr
			}
			var d struct {
				Steps []wb.ConstructionNextStep `json:"construction_next_steps"`
			}
			if err = json.Unmarshal(diagnosis, &d); err != nil {
				return err
			}
			if len(d.Steps) != 1 {
				return fmt.Errorf("missing Gooo next step")
			}
			rows = append(rows, observation{mode, budget, s.Construction[0], s.Passed, s.Total, s.Unit, d.Steps[0].Action, s.InputSHA})
		}
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
	summary := map[string]any{"schema": "gooo/source-budget-study/v1", "compiler": stamp(compilerInfo), "experiment": stamp(selfInfo), "observations": rows,
		"scope": "Four source budgets over one eight-candidate program. Seven runtime inputs include five construction cases and two additional inputs. Diagnostic actions execute in Gooo with zero diagnostic model calls. Finite observations do not establish general accuracy or speedup."}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*out, "summary.json"), append(raw, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

func stamp(info *debug.BuildInfo) map[string]string {
	m := map[string]string{"go_version": info.GoVersion}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" || s.Key == "vcs.modified" {
			m[s.Key] = s.Value
		}
	}
	return m
}
