package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func originAuditFixture(t *testing.T) ([]byte, FeatureAuditInput) {
	t.Helper()
	raw, err := os.ReadFile("examples/feature-audit/filename-origin-order.json")
	if err != nil {
		t.Fatal(err)
	}
	input, err := readFeatureAuditInput(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, input
}

func TestFeatureAuditOriginRelationsRetainTheOperatorGap(t *testing.T) {
	_, input := originAuditFixture(t)
	r, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || r.Rows != 8 || r.Families != 1 || r.DistinctFeatures != 4 || r.ConflictingGroups != 4 ||
		r.MaximumCompatible != 4 || r.ModelObserved || r.ModelCalls != 0 {
		t.Fatal("source origin audit differs", r, err)
	}
	for i, group := range r.Groups {
		if group.Maximum != 1 || !reflect.DeepEqual(group.Rows, []string{input.Cases[2*i].ID, input.Cases[2*i+1].ID}) {
			t.Fatal("and/or alternatives should still collide", group)
		}
	}
	again, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || !reflect.DeepEqual(r, again) {
		t.Fatal("origin audit is not reproducible", again, err)
	}
	// Projection consumes source context only. Altering the supplied accepted
	// masks affects compatibility counts without changing any feature hash.
	for i := range input.Cases {
		input.Cases[i].AcceptedMasks = []uint16{0}
	}
	relabeled, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || relabeled.MaximumCompatible != 8 || relabeled.ConflictingGroups != 0 {
		t.Fatal("equal accepted masks should be compatible", relabeled, err)
	}
	for i := range r.Groups {
		if r.Groups[i].FeatureSHA256 != relabeled.Groups[i].FeatureSHA256 {
			t.Fatal("labels leaked into model features")
		}
	}
}

func TestFeatureAuditOriginRequiresItsOwnContextAndWeights(t *testing.T) {
	_, origin := originAuditFixture(t)
	_, expression := featureAuditFixture(t)
	for _, mutate := range []func(*FeatureAuditInput){
		func(i *FeatureAuditInput) { i.FeatureVersion = jointdecision.RecordSharedFeatureVersion },
		func(i *FeatureAuditInput) { i.Cases[0].Choices = expression.Cases[0].Choices },
		func(i *FeatureAuditInput) { i.Cases[0].OriginChoices = nil },
		func(i *FeatureAuditInput) { i.Cases[0].OriginChoices = i.Cases[0].OriginChoices[:2] },
	} {
		copy := origin
		copy.Cases = append([]FeatureAuditCase(nil), origin.Cases...)
		mutate(&copy)
		raw, _ := json.Marshal(copy)
		if _, err := readFeatureAuditInput(raw); err == nil {
			t.Fatal("mixed feature shape was accepted")
		}
		if _, err := auditFeatureRows(context.Background(), copy, nil); err == nil {
			t.Fatal("projection accepted a mismatched row")
		}
	}
	model, err := loadAuditModel("models/shared-qat/model.json", expression.FeatureVersion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auditFeatureRows(context.Background(), origin, model); err == nil {
		t.Fatal("v1 weights were used for v2 features")
	}
	if _, err := loadAuditModel("models/shared-qat/model.json", origin.FeatureVersion); err == nil {
		t.Fatal("v1 model metadata was accepted for v2")
	}
	origin.Cases[0].OriginChoices[0].Origins[0][0] = 513
	if _, err := auditFeatureRows(context.Background(), origin, nil); err == nil {
		t.Fatal("ancestor count above compiler bound was accepted")
	}
}

func TestFeatureAuditOriginNativeGoooAssessment(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER to execute the Gooo assessment")
	}
	raw, _ := originAuditFixture(t)
	r, err := AuditRecordFeatures(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "audit")}, raw)
	if err != nil || r.Policy.Code != "representation-collision" || r.MaximumCompatible != 4 ||
		r.ModelObserved || r.ModelCalls != 0 || !r.PolicyReplayed || r.PolicyCompilerSHA == "" {
		t.Fatal("origin source audit did not reach the Gooo assessment", r, err)
	}
}
