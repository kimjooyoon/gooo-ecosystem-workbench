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

func graphAuditFixture(t *testing.T) ([]byte, FeatureAuditInput) {
	t.Helper()
	raw, err := os.ReadFile("examples/feature-audit/filename-graph-order.json")
	if err != nil {
		t.Fatal(err)
	}
	input, err := readFeatureAuditInput(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, input
}

func TestFeatureAuditGraphDistinguishesFilenameArrangements(t *testing.T) {
	_, input := graphAuditFixture(t)
	r, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || r.Rows != 8 || r.Families != 1 || r.DistinctFeatures != 8 || r.ConflictingGroups != 0 ||
		r.MaximumCompatible != 8 || r.ModelObserved || r.ModelCalls != 0 {
		t.Fatal("source graph audit differs", r, err)
	}
	again, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || !reflect.DeepEqual(r, again) {
		t.Fatal("graph audit is not reproducible", err)
	}
	for i := range input.Cases {
		input.Cases[i].AcceptedMasks = []uint16{0}
	}
	relabeled, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || relabeled.DistinctFeatures != 8 || relabeled.MaximumCompatible != 8 || relabeled.ConflictingGroups != 0 {
		t.Fatal("relabeled graph audit failed", err)
	}
	for i := range r.Groups {
		if r.Groups[i].FeatureSHA256 != relabeled.Groups[i].FeatureSHA256 {
			t.Fatal("labels changed features")
		}
	}
	_, input = graphAuditFixture(t)
	input.Cases[1].Graph = input.Cases[0].Graph
	conflict, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || conflict.MaximumCompatible != 7 || conflict.ConflictingGroups != 1 || conflict.DistinctFeatures != 7 {
		t.Fatal("graph contract hid conflicting labels", conflict, err)
	}
}

func TestFeatureAuditGraphRequiresOwnShapeAndWeights(t *testing.T) {
	_, expression := featureAuditFixture(t)
	_, origin := originAuditFixture(t)
	for _, mutate := range []func(*FeatureAuditInput){
		func(i *FeatureAuditInput) { i.FeatureVersion = jointdecision.RecordSharedFeatureVersion },
		func(i *FeatureAuditInput) { i.FeatureVersion = jointdecision.RecordOriginSharedFeatureVersion },
		func(i *FeatureAuditInput) { i.Cases[0].Choices = expression.Cases[0].Choices },
		func(i *FeatureAuditInput) { i.Cases[0].OriginChoices = origin.Cases[0].OriginChoices },
		func(i *FeatureAuditInput) { i.Cases[0].Graph = nil },
	} {
		_, input := graphAuditFixture(t)
		mutate(&input)
		raw, _ := json.Marshal(input)
		if _, err := readFeatureAuditInput(raw); err == nil {
			t.Fatal("mixed graph contract accepted")
		}
		if _, err := auditFeatureRows(context.Background(), input, nil); err == nil {
			t.Fatal("mixed graph projected")
		}
	}
	_, input := graphAuditFixture(t)
	model, err := loadAuditModel("models/shared-qat/model.json", expression.FeatureVersion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auditFeatureRows(context.Background(), input, model); err == nil {
		t.Fatal("v1 weights used for v3")
	}
	if _, err := loadAuditModel("models/shared-qat/model.json", input.FeatureVersion); err == nil {
		t.Fatal("v1 metadata loaded as v3")
	}
	input.Cases[0].Graph.Nodes[0].Parents[0] = 1
	if _, err := auditFeatureRows(context.Background(), input, nil); err == nil {
		t.Fatal("cyclic graph accepted")
	}
}

func TestFeatureAuditGraphNativeGoooAssessment(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER to execute the Gooo assessment")
	}
	raw, _ := graphAuditFixture(t)
	r, err := AuditRecordFeatures(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "audit")}, raw)
	if err != nil || r.Policy.Code != "input-consistent" || r.Policy.Action != "evaluate-chooser" ||
		r.MaximumCompatible != 8 || r.ModelObserved || r.ModelCalls != 0 || !r.PolicyReplayed || r.PolicyCompilerSHA == "" {
		t.Fatal("graph audit did not reach Gooo assessment", r, err)
	}
}
