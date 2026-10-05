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

	jev "github.com/kimjooyoon/gooo-jev/gooo"
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

func TestReferenceDocumentUsesDeclaredNamesAndStableTypeIDs(t *testing.T) {
	contract := PublicInterface{}
	contract.Package.Name, contract.Package.Namespace = "billing", "billing"
	contract.SubjectDigest = "sha256:subject"
	contract.Operation.Activity = "PayOrder"
	contract.Operation.Inputs = []TypeReference{{Name: "Order", ID: "urn:gooo:billing:order"}, {Name: "Method", ID: "urn:gooo:billing:method"}}
	contract.Operation.Output = TypeReference{Name: "Receipt", ID: "urn:gooo:billing:receipt"}
	contract.Digest = "sha256:interface"
	doc := referenceDocument(contract)
	for _, want := range []string{"# PayOrder", "`PayOrder(Order, Method) -> Receipt`", "Source digest: `sha256:subject`", "urn:gooo:billing:order", "urn:gooo:billing:method", "urn:gooo:billing:receipt", "Interface digest: `sha256:interface`", "does not claim to describe runtime behavior"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("reference omits %q: %s", want, doc)
		}
	}
}
func TestCompletenessKeepsUnknownAsFirstUnresolved(t *testing.T) {
	assessment := CompletenessAssessment{
		DeclarationStatus: "PASS", GenerationStatus: "PASS", ReverseObservationStatus: "PASS",
		UseCaseStatus: "PROGRESS", BoundaryStatus: "UNKNOWN", ProvenanceStatus: "PASS",
	}
	if got := firstUnresolved(assessment); got != "use_case" {
		t.Fatalf("first unresolved stage = %q, want use_case", got)
	}
	assessment.UseCaseStatus = "PASS"
	if got := firstUnresolved(assessment); got != "boundary" {
		t.Fatalf("UNKNOWN was not retained as unresolved: %q", got)
	}
}

func TestCapabilityEvidenceKeepsCatalogAndExecutionBoundariesSeparate(t *testing.T) {
	cases := []struct {
		status   jev.CapabilityQueryState
		bound    bool
		coverage string
		stage    string
	}{
		{status: jev.CapabilityQueryAvailable, bound: true, coverage: "PROGRESS", stage: "generation"},
		{status: jev.CapabilityQueryAvailable, bound: false, coverage: "UNKNOWN", stage: "declaration_binding"},
		{status: jev.CapabilityQueryDeferred, bound: true, coverage: "UNKNOWN", stage: "external_boundary"},
		{status: jev.CapabilityQueryUnknown, bound: false, coverage: "UNKNOWN", stage: "capability_catalog"},
	}
	for _, tc := range cases {
		if got := capabilityUseCaseCoverage(tc.status, tc.bound); got != tc.coverage {
			t.Fatalf("coverage(%s, %v) = %s, want %s", tc.status, tc.bound, got, tc.coverage)
		}
		if got := capabilityFirstUnresolved(tc.status, tc.bound); got != tc.stage {
			t.Fatalf("first unresolved(%s, %v) = %s, want %s", tc.status, tc.bound, got, tc.stage)
		}
	}
}

func TestNativeCapabilityDiscoveryIsGoooAssessedAndReplayed(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual Gooo/native verification")
	}
	declaration, err := os.ReadFile("examples/catalog/operations.gooo")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		query   string
		want    string
		stage   string
		useCase string
		bound   bool
	}{
		{query: "How do I generate code?", want: "AVAILABLE", stage: "generation", useCase: "PROGRESS", bound: true},
		{query: "코드 생성은 어떻게 해?", want: "AVAILABLE", stage: "generation", useCase: "PROGRESS", bound: true},
		{query: "How do I generate code?", want: "AVAILABLE", stage: "declaration_binding", useCase: "UNKNOWN", bound: false},
		{query: "Can gooo execute this?", want: "DEFERRED", stage: "external_boundary", useCase: "UNKNOWN", bound: true},
		{query: "quantum breakfast compiler", want: "UNKNOWN", stage: "capability_catalog", useCase: "UNKNOWN", bound: false},
	}
	for i, tc := range cases {
		declarationInput := ""
		if tc.bound {
			declarationInput = string(declaration)
		}
		out, err := DiscoverCapability(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), fmt.Sprintf("discover-%d", i))}, tc.query, declarationInput)
		if err != nil {
			t.Fatal(err)
		}
		bound := "false"
		if tc.bound {
			bound = "true"
		}
		if out.Trail.Response.Status != jev.CapabilityQueryState(tc.want) || out.Assessment.CatalogState != tc.want ||
			out.Assessment.FirstUnresolvedStage != tc.stage || out.Assessment.RealUseCaseCoverage != tc.useCase ||
			out.Assessment.DeclarationBound != bound || out.Assessment.ExecutionAttempted != "false" ||
			out.Assessment.ProviderInvocations != "0" || out.NamedPassed != 2 || out.NamedTotal != 2 ||
			out.RecordFieldsPassed != 22 || out.RecordFieldsTotal != 22 || !out.SavedReplayVerified || out.GenerationModelCalls != 0 {
			t.Fatalf("capability discovery evidence does not match bounded expectation: %+v", out)
		}
		if out.JEVModuleVersion == "" {
			t.Fatal("capability discovery did not retain its JEV implementation version")
		}
	}
}
func TestCompletenessRejectsASubsetOfVerificationRecipes(t *testing.T) {
	input := t.TempDir()
	report := map[string]any{
		"schema": "gooo/ecosystem-workbench-verification/v1", "scope": "test",
		"recipes": []Summary{{Recipe: "stdlib", Mode: "deterministic"}},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(input, "summary.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = CompletenessReceiptFor(context.Background(), Options{Out: filepath.Join(t.TempDir(), "out")}, input); err == nil {
		t.Fatal("receipt accepted a partial set of workbench recipes")
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
	if len(summaries) != 7 {
		t.Fatal(summaries)
	}
	var domainCaseFound bool
	var ciPlanFound bool
	for _, s := range summaries {
		if s.NamedPassed != s.NamedTotal || !s.ReplayVerified {
			t.Fatal(s)
		}
		if s.Recipe == "invoice-approval" {
			domainCaseFound = s.Mode == "deterministic" && s.NamedPassed == 6 && s.NamedTotal == 6 && s.FieldsPassed == 18 && s.FieldsTotal == 18
		}
		if s.Recipe == "ci-plan" {
			ciPlanFound = s.Mode == "deterministic" && s.NamedPassed == 24 && s.NamedTotal == 24 && s.FieldsPassed == 96 && s.FieldsTotal == 96 && s.ModelCalls == 0
		}
	}
	if !domainCaseFound {
		t.Fatal("invoice approval did not verify all three actual outputs and all eighteen fields")
	}
	if !ciPlanFound {
		t.Fatal("CI plan did not verify its 12 fixed cases, 24 activity outputs and 96 fields without model calls")
	}
	receipt, e := CompletenessReceiptFor(ctx, Options{Compiler: compiler, Out: filepath.Join(root, "receipt")}, filepath.Join(root, "verify"))
	if e != nil || receipt.DeclarationStatus != "PASS" || receipt.GenerationStatus != "PASS" ||
		receipt.ReverseObservationStatus != "PASS" || receipt.UseCaseStatus != "PROGRESS" ||
		receipt.BoundaryStatus != "UNKNOWN" || receipt.FirstUnresolvedStage != "use_case" {
		t.Fatal(receipt, e)
	}
	resultPath := filepath.Join(root, "verify", "stdlib-deterministic", "result.json")
	original, e := os.ReadFile(resultPath)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(resultPath, append(original, ' '), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e = CompletenessReceiptFor(ctx, Options{Compiler: compiler, Out: filepath.Join(root, "tampered-receipt")}, filepath.Join(root, "verify")); e == nil {
		t.Fatal("receipt accepted a result whose digest differs from its verification summary")
	}
	for _, profile := range []string{"record", "scalar"} {
		p, e := Scaffold(ctx, Options{Compiler: compiler, Model: "builtin", Out: filepath.Join(root, profile)}, profile)
		if e != nil || p.Filename != "main.gooo" || !strings.Contains(p.Source, "activity Identity") {
			t.Fatal(p, e)
		}
	}
	reference, e := Reference(ctx, Options{Compiler: compiler, Out: filepath.Join(root, "reference")}, "examples/catalog", "ApproveInvoice")
	if e != nil || !strings.Contains(reference, "`ApproveInvoice(Invoice, Reviewer) -> Approval`") ||
		!strings.Contains(reference, "urn:gooo:example:checkout:reviewer") ||
		!strings.Contains(reference, "Interface digest: `sha256:") {
		t.Fatal(reference, e)
	}
	var referenceReceipt struct {
		Decision    string `json:"decision"`
		NamedPassed int    `json:"named_passed"`
		NamedTotal  int    `json:"named_total"`
		Replay      bool   `json:"saved_replay_verified"`
		ModelCalls  int    `json:"generation_model_calls"`
	}
	receiptBytes, e := os.ReadFile(filepath.Join(root, "reference", "reference-receipt.json"))
	if e != nil || json.Unmarshal(receiptBytes, &referenceReceipt) != nil ||
		referenceReceipt.Decision != "PASS" || referenceReceipt.NamedPassed != 1 ||
		referenceReceipt.NamedTotal != 1 || !referenceReceipt.Replay || referenceReceipt.ModelCalls != 0 {
		t.Fatal(string(receiptBytes), e)
	}
	_, e = Reference(ctx, Options{Compiler: compiler, Out: filepath.Join(root, "unknown-entry")}, "examples/catalog", "MissingActivity")
	if e == nil {
		t.Fatal("missing package activity produced API documentation")
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
