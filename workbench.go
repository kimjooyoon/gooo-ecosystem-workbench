package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
)

type Options struct{ Compiler, Model, Out string }
type Snapshot struct {
	Passed   int64  `json:"passed"`
	Total    int64  `json:"total"`
	Rejected int64  `json:"rejected"`
	Detail   string `json:"detail"`
	InputSHA string `json:"input_sha256,omitempty"`
}
type Project struct {
	Filename string `json:"filename"`
	Source   string `json:"source"`
	Next     string `json:"next"`
}
type Summary struct {
	Recipe          string `json:"recipe"`
	Mode            string `json:"mode"`
	CompilerSource  string `json:"compiler_source"`
	NamedPassed     int    `json:"named_passed"`
	NamedTotal      int    `json:"named_total"`
	FieldsPassed    int    `json:"record_fields_passed"`
	FieldsTotal     int    `json:"record_fields_total"`
	SelectionPassed int    `json:"selection_fields_passed"`
	SelectionTotal  int    `json:"selection_fields_total"`
	ModelCalls      int    `json:"generation_model_calls"`
	Rejected        int    `json:"type_rejected_candidates"`
	ReplayVerified  bool   `json:"saved_replay_verified"`
	ResultSHA       string `json:"result_sha256"`
}
type result struct {
	Generated   bool `json:"generated_now"`
	Composition struct {
		Steps []struct {
			Generation struct {
				Report struct {
					Assembly *struct {
						Calls    int `json:"model_calls"`
						Passed   int `json:"fields_passed"`
						Total    int `json:"fields_total"`
						Attempts []struct {
							Status string `json:"status"`
							Reason string `json:"reason"`
						} `json:"attempts"`
					} `json:"record_assembly"`
				} `json:"report"`
			} `json:"generation"`
		} `json:"steps"`
	} `json:"composition"`
	Runtime struct {
		Source string `json:"producer_source_sha"`
		Calls  int    `json:"model_calls"`
		Passed int    `json:"finite_passed"`
		Total  int    `json:"finite_total"`
		Traces []struct {
			Deliveries []struct {
				ID       string          `json:"activity_id"`
				Actual   json.RawMessage `json:"actual"`
				Expected json.RawMessage `json:"expected"`
			} `json:"deliveries"`
		} `json:"traces"`
	} `json:"runtime"`
}

func write(path string, data []byte) error { return os.WriteFile(path, data, 0644) }
func save(path string, value any) error {
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	return write(path, append(b, '\n'))
}
func command(ctx context.Context, compiler string, args ...string) ([]byte, error) {
	if compiler == "" {
		compiler = "gooo"
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	c := exec.CommandContext(ctx, compiler, args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	b, e := c.Output()
	if e != nil {
		return b, fmt.Errorf("Gooo %s: %w: %s", args[0], e, strings.TrimSpace(stderr.String()))
	}
	return b, nil
}
func newOutput(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("a new --out directory is required")
	}
	p, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	if _, e = os.Stat(p); !os.IsNotExist(e) {
		return "", fmt.Errorf("output already exists or cannot be inspected: %s", p)
	}
	if e = os.MkdirAll(p, 0755); e != nil {
		return "", e
	}
	return p, nil
}
func prepareModel(model, out string) (string, error) {
	if model == "" {
		return "", nil
	}
	if model != "builtin" {
		return filepath.Abs(model)
	}
	dir := filepath.Join(out, "model")
	if e := os.Mkdir(dir, 0755); e != nil {
		return "", e
	}
	for _, name := range []string{"model.json", "weights.bin"} {
		b, e := assets.ReadFile("models/shared-qat/" + name)
		if e != nil {
			return "", e
		}
		if e = write(filepath.Join(dir, name), b); e != nil {
			return "", e
		}
	}
	return filepath.Join(dir, "model.json"), nil
}
func runRecipe(ctx context.Context, o Options, root, recipe, label, model string, cases []byte) ([]byte, result, error) {
	var r result
	source, e := assets.ReadFile("recipes/" + recipe + ".gooo")
	if e != nil {
		return nil, r, e
	}
	if cases == nil {
		cases, e = assets.ReadFile("recipes/" + recipe + "-cases.json")
		if e != nil {
			return nil, r, e
		}
	}
	dir := filepath.Join(root, label)
	if e = os.Mkdir(dir, 0755); e != nil {
		return nil, r, e
	}
	sourcePath, casesPath := filepath.Join(dir, "source.gooo"), filepath.Join(dir, "cases.json")
	if e = write(sourcePath, source); e != nil {
		return nil, r, e
	}
	if e = write(casesPath, cases); e != nil {
		return nil, r, e
	}
	args := []string{"body-compose", "--source", sourcePath, "--cases", casesPath, "--out", filepath.Join(dir, "composition")}
	if model != "" {
		args = append(args, "--model", model)
	}
	b, e := command(ctx, o.Compiler, args...)
	if len(b) > 0 {
		if err := write(filepath.Join(dir, "result.json"), b); err != nil {
			return nil, r, err
		}
	}
	if e != nil {
		return b, r, e
	}
	e = json.Unmarshal(b, &r)
	return b, r, e
}
func decodeValue(raw []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	e := d.Decode(&v)
	return v, e
}

// ReadSnapshot accepts a compiler body-compose result or explicit counters.
// It recounts actual values and transports the remaining gaps to the Gooo recipe.
func ReadSnapshot(raw []byte) (Snapshot, error) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return Snapshot{}, err
	}
	if keys["runtime"] == nil {
		if keys["passed"] == nil || keys["total"] == nil {
			return Snapshot{}, fmt.Errorf("provide a body-compose result or passed/total counters")
		}
		var s Snapshot
		err := json.Unmarshal(raw, &s)
		return s, err
	}
	counts, err := summarize(raw, "observation", "captured")
	if err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{Passed: int64(counts.NamedPassed), Total: int64(counts.NamedTotal), Rejected: int64(counts.Rejected), InputSHA: fmt.Sprintf("%x", sha256.Sum256(raw))}
	if counts.FieldsTotal > 0 {
		s.Passed, s.Total = int64(counts.FieldsPassed), int64(counts.FieldsTotal)
	}
	var r result
	if err = json.Unmarshal(raw, &r); err != nil {
		return s, err
	}
	var gaps []map[string]any
	var rejected []string
	for _, trace := range r.Runtime.Traces {
		for _, value := range trace.Deliveries {
			if len(value.Expected) == 0 {
				continue
			}
			actual, _ := decodeValue(value.Actual)
			expected, _ := decodeValue(value.Expected)
			if fields, ok := expected.(map[string]any); ok && counts.FieldsTotal > 0 {
				a, _ := actual.(map[string]any)
				names := make([]string, 0, len(fields))
				for field := range fields {
					names = append(names, field)
				}
				slices.Sort(names)
				for _, field := range names {
					want := fields[field]
					got, exists := a[field]
					if !exists || !reflect.DeepEqual(got, want) {
						gaps = append(gaps, map[string]any{"activity": value.ID, "field": field, "actual": got, "expected": want})
					}
				}
			} else if counts.FieldsTotal == 0 && !reflect.DeepEqual(actual, expected) {
				gaps = append(gaps, map[string]any{"activity": value.ID, "actual": actual, "expected": expected})
			}
		}
	}
	for _, step := range r.Composition.Steps {
		if a := step.Generation.Report.Assembly; a != nil {
			for _, trial := range a.Attempts {
				if trial.Status == "TYPECHECK_FAILED" {
					rejected = append(rejected, trial.Reason)
				}
			}
		}
	}
	detail, err := json.Marshal(map[string]any{"gaps": gaps, "type_rejections": rejected})
	if len(detail) > 1024 {
		detail, err = json.Marshal(map[string]any{"detail_limited": true, "field_gaps": len(gaps), "type_rejections": len(rejected), "input_sha256": s.InputSHA})
	}
	s.Detail = string(detail)
	return s, err
}
func summarize(raw []byte, recipe, mode string) (Summary, error) {
	s := Summary{Recipe: recipe, Mode: mode, ResultSHA: fmt.Sprintf("%x", sha256.Sum256(raw))}
	var r result
	if e := json.Unmarshal(raw, &r); e != nil {
		return s, e
	}
	s.CompilerSource = r.Runtime.Source
	for _, step := range r.Composition.Steps {
		if a := step.Generation.Report.Assembly; a != nil {
			s.ModelCalls += a.Calls
			s.SelectionPassed += a.Passed
			s.SelectionTotal += a.Total
			for _, attempt := range a.Attempts {
				if attempt.Status == "TYPECHECK_FAILED" {
					s.Rejected++
				}
			}
		}
	}
	for _, trace := range r.Runtime.Traces {
		for _, v := range trace.Deliveries {
			if len(v.Expected) == 0 {
				continue
			}
			actual, e := decodeValue(v.Actual)
			if e != nil {
				return s, e
			}
			expected, e := decodeValue(v.Expected)
			if e != nil {
				return s, e
			}
			s.NamedTotal++
			if reflect.DeepEqual(actual, expected) {
				s.NamedPassed++
			}
			if fields, ok := expected.(map[string]any); ok {
				a, _ := actual.(map[string]any)
				for name, value := range fields {
					s.FieldsTotal++
					if got, exists := a[name]; exists && reflect.DeepEqual(got, value) {
						s.FieldsPassed++
					}
				}
			}
		}
	}
	if s.NamedPassed != r.Runtime.Passed || s.NamedTotal != r.Runtime.Total || r.Runtime.Calls != 0 {
		return s, fmt.Errorf("native values disagree with reported counts or runtime predicted")
	}
	return s, nil
}

// Verify checks Gooo-generated functions against explicit finite expectations.
// Optional model ordering has exactly one prediction per assembling recipe.
func Verify(ctx context.Context, o Options) ([]Summary, error) {
	root, e := newOutput(o.Out)
	if e != nil {
		return nil, e
	}
	model, e := prepareModel(o.Model, root)
	if e != nil {
		return nil, e
	}
	var summaries []Summary
	for _, recipe := range []string{"stdlib", "diagnostics", "starter"} {
		modes := []string{"deterministic"}
		if model != "" && recipe != "stdlib" {
			modes = append(modes, "model")
		}
		for _, mode := range modes {
			m := ""
			if mode == "model" {
				m = model
			}
			label := recipe + "-" + mode
			raw, _, e := runRecipe(ctx, o, root, recipe, label, m, nil)
			if e != nil {
				return summaries, e
			}
			s, e := summarize(raw, recipe, mode)
			if e != nil {
				return summaries, e
			}
			expectCalls := 0
			if mode == "model" {
				expectCalls = 1
			}
			if s.NamedTotal == 0 || s.NamedPassed != s.NamedTotal || s.FieldsPassed != s.FieldsTotal || s.ModelCalls != expectCalls || s.SelectionPassed != s.SelectionTotal {
				return summaries, fmt.Errorf("recipe incomplete: %+v", s)
			}
			dir := filepath.Join(root, label, "composition")
			replay, e := command(ctx, o.Compiler, "body-compose", "--source", filepath.Join(dir, "original.gooo"), "--cases", filepath.Join(dir, "cases.json"), "--composition", filepath.Join(dir, "composition.json"))
			if e != nil {
				return summaries, e
			}
			if e = write(filepath.Join(root, label, "replay.json"), replay); e != nil {
				return summaries, e
			}
			rs, e := summarize(replay, recipe, mode)
			if e != nil {
				return summaries, e
			}
			var saved result
			if e = json.Unmarshal(replay, &saved); e != nil {
				return summaries, e
			}
			if saved.Generated || rs.NamedPassed != s.NamedPassed || rs.NamedTotal != s.NamedTotal || rs.FieldsPassed != s.FieldsPassed {
				return summaries, fmt.Errorf("saved replay differs")
			}
			s.ReplayVerified = true
			summaries = append(summaries, s)
		}
	}
	return summaries, save(filepath.Join(root, "summary.json"), map[string]any{"schema": "gooo/ecosystem-workbench-verification/v1", "scope": "13 authored standard functions and two ecosystem recipes; finite examples and separately compiled native values; no training or performance study", "recipes": summaries})
}

func runtimeCase(inputs map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{map[string]any{"inputs": inputs, "expected": map[string]any{"ObservationEcho": inputs["ObservationEcho"]}}}})
	return b
}

func starterCase(profile string) ([]byte, error) {
	raw, err := assets.ReadFile("recipes/starter-cases.json")
	if err != nil {
		return nil, err
	}
	var bank struct {
		Cases []json.RawMessage `json:"cases"`
	}
	if err = json.Unmarshal(raw, &bank); err != nil {
		return nil, err
	}
	for _, row := range bank.Cases {
		var entry struct {
			Inputs map[string]string `json:"inputs"`
		}
		if err = json.Unmarshal(row, &entry); err != nil {
			return nil, err
		}
		if entry.Inputs["Plan"] == profile {
			return json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []json.RawMessage{row}})
		}
	}
	return nil, fmt.Errorf("no published starter example for profile %q", profile)
}
func actualFor(r result, id string, dest any) error {
	for _, trace := range r.Runtime.Traces {
		for _, v := range trace.Deliveries {
			if v.ID == id {
				return json.Unmarshal(v.Actual, dest)
			}
		}
	}
	return fmt.Errorf("Gooo produced no %s value", id)
}

// Scaffold lets a Gooo program produce another Gooo source file.
func Scaffold(ctx context.Context, o Options, profile string) (Project, error) {
	var p Project
	root, e := newOutput(o.Out)
	if e != nil {
		return p, e
	}
	model, e := prepareModel(o.Model, root)
	if e != nil {
		return p, e
	}
	cases, e := starterCase(profile)
	if e != nil {
		return p, e
	}
	_, r, e := runRecipe(ctx, o, root, "starter", "plan", model, cases)
	if e != nil {
		return p, e
	}
	if e = actualFor(r, "starterplanner://activity/plan", &p); e != nil {
		return p, e
	}
	if e = save(filepath.Join(root, "project-plan.json"), p); e != nil {
		return p, e
	}
	if p.Filename != "main.gooo" || p.Source == "" {
		return p, fmt.Errorf("Gooo project plan: %s", p.Next)
	}
	path := filepath.Join(root, p.Filename)
	if e = write(path, []byte(p.Source)); e != nil {
		return p, e
	}
	if _, e = command(ctx, o.Compiler, "check", path); e != nil {
		return p, e
	}
	body, e := command(ctx, o.Compiler, "body-codegen", "--json", "--activity", "Identity", path)
	if e != nil {
		return p, e
	}
	return p, write(filepath.Join(root, "identity-generation.json"), body)
}

// Diagnose transports counters; branch rules and the proposed action live in Gooo.
func Diagnose(ctx context.Context, o Options, s Snapshot) (json.RawMessage, error) {
	root, e := newOutput(o.Out)
	if e != nil {
		return nil, e
	}
	model, e := prepareModel(o.Model, root)
	if e != nil {
		return nil, e
	}
	if e = save(filepath.Join(root, "observation.json"), s); e != nil {
		return nil, e
	}
	_, r, e := runRecipe(ctx, o, root, "diagnostics", "diagnose", model, runtimeCase(map[string]any{"Diagnose.input0": s.Passed, "Diagnose.input1": s.Total, "Diagnose.input2": s.Rejected, "Diagnose.input3": s.Detail, "ObservationEcho": s.Detail}))
	if e != nil {
		return nil, e
	}
	var actual json.RawMessage
	if e = actualFor(r, "diagnostics://activity/diagnose", &actual); e != nil {
		return nil, e
	}
	return actual, write(filepath.Join(root, "diagnostic.json"), append(actual, '\n'))
}
