// dependent-splice evaluates a frozen new program without updating the model.
package main

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	wb "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

const frozenModel = "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json"

type runReport struct {
	Mode         string          `json:"mode"`
	Budget       int             `json:"budget"`
	Assembly     json.RawMessage `json:"assembly"`
	Native       counts          `json:"native"`
	Replay       counts          `json:"replay"`
	GeneratedSHA string          `json:"generated_sha256"`
	DurationMS   float64         `json:"duration_ms"`
}

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
	root, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	if err = os.Mkdir(root, 0755); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	source, err := os.ReadFile("recipes/splice.gooo")
	if err != nil {
		return err
	}
	rows := roster()
	if err = assertDisjoint(source, rows); err != nil {
		return err
	}
	if err = save(filepath.Join(root, "roster.json"), rows); err != nil {
		return err
	}
	paths, groups, err := batches(root, rows)
	if err != nil {
		return err
	}
	var reports []runReport
	for _, mode := range []string{"deterministic", "model"} {
		for _, budget := range []int{1, 2, 4, 8} {
			r, err := evaluate(ctx, *compiler, root, source, paths, groups, mode, budget)
			if err != nil {
				return err
			}
			reports = append(reports, r)
			fmt.Printf("%s budget=%d native=%d/%d fields=%d/%d replay=%d/%d\n", mode, budget, r.Native.Passed, r.Native.Cases, r.Native.FieldsPassed, r.Native.Fields, r.Replay.Passed, r.Replay.Cases)
		}
	}
	for _, mode := range []string{"deterministic", "model"} {
		if err = dogfood(ctx, *compiler, root, mode); err != nil {
			return err
		}
	}
	compilerPath, err := exec.LookPath(*compiler)
	if err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(compilerPath)
	if err != nil {
		return err
	}
	self, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("missing experiment build metadata")
	}
	digests := map[string]string{}
	for _, path := range []string{"recipes/splice.gooo", "experiments/dependent-splice/PLAN.md", frozenModel, filepath.Join(filepath.Dir(frozenModel), "weights.bin")} {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digests[path] = fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	}
	return save(filepath.Join(root, "summary.json"), map[string]any{"schema": "gooo/dependent-splice-study/v1", "compiler": info, "experiment": self,
		"digests": digests, "program_families": 1, "unique_native_inputs": len(rows), "selection_overlap": 0, "runs": reports,
		"scope": "One new program with three dependent binary field choices and fixed bilingual wording. Unchanged graph QAT trained on filenames, division and retry. Four attempt budgets per mode; each model construction predicts once. Native and saved executions predict zero times. Counts are finite supplied examples, not a probability of arbitrary intent completion. Dogfood runs are separate full-budget constructions."})
}

func evaluate(ctx context.Context, compiler, root string, source []byte, paths []string, groups [][]wb.SpliceRequest, mode string, budget int) (runReport, error) {
	r := runReport{Mode: mode, Budget: budget}
	started := time.Now()
	dir := filepath.Join(root, fmt.Sprintf("%s-%d", mode, budget))
	if err := os.Mkdir(dir, 0755); err != nil {
		return r, err
	}
	if strings.Count(string(source), `attempts "8"`) != 1 {
		return r, fmt.Errorf("unexpected source budget")
	}
	sourcePath := filepath.Join(dir, "source.gooo")
	if err := os.WriteFile(sourcePath, []byte(strings.Replace(string(source), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1)), 0644); err != nil {
		return r, err
	}
	composition := filepath.Join(dir, "composition")
	var prior [][]wb.SpliceEdit
	for _, replay := range []bool{false, true} {
		for i, path := range paths {
			args := []string{"body-compose", "--source", sourcePath, "--cases", path}
			fresh := !replay && i == 0
			if fresh {
				args = append(args, "--out", composition)
				if mode == "model" {
					args = append(args, "--model", frozenModel)
				}
			} else {
				args = append(args, "--composition", filepath.Join(composition, "composition.json"))
			}
			raw, err := execute(ctx, compiler, filepath.Join(dir, fmt.Sprintf("replay-%t-batch-%02d.json", replay, i)), args...)
			if err != nil {
				return r, err
			}
			var observed observation
			if err = json.Unmarshal(raw, &observed); err != nil {
				return r, err
			}
			if observed.Generated != fresh || len(observed.Composition.Steps) != 1 {
				return r, fmt.Errorf("unexpected construction count")
			}
			if fresh {
				r.GeneratedSHA = observed.Composition.SHA
				r.Assembly = observed.Composition.Steps[0].Generation.Report.Assembly
				var a struct {
					Calls  int `json:"model_calls"`
					Passed int `json:"passed"`
					Total  int `json:"total"`
				}
				if err = json.Unmarshal(r.Assembly, &a); err != nil {
					return r, err
				}
				wantCalls := 0
				if mode == "model" {
					wantCalls = 1
				}
				if a.Calls != wantCalls || a.Total != 8 || (budget == 8 && a.Passed != 8) {
					return r, fmt.Errorf("construction inference or finite counts differ")
				}
			}
			if r.GeneratedSHA == "" || observed.Composition.SHA != r.GeneratedSHA {
				return r, fmt.Errorf("saved source changed")
			}
			c, values, err := recount(observed, groups[i])
			if err != nil {
				return r, err
			}
			if replay {
				if !reflect.DeepEqual(values, prior[i]) {
					return r, fmt.Errorf("saved replay output differs")
				}
				addCounts(&r.Replay, c)
			} else {
				prior = append(prior, values)
				addCounts(&r.Native, c)
			}
		}
	}
	if budget == 8 && r.Native.Passed != r.Native.Cases {
		return r, fmt.Errorf("full construction misses independent oracle")
	}
	r.DurationMS = float64(time.Since(started).Microseconds()) / 1000
	return r, save(filepath.Join(dir, "summary.json"), r)
}

func dogfood(ctx context.Context, compiler, root, mode string) error {
	raw, err := os.ReadFile("examples/source-splice/request.json")
	if err != nil {
		return err
	}
	request, err := wb.ReadSpliceRequest(raw)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "dogfood-"+mode)
	o := wb.Options{Compiler: compiler, Out: dir}
	if mode == "model" {
		o.Model = frozenModel
	}
	report, err := wb.SpliceSource(ctx, o, request)
	if err != nil {
		return err
	}
	if report.Edit != oracle(request) {
		return fmt.Errorf("dogfood edit differs from oracle")
	}
	var cases []any
	for _, n := range []int64{-5, 0, 9} {
		cases = append(cases, map[string]any{"inputs": map[string]any{"Calculate": n}, "expected": map[string]any{"Calculate": n * 2}})
	}
	path := filepath.Join(dir, "edited-cases.json")
	if err = save(path, map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases}); err != nil {
		return err
	}
	raw, err = execute(ctx, compiler, filepath.Join(dir, "edited-execution.json"), "body-compose", "--source", filepath.Join(dir, "edited.txt"), "--cases", path, "--out", filepath.Join(dir, "edited-program"))
	if err != nil {
		return err
	}
	var result struct {
		Runtime struct {
			Passed int `json:"finite_passed"`
			Total  int `json:"finite_total"`
			Calls  int `json:"model_calls"`
		} `json:"runtime"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return err
	}
	if result.Runtime.Passed != 3 || result.Runtime.Total != 3 || result.Runtime.Calls != 0 {
		return fmt.Errorf("edited Gooo program failed its independent inputs")
	}
	return nil
}

func addCounts(dst *counts, src counts) {
	dst.Cases += src.Cases
	dst.Passed += src.Passed
	dst.Fields += src.Fields
	dst.FieldsPassed += src.FieldsPassed
}

func save(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
