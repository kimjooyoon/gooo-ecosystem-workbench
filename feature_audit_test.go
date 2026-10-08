package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func featureAuditFixture(t *testing.T) ([]byte, FeatureAuditInput) {
	t.Helper()
	raw, err := os.ReadFile("examples/feature-audit/filename-order.json")
	if err != nil {
		t.Fatal(err)
	}
	input, err := readFeatureAuditInput(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, input
}

func TestFeatureAuditFindsIrrecoverableChoiceCollisions(t *testing.T) {
	_, input := featureAuditFixture(t)
	model, err := jointdecision.LoadRecordSharedThree("models/shared-qat/model.json")
	if err != nil {
		t.Fatal(err)
	}
	report, err := auditFeatureRows(context.Background(), input, model)
	if err != nil {
		t.Fatal(err)
	}
	if report.Rows != 8 || report.Families != 1 || report.DistinctFeatures != 2 || report.ConflictingGroups != 2 ||
		report.MaximumCompatible != 2 || report.ModelSatisfied != 1 || report.ModelCalls != 8 {
		t.Fatal("candidate order audit differs", report)
	}
	for _, group := range report.Groups {
		if len(group.Rows) != 4 || group.Maximum != 1 {
			t.Fatal("incompatible labels did not share exact feature bits", group)
		}
	}
	again, err := auditFeatureRows(context.Background(), input, model)
	if err != nil || !reflect.DeepEqual(report, again) {
		t.Fatal("audit changed on replay", err)
	}
}

func TestFeatureAuditAcceptsMultipleEquivalentLabels(t *testing.T) {
	_, input := featureAuditFixture(t)
	a := input.Cases[0]
	b := a
	b.ID = "same-features-other-labels"
	a.AcceptedMasks, b.AcceptedMasks = []uint16{0, 1}, []uint16{1, 2}
	input.Cases = []FeatureAuditCase{a, b}
	report, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || report.DistinctFeatures != 1 || report.MaximumCompatible != 2 || report.ConflictingGroups != 0 || report.ModelObserved {
		t.Fatal("shared acceptable label was treated as a conflict", report, err)
	}
}

func TestFeatureAuditRejectsUnknownAndContradictoryInputs(t *testing.T) {
	_, input := featureAuditFixture(t)
	for _, mutate := range []func(*FeatureAuditInput){
		func(i *FeatureAuditInput) { i.FeatureVersion = "different" },
		func(i *FeatureAuditInput) { i.LabelSource = "" },
		func(i *FeatureAuditInput) { i.Cases = nil },
		func(i *FeatureAuditInput) { i.Cases[1].ID = i.Cases[0].ID },
		func(i *FeatureAuditInput) { i.Cases[0].Family = "" },
		func(i *FeatureAuditInput) { i.Cases[0].Choices = i.Cases[0].Choices[:2] },
		func(i *FeatureAuditInput) {
			i.Cases[0].Choices = append(append([]jointdecision.RecordChoice(nil), i.Cases[0].Choices...), i.Cases[0].Choices[0])
		},
		func(i *FeatureAuditInput) { i.Cases[0].AcceptedMasks = nil },
		func(i *FeatureAuditInput) { i.Cases[0].AcceptedMasks = []uint16{8} },
		func(i *FeatureAuditInput) { i.Cases[0].AcceptedMasks = []uint16{1, 1} },
	} {
		copy := input
		copy.Cases = append([]FeatureAuditCase(nil), input.Cases...)
		mutate(&copy)
		raw, _ := json.Marshal(copy)
		if _, err := readFeatureAuditInput(raw); err == nil {
			t.Fatal("invalid audit input accepted", copy)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := auditFeatureRows(ctx, input, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("audit did not retain cancellation", err)
	}
}

func TestFeatureAuditNativeGoooAssessment(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER to execute the Gooo assessment")
	}
	raw, _ := featureAuditFixture(t)
	root := filepath.Join(t.TempDir(), "audit")
	report, err := AuditRecordFeatures(context.Background(), Options{Compiler: compiler, Model: "builtin", Out: root}, raw)
	if err != nil || report.Policy.Code != "representation-collision" || report.Policy.Action != "preserve-distinguishing-source-facts" ||
		report.MaximumCompatible != 2 || report.ModelSatisfied != 1 || report.PolicySourceSHA256 == "" || !report.PolicyReplayed {
		t.Fatal("Gooo assessment differs", report, err)
	}
	if _, err := os.Stat(filepath.Join(root, "assessment", "composition", "composition.json")); err != nil {
		t.Fatal("saved assessment missing", err)
	}
	for _, test := range []struct {
		rows, maximum, matched int
		observed               bool
		code                   string
	}{
		{0, 0, 0, false, "invalid-counts"},
		{2, 1, 2, true, "invalid-counts"},
		{2, 2, 0, false, "input-consistent"},
		{2, 2, 1, true, "chooser-gap"},
		{2, 2, 2, true, "observed-fit"},
	} {
		out := t.TempDir()
		value := map[string]any{"rows": test.rows, "maximum": test.maximum, "model_observed": test.observed, "model_matched": test.matched}
		cases := runtimeCase(map[string]any{"ObservationEcho": value})
		_, r, err := runRecipe(context.Background(), Options{Compiler: compiler}, out, "feature-audit", "assessment", "", cases)
		if err != nil {
			t.Fatal(err)
		}
		var policy FeatureAuditPolicy
		if err := actualFor(r, "featureaudit://activity/assess", &policy); err != nil || policy.Code != test.code || r.Runtime.Calls != 0 {
			t.Fatal("native policy branch differs", policy, err)
		}
	}
}
