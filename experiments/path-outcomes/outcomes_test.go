package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) (sourceExport, observation) {
	t.Helper()
	source, err := os.ReadFile("testdata/source.gooo.fixture")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/original-context.json")
	if err != nil {
		t.Fatal(err)
	}
	e, p, err := bindExport(raw, source)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := os.ReadFile("testdata/targets.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := observe(context.Background(), e, p, cases)
	if err != nil {
		t.Fatal(err)
	}
	return e, r
}

func TestJointTargetsPreserveAlternativeImplementations(t *testing.T) {
	_, r := fixture(t)
	if !reflect.DeepEqual(r.Compatible, []int{1, 2}) || r.Declared != 8 || r.MarginalProduct != 4 || r.InvalidProduct != 2 {
		t.Fatalf("joint targets lost the branch/comparison interaction: %+v", r)
	}
	if !reflect.DeepEqual(r.FiniteGroups, [][]int{{0, 3}, {1, 2}, {4, 5, 6, 7}}) {
		t.Fatalf("finite output groups differ: %v", r.FiniteGroups)
	}
	for _, mask := range r.Compatible {
		if got := *r.Candidates[mask].Outcomes[7].Actual; got != 9007199254740993 {
			t.Fatalf("exact int64 lost: %d", got)
		}
	}
	if r.NativeParity || r.NativeChecked != 0 || r.TrainingPerformed || r.ModelCalls != 0 {
		t.Fatal("interpreted enumeration invented native/training/model evidence")
	}
}

func TestNativeProjectionParityAndMissingEvidence(t *testing.T) {
	e, r := fixture(t)
	if err := native(context.Background(), t.TempDir(), e.Plan.Base.Name, &r); err != nil {
		t.Fatal(err)
	}
	if !r.NativeParity || r.NativeChecked != 72 {
		t.Fatalf("native accounting differs: %d %t", r.NativeChecked, r.NativeParity)
	}
	if err := compareNative([]byte("[]"), &r); err == nil {
		t.Fatal("empty native output claimed parity")
	}
	if r.NativeParity || r.NativeChecked != 0 {
		t.Fatal("failed comparison retained stale success")
	}
	rows := make([]nativeRow, len(r.Candidates))
	for i, c := range r.Candidates {
		rows[i].Mask = c.Mask
		for _, o := range c.Outcomes {
			rows[i].Values = append(rows[i].Values, nativeValue{Value: *o.Actual})
		}
	}
	rows[1].Values[7].Value = 9007199254740992
	raw, _ := json.Marshal(rows)
	if err := compareNative(raw, &r); err == nil {
		t.Fatal("one-bit int64 projection discrepancy accepted")
	}
}

func TestNativeFailureCannotMasqueradeAsZero(t *testing.T) {
	_, r := fixture(t)
	row := nativeRow{Mask: 0}
	for _, o := range r.Candidates[0].Outcomes {
		row.Values = append(row.Values, nativeValue{Value: *o.Actual})
	}
	row.Values[3].Failure = "observed native fault"
	raw, _ := json.Marshal([]nativeRow{row})
	if err := compareNative(raw, &r); err == nil || !strings.Contains(err.Error(), "input 0") {
		t.Fatal("native fault confused with a successful zero result", err)
	}
}

func TestBindingRejectsChangedPlanAndIntent(t *testing.T) {
	source, _ := os.ReadFile("testdata/source.gooo.fixture")
	raw, _ := os.ReadFile("testdata/original-context.json")
	if _, _, err := bindExport(raw, append(source, '\n')); err == nil {
		t.Fatal("changed source accepted under original receipt")
	}
	for _, change := range []func(*sourceExport){
		func(e *sourceExport) { e.Plan.Decisions[0].Intent = "choose another option" },
		func(e *sourceExport) { e.Inputs[0].Text += " extra" },
		func(e *sourceExport) { e.Inputs[0].ID = e.Inputs[1].ID },
		func(e *sourceExport) { e.Predictions = 1 },
		func(e *sourceExport) { e.Context.Status = "DECLINED_TO_DETERMINISTIC" },
	} {
		var e sourceExport
		if err := json.Unmarshal(raw, &e); err != nil {
			t.Fatal(err)
		}
		change(&e)
		changed, _ := json.Marshal(e)
		if _, _, err := bindExport(changed, source); err == nil {
			t.Fatal("changed context accepted")
		}
	}
}

func TestCasesCannotInflateOrRoundTargets(t *testing.T) {
	for _, raw := range []string{
		`[]`, `[{"input":1}]`, `[{"input":null,"expected":0}]`,
		`[{"input":1,"expected":1},{"input":1,"expected":2}]`,
		`[{"input":9007199254740993.0,"expected":0}]`,
		`[{"input":9223372036854775808,"expected":0}]`,
	} {
		if _, err := decodeCases([]byte(raw)); err == nil {
			t.Fatalf("invalid target accepted: %s", raw)
		}
	}
	cases, err := decodeCases([]byte(`[{"input":9007199254740993,"expected":-9007199254740993}]`))
	if err != nil || cases[0].Input != 9007199254740993 || cases[0].Expected != -9007199254740993 {
		t.Fatal("exact integer target changed", cases, err)
	}
}

func TestCancelledObservationAndNoCompatibleTarget(t *testing.T) {
	e, _ := fixture(t)
	raw, _ := os.ReadFile("testdata/original-context.json")
	source, _ := os.ReadFile("testdata/source.gooo.fixture")
	_, p, err := bindExport(raw, source)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := observe(ctx, e, p, []byte(`[{"input":0,"expected":0}]`)); err != context.Canceled {
		t.Fatal("cancelled observation continued", err)
	}
	r, err := observe(context.Background(), e, p, []byte(`[{"input":0,"expected":99}]`))
	if err != nil || len(r.Compatible) != 0 || r.MarginalProduct != 0 || r.InvalidProduct != 0 {
		t.Fatal("unsatisfied corpus invented a target", r.Compatible, err)
	}
	e.Plan.Decisions = append(e.Plan.Decisions, e.Plan.Decisions...)
	e.Plan.Decisions = append(e.Plan.Decisions, e.Plan.Decisions[0])
	if _, err := observe(context.Background(), e, p, []byte(`[{"input":0,"expected":0}]`)); err == nil || !strings.Contains(err.Error(), "64 combinations") {
		t.Fatal("partial palette presented as exhaustive targets", err)
	}
}
