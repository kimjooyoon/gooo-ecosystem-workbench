package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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
type TypeReference struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}
type PublicInterface struct {
	Schema        string `json:"schema"`
	Decision      string `json:"decision"`
	Resolution    string `json:"resolution"`
	Reason        string `json:"reason"`
	Kind          string `json:"kind"`
	SubjectDigest string `json:"subject_digest"`
	Package       struct {
		Path      string `json:"path"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"package"`
	Operation struct {
		Activity string          `json:"activity"`
		Inputs   []TypeReference `json:"inputs"`
		Output   TypeReference   `json:"output"`
	} `json:"operation"`
	Definitions struct {
		Language string   `json:"language"`
		Files    []string `json:"files"`
	} `json:"definitions"`
	Extensions struct {
		RegisteredEmitters int      `json:"registered_emitters"`
		Kinds              []string `json:"kinds"`
	} `json:"extensions"`
	Effects struct {
		RepositoryWrites  int  `json:"repository_writes"`
		MutationAuthority bool `json:"mutation_authority"`
	} `json:"effects"`
	Digest string `json:"digest"`
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
type CompletenessAssessment struct {
	Domain                     string `json:"domain"`
	AllowedInvestment          string `json:"allowed_investment"`
	ExcludedScope              string `json:"excluded_scope"`
	DeclarationStatus          string `json:"declaration_status"`
	DeclarationEvidence        string `json:"declaration_evidence"`
	GenerationStatus           string `json:"generation_status"`
	GenerationEvidence         string `json:"generation_evidence"`
	ReverseObservationStatus   string `json:"reverse_observation_status"`
	ReverseObservationEvidence string `json:"reverse_observation_evidence"`
	UseCaseStatus              string `json:"use_case_status"`
	UseCaseEvidence            string `json:"use_case_evidence"`
	BoundaryStatus             string `json:"boundary_status"`
	BoundaryEvidence           string `json:"boundary_evidence"`
	ProvenanceStatus           string `json:"provenance_status"`
	ProvenanceEvidence         string `json:"provenance_evidence"`
}
type CompletenessReceipt struct {
	Domain                     string `json:"domain"`
	AllowedInvestment          string `json:"allowed_investment"`
	ExcludedScope              string `json:"excluded_scope"`
	DeclarationStatus          string `json:"declaration_status"`
	DeclarationEvidence        string `json:"declaration_evidence"`
	GenerationStatus           string `json:"generation_status"`
	GenerationEvidence         string `json:"generation_evidence"`
	ReverseObservationStatus   string `json:"reverse_observation_status"`
	ReverseObservationEvidence string `json:"reverse_observation_evidence"`
	UseCaseStatus              string `json:"use_case_status"`
	UseCaseEvidence            string `json:"use_case_evidence"`
	BoundaryStatus             string `json:"boundary_status"`
	BoundaryEvidence           string `json:"boundary_evidence"`
	ProvenanceStatus           string `json:"provenance_status"`
	ProvenanceEvidence         string `json:"provenance_evidence"`
	FirstUnresolvedStage       string `json:"first_unresolved_stage"`
}
type result struct {
	Generated   bool `json:"generated_now"`
	Composition struct {
		OriginalSourceSHA string `json:"original_source_sha256"`
		GeneratedSHA      string `json:"generated_sha256"`
		Steps             []struct {
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
	sourceName, casesName := "recipes/"+recipe+".gooo", "recipes/"+recipe+"-cases.json"
	if recipe == "invoice-approval" {
		sourceName, casesName = "examples/invoice-approval/approval.gooo", "examples/invoice-approval/cases.json"
	}
	source, e := assets.ReadFile(sourceName)
	if e != nil {
		return nil, r, e
	}
	if cases == nil {
		cases, e = assets.ReadFile(casesName)
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
	for _, recipe := range []string{"stdlib", "diagnostics", "starter", "invoice-approval"} {
		modes := []string{"deterministic"}
		if model != "" && (recipe == "diagnostics" || recipe == "starter") {
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
	repositoryHead := ""
	repositoryClean := false
	if status, statusErr := exec.CommandContext(ctx, "git", "status", "--porcelain").Output(); statusErr == nil && len(bytes.TrimSpace(status)) == 0 {
		repositoryClean = true
	}
	if repositoryClean {
		if head, headErr := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output(); headErr == nil {
			repositoryHead = strings.TrimSpace(string(head))
		}
	}
	return summaries, save(filepath.Join(root, "summary.json"), map[string]any{
		"schema":          "gooo/ecosystem-workbench-verification/v1",
		"scope":           "13 authored standard functions, two ecosystem recipes, and a finite invoice-approval domain example; separately compiled native values; no training or performance study",
		"repository_head": repositoryHead, "repository_clean": repositoryClean, "recipes": summaries,
	})
}

func digestFiles(root string, names ...string) (string, error) {
	h := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return "", err
		}
		if _, err = io.WriteString(h, name+"\x00"); err != nil {
			return "", err
		}
		if _, err = h.Write(data); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

func firstUnresolved(a CompletenessAssessment) string {
	for _, stage := range []struct{ name, status string }{
		{"declaration", a.DeclarationStatus}, {"generation", a.GenerationStatus},
		{"reverse_observation", a.ReverseObservationStatus}, {"use_case", a.UseCaseStatus},
		{"boundary", a.BoundaryStatus}, {"provenance", a.ProvenanceStatus},
	} {
		if stage.status != "PASS" {
			return stage.name
		}
	}
	return "none"
}

// CompletenessReceipt turns a completed workbench verification into an
// evidence-linked receipt. It deliberately reports finite fixture coverage
// as PROGRESS and unmeasured effects as UNKNOWN instead of inventing a score.
func CompletenessReceiptFor(ctx context.Context, o Options, verificationDir string) (CompletenessReceipt, error) {
	var receipt CompletenessReceipt
	if o.Model != "" {
		return receipt, fmt.Errorf("completeness receipts are deterministic; omit --model")
	}
	if verificationDir == "" {
		return receipt, fmt.Errorf("completeness requires --input pointing to a completed verify output directory")
	}
	inputRoot, err := filepath.Abs(verificationDir)
	if err != nil {
		return receipt, err
	}
	var report struct {
		Schema          string    `json:"schema"`
		Scope           string    `json:"scope"`
		RepositoryHead  string    `json:"repository_head"`
		RepositoryClean bool      `json:"repository_clean"`
		Recipes         []Summary `json:"recipes"`
	}
	summaryBytes, err := os.ReadFile(filepath.Join(inputRoot, "summary.json"))
	if err != nil {
		return receipt, fmt.Errorf("read completed verification summary: %w", err)
	}
	if err = json.Unmarshal(summaryBytes, &report); err != nil {
		return receipt, fmt.Errorf("read completed verification summary: %w", err)
	}
	if report.Schema != "gooo/ecosystem-workbench-verification/v1" || report.Scope == "" || len(report.Recipes) == 0 {
		return receipt, fmt.Errorf("input is not a complete ecosystem verification report")
	}
	seen := make(map[string]bool, len(report.Recipes))
	for _, row := range report.Recipes {
		if (row.Recipe != "stdlib" && row.Recipe != "diagnostics" && row.Recipe != "starter" && row.Recipe != "invoice-approval") ||
			(row.Mode != "deterministic" && row.Mode != "model") ||
			(row.Mode == "model" && row.Recipe != "diagnostics" && row.Recipe != "starter") {
			return receipt, fmt.Errorf("verification includes an unknown recipe/mode: %s/%s", row.Recipe, row.Mode)
		}
		key := row.Recipe + "/" + row.Mode
		if seen[key] {
			return receipt, fmt.Errorf("verification repeats recipe/mode %s", key)
		}
		seen[key] = true
	}
	for _, required := range []string{"stdlib/deterministic", "diagnostics/deterministic", "starter/deterministic", "invoice-approval/deterministic"} {
		if !seen[required] {
			return receipt, fmt.Errorf("verification omits required recipe/mode %s", required)
		}
	}
	if (len(seen) != 4 && len(seen) != 6) || (len(seen) == 6 && (!seen["diagnostics/model"] || !seen["starter/model"])) {
		return receipt, fmt.Errorf("verification contains an incomplete deterministic/model recipe set")
	}
	var sourcePaths, generationPaths, reversePaths, useCasePaths []string
	var compilerSource string
	for _, row := range report.Recipes {
		if row.NamedTotal == 0 || row.NamedPassed != row.NamedTotal || row.FieldsPassed != row.FieldsTotal ||
			row.SelectionPassed != row.SelectionTotal || !row.ReplayVerified || row.ResultSHA == "" || row.CompilerSource == "" {
			return receipt, fmt.Errorf("verification row is incomplete: %s/%s", row.Recipe, row.Mode)
		}
		if compilerSource == "" {
			compilerSource = row.CompilerSource
		} else if compilerSource != row.CompilerSource {
			return receipt, fmt.Errorf("verification mixes compiler source revisions")
		}
		label := row.Recipe + "-" + row.Mode
		for _, rel := range []string{filepath.Join(label, "source.gooo"), filepath.Join(label, "cases.json")} {
			if _, err = os.Stat(filepath.Join(inputRoot, rel)); err != nil {
				return receipt, fmt.Errorf("missing declaration/use-case evidence %s: %w", rel, err)
			}
		}
		sourcePaths = append(sourcePaths, filepath.Join(label, "source.gooo"))
		useCasePaths = append(useCasePaths, filepath.Join(label, "cases.json"), filepath.Join(label, "result.json"))
		originalBytes, readErr := os.ReadFile(filepath.Join(inputRoot, label, "result.json"))
		if readErr != nil || fmt.Sprintf("%x", sha256.Sum256(originalBytes)) != row.ResultSHA {
			return receipt, fmt.Errorf("verification result digest does not match its summary: %s", label)
		}
		var original result
		if err = json.Unmarshal(originalBytes, &original); err != nil {
			return receipt, fmt.Errorf("read verification execution result %s: %w", label, err)
		}
		observedSummary, summaryErr := summarize(originalBytes, row.Recipe, row.Mode)
		observedSummary.ReplayVerified = row.ReplayVerified
		if summaryErr != nil || !row.ReplayVerified || !reflect.DeepEqual(observedSummary, row) || original.Runtime.Source != row.CompilerSource {
			return receipt, fmt.Errorf("verification summary differs from its execution evidence: %s", label)
		}
		sourceBytes, readErr := os.ReadFile(filepath.Join(inputRoot, label, "source.gooo"))
		if readErr != nil || original.Composition.OriginalSourceSHA != "sha256:"+fmt.Sprintf("%x", sha256.Sum256(sourceBytes)) {
			return receipt, fmt.Errorf("verification declaration digest does not match its execution: %s", label)
		}
		for _, rel := range []string{filepath.Join(label, "composition", "composition.json"), filepath.Join(label, "composition", "generated.go")} {
			if _, err = os.Stat(filepath.Join(inputRoot, rel)); err != nil {
				return receipt, fmt.Errorf("missing generation evidence %s: %w", rel, err)
			}
			generationPaths = append(generationPaths, rel)
		}
		replayPath := filepath.Join(label, "replay.json")
		replayBytes, readErr := os.ReadFile(filepath.Join(inputRoot, replayPath))
		if readErr != nil {
			return receipt, fmt.Errorf("missing reverse-observation evidence %s: %w", replayPath, readErr)
		}
		var replay result
		if err = json.Unmarshal(replayBytes, &replay); err != nil || replay.Generated || replay.Runtime.Calls != 0 || replay.Runtime.Total != row.NamedTotal || replay.Runtime.Passed != row.NamedPassed || !reflect.DeepEqual(replay.Runtime.Traces, original.Runtime.Traces) {
			return receipt, fmt.Errorf("saved replay does not match verified observations: %s", replayPath)
		}
		reversePaths = append(reversePaths, replayPath)
		generatedBytes, readErr := os.ReadFile(filepath.Join(inputRoot, label, "composition", "generated.go"))
		if readErr != nil || original.Composition.GeneratedSHA != "sha256:"+fmt.Sprintf("%x", sha256.Sum256(generatedBytes)) {
			return receipt, fmt.Errorf("generated source digest does not match its execution: %s", label)
		}
	}
	declarationDigest, err := digestFiles(inputRoot, sourcePaths...)
	if err != nil {
		return receipt, err
	}
	generationDigest, err := digestFiles(inputRoot, generationPaths...)
	if err != nil {
		return receipt, err
	}
	reverseDigest, err := digestFiles(inputRoot, reversePaths...)
	if err != nil {
		return receipt, err
	}
	useCaseDigest, err := digestFiles(inputRoot, useCasePaths...)
	if err != nil {
		return receipt, err
	}
	headBytes, headErr := exec.CommandContext(ctx, "git", "-C", inputRoot, "rev-parse", "HEAD").Output()
	provenanceStatus := "UNKNOWN"
	provenanceEvidence := "UNKNOWN: verification was produced from a dirty or unidentified source checkout"
	if report.RepositoryHead != "" && headErr == nil && strings.TrimSpace(string(headBytes)) != report.RepositoryHead {
		provenanceEvidence = "UNKNOWN: current source checkout differs from verify-time HEAD=" + report.RepositoryHead
	}
	if report.RepositoryClean && report.RepositoryHead != "" && headErr == nil && strings.TrimSpace(string(headBytes)) == report.RepositoryHead && compilerSource != "" {
		summaryDigest, digestErr := digestFiles(inputRoot, "summary.json")
		if digestErr != nil {
			return receipt, digestErr
		}
		provenanceStatus = "PASS"
		provenanceEvidence = "HEAD=" + report.RepositoryHead + "; compiler_source=" + compilerSource + "; summary=" + summaryDigest
	}
	assessment := CompletenessAssessment{
		Domain:            "Gooo workbench including the invoice-approval example",
		AllowedInvestment: "local Gooo verification and evidence generation; no model training",
		ExcludedScope:     "unobserved user workloads, universal semantic completeness, production behavior, and unmeasured effect boundaries",
		DeclarationStatus: "PASS", DeclarationEvidence: declarationDigest,
		GenerationStatus: "PASS", GenerationEvidence: generationDigest,
		ReverseObservationStatus: "PASS", ReverseObservationEvidence: reverseDigest,
		UseCaseStatus: "PROGRESS", UseCaseEvidence: useCaseDigest + "; finite authored examples only; no independent-use claim",
		BoundaryStatus: "UNKNOWN", BoundaryEvidence: "UNMEASURED: external effects and deployment boundaries are not instrumented by this receipt",
		ProvenanceStatus: provenanceStatus, ProvenanceEvidence: provenanceEvidence,
	}
	want := CompletenessReceipt{
		Domain:            assessment.Domain,
		AllowedInvestment: assessment.AllowedInvestment, ExcludedScope: assessment.ExcludedScope,
		DeclarationStatus: assessment.DeclarationStatus, DeclarationEvidence: assessment.DeclarationEvidence,
		GenerationStatus: assessment.GenerationStatus, GenerationEvidence: assessment.GenerationEvidence,
		ReverseObservationStatus: assessment.ReverseObservationStatus, ReverseObservationEvidence: assessment.ReverseObservationEvidence,
		UseCaseStatus: assessment.UseCaseStatus, UseCaseEvidence: assessment.UseCaseEvidence,
		BoundaryStatus: assessment.BoundaryStatus, BoundaryEvidence: assessment.BoundaryEvidence,
		ProvenanceStatus: assessment.ProvenanceStatus, ProvenanceEvidence: assessment.ProvenanceEvidence,
		FirstUnresolvedStage: firstUnresolved(assessment),
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return receipt, err
	}
	assessmentBytes, err := json.Marshal(assessment)
	if err != nil {
		return receipt, err
	}
	if err = write(filepath.Join(root, "assessment.json"), append(assessmentBytes, '\n')); err != nil {
		return receipt, err
	}
	cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{
		map[string]any{"inputs": map[string]any{"Build": assessment}, "expected": map[string]any{"Echo": want}},
	}})
	if err != nil {
		return receipt, err
	}
	raw, composed, err := runRecipe(ctx, o, root, "completeness", "receipt", "", cases)
	if err != nil {
		return receipt, err
	}
	if err = actualFor(composed, "gooo-completeness://activity/echo", &receipt); err != nil {
		return receipt, err
	}
	if !reflect.DeepEqual(receipt, want) {
		return receipt, fmt.Errorf("Gooo completeness receipt differs from independently derived evidence")
	}
	counts, err := summarize(raw, "completeness", "deterministic")
	if err != nil || counts.NamedPassed != 1 || counts.NamedTotal != 1 || counts.FieldsPassed != counts.FieldsTotal || counts.ModelCalls != 0 {
		return receipt, fmt.Errorf("Gooo completeness receipt was not fully observed: %+v: %v", counts, err)
	}
	receiptPath := filepath.Join(root, "receipt.json")
	if err = save(receiptPath, receipt); err != nil {
		return receipt, err
	}
	if err = write(filepath.Join(root, "body-compose-result.json"), raw); err != nil {
		return receipt, err
	}
	recipeDir := filepath.Join(root, "receipt", "composition")
	compositionBytes, err := os.ReadFile(filepath.Join(recipeDir, "composition.json"))
	if err != nil {
		return receipt, err
	}
	var composition struct {
		Schema string `json:"schema"`
		Source string `json:"original_source_sha256"`
		Plan   struct {
			RecordTypes []struct {
				Name   string `json:"name"`
				ID     string `json:"id"`
				Fields []struct {
					Name string `json:"name"`
				} `json:"fields"`
			} `json:"record_types"`
		} `json:"plan"`
	}
	if err = json.Unmarshal(compositionBytes, &composition); err != nil {
		return receipt, fmt.Errorf("read Gooo completeness contract from compiler composition: %w", err)
	}
	if composition.Schema != "gooo/body-composition/v1" {
		return receipt, fmt.Errorf("compiler composition has unknown schema %q", composition.Schema)
	}
	var receiptType *struct {
		Name   string `json:"name"`
		ID     string `json:"id"`
		Fields []struct {
			Name string `json:"name"`
		} `json:"fields"`
	}
	for i := range composition.Plan.RecordTypes {
		if composition.Plan.RecordTypes[i].Name == "CompletenessReceipt" {
			receiptType = &composition.Plan.RecordTypes[i]
			break
		}
	}
	if receiptType == nil || receiptType.ID != "gooo://completeness/domain-completeness-receipt/v1" || len(receiptType.Fields) != reflect.TypeOf(receipt).NumField() {
		return receipt, fmt.Errorf("compiler receipt contract does not match the generated receipt structure")
	}
	fieldNames := make([]string, len(receiptType.Fields))
	for i, field := range receiptType.Fields {
		fieldNames[i] = field.Name
	}
	wantFieldNames := make([]string, 0, reflect.TypeOf(receipt).NumField())
	for i := 0; i < reflect.TypeOf(receipt).NumField(); i++ {
		wantFieldNames = append(wantFieldNames, strings.Split(reflect.TypeOf(receipt).Field(i).Tag.Get("json"), ",")[0])
	}
	if !reflect.DeepEqual(fieldNames, wantFieldNames) {
		return receipt, fmt.Errorf("Gooo receipt fields differ from the exported JSON structure")
	}
	if err = save(filepath.Join(root, "receipt-contract.json"), map[string]any{
		"schema": "gooo/completeness-receipt-contract/v1", "entity_id": receiptType.ID,
		"fields": fieldNames, "source_digest": composition.Source,
	}); err != nil {
		return receipt, err
	}
	replayBytes, err := command(ctx, o.Compiler, "body-compose", "--source", filepath.Join(recipeDir, "original.gooo"), "--cases", filepath.Join(recipeDir, "cases.json"), "--composition", filepath.Join(recipeDir, "composition.json"))
	if err != nil {
		return receipt, err
	}
	if err = write(filepath.Join(root, "replay.json"), replayBytes); err != nil {
		return receipt, err
	}
	var replay result
	if err = json.Unmarshal(replayBytes, &replay); err != nil || replay.Generated || replay.Runtime.Calls != 0 || replay.Runtime.Passed != 1 || replay.Runtime.Total != 1 {
		return receipt, fmt.Errorf("saved completeness receipt replay did not reproduce")
	}
	return receipt, nil
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

func referenceSignature(contract PublicInterface) string {
	inputs := make([]string, len(contract.Operation.Inputs))
	for index, input := range contract.Operation.Inputs {
		inputs[index] = input.Name
	}
	return contract.Operation.Activity + "(" + strings.Join(inputs, ", ") + ") -> " + contract.Operation.Output.Name
}

func referenceDetails(contract PublicInterface) string {
	var details strings.Builder
	fmt.Fprintf(&details, "## Package\n\nSource digest: `%s`\n\n", contract.SubjectDigest)
	fmt.Fprintf(&details, "- Package: `%s`\n- Namespace: `%s`\n", contract.Package.Name, contract.Package.Namespace)
	details.WriteString("\n## Inputs\n\n")
	for _, input := range contract.Operation.Inputs {
		fmt.Fprintf(&details, "- `%s` — `%s`\n", input.Name, input.ID)
	}
	fmt.Fprintf(&details, "\n## Output\n\n- `%s` — `%s`\n", contract.Operation.Output.Name, contract.Operation.Output.ID)
	fmt.Fprintf(&details, "\nInterface digest: `%s`", contract.Digest)
	return details.String()
}

func referenceDocument(contract PublicInterface) string {
	return "# " + contract.Operation.Activity + "\n\n`" + referenceSignature(contract) + "`\n\n" + referenceDetails(contract) +
		"\n\nThis page documents the declared interface and stable type identities. It does not claim to describe runtime behavior.\n"
}

// Reference generates a declaration-level API page. The compiler resolves the
// public signature and Gooo owns the prose template.
func Reference(ctx context.Context, o Options, packageDir, entry string) (string, error) {
	if o.Model != "" {
		return "", fmt.Errorf("reference uses a deterministic Gooo template; omit --model")
	}
	if packageDir == "" || entry == "" {
		return "", fmt.Errorf("reference requires --package and --entry")
	}
	root, err := newOutput(o.Out)
	if err != nil {
		return "", err
	}
	interfaceData, err := command(ctx, o.Compiler, "emit", "--kind", "operation-interface", "--entry", entry, packageDir)
	if len(interfaceData) > 0 {
		if writeErr := write(filepath.Join(root, "operation-interface.json"), interfaceData); writeErr != nil {
			return "", writeErr
		}
	}
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(interfaceData))
	decoder.DisallowUnknownFields()
	var contract PublicInterface
	if err = decoder.Decode(&contract); err != nil {
		return "", fmt.Errorf("decode compiler operation interface: %w", err)
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return "", fmt.Errorf("operation interface contains trailing JSON")
	}
	if contract.Schema != "gooo/operation-interface/v1" || contract.Decision != "PASS" ||
		contract.Resolution != "INTERFACE_ONLY" || contract.Operation.Activity != entry ||
		contract.SubjectDigest == "" || contract.Package.Name == "" || contract.Package.Namespace == "" || contract.Digest == "" ||
		contract.Kind != "operation-interface" || contract.Effects.RepositoryWrites != 0 || contract.Effects.MutationAuthority ||
		contract.Operation.Output.Name == "" || contract.Operation.Output.ID == "" {
		return "", fmt.Errorf("compiler did not provide a complete public operation interface")
	}
	for _, input := range contract.Operation.Inputs {
		if input.Name == "" || input.ID == "" {
			return "", fmt.Errorf("compiler operation interface has an incomplete input type")
		}
	}
	inputs := map[string]any{
		"Reference.input0": contract.Operation.Activity,
		"Reference.input1": referenceSignature(contract),
		"Reference.input2": referenceDetails(contract),
	}
	want := referenceDocument(contract)
	cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{
		map[string]any{"inputs": inputs, "expected": map[string]string{"Echo": want}},
	}})
	if err != nil {
		return "", err
	}
	raw, result, err := runRecipe(ctx, o, root, "reference", "reference", "", cases)
	if err != nil {
		return "", err
	}
	var actual string
	if err = actualFor(result, "apidocs://activity/reference", &actual); err != nil {
		return "", err
	}
	if actual != want {
		return actual, fmt.Errorf("Gooo reference output differs from the independently rendered contract fixture")
	}
	counts, err := summarize(raw, "reference", "deterministic")
	if err != nil || counts.NamedPassed != 1 || counts.NamedTotal != 1 || counts.ModelCalls != 0 {
		return actual, fmt.Errorf("reference output observation was incomplete: %+v: %v", counts, err)
	}
	if err = write(filepath.Join(root, "reference.md"), []byte(actual)); err != nil {
		return "", err
	}
	recipeDir := filepath.Join(root, "reference", "composition")
	replay, err := command(ctx, o.Compiler, "body-compose", "--source", filepath.Join(recipeDir, "original.gooo"), "--cases", filepath.Join(recipeDir, "cases.json"), "--composition", filepath.Join(recipeDir, "composition.json"))
	if err != nil {
		return "", err
	}
	if err = write(filepath.Join(root, "replay.json"), replay); err != nil {
		return "", err
	}
	replayCounts, err := summarize(replay, "reference", "deterministic")
	if err != nil || replayCounts.NamedPassed != counts.NamedPassed || replayCounts.NamedTotal != counts.NamedTotal || replayCounts.ModelCalls != 0 {
		return "", fmt.Errorf("saved API reference replay differs: %+v: %v", replayCounts, err)
	}
	if err = save(filepath.Join(root, "reference-receipt.json"), map[string]any{
		"schema": "gooo/ecosystem-api-reference/v1", "decision": "PASS",
		"compiler_interface_digest": contract.Digest, "entry": entry,
		"source_digest":    contract.SubjectDigest,
		"input_type_count": len(contract.Operation.Inputs), "output_type_count": 1,
		"reference_sha256":       fmt.Sprintf("%x", sha256.Sum256([]byte(actual))),
		"scope":                  "Gooo-rendered declaration-level reference; runtime behavior is not described",
		"named_passed":           counts.NamedPassed,
		"named_total":            counts.NamedTotal,
		"saved_replay_verified":  true,
		"generation_model_calls": counts.ModelCalls,
	}); err != nil {
		return "", err
	}
	return actual, write(filepath.Join(root, "body-compose-result.json"), raw)
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
