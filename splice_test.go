package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpliceRequiresExplicitBoundedRequest(t *testing.T) {
	valid := `{"source":"","before":"","expected":"","after":"","replacement":"가"}`
	if _, err := ReadSpliceRequest([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"", "null", valid + valid, strings.Replace(valid, `"before":"",`, "", 1), strings.Replace(valid, `"before":""`, `"before":null`, 1), strings.Replace(valid, `"before":""`, `"before":true`, 1), strings.Replace(valid, `"before"`, `"unknown"`, 1), strings.Replace(valid, "가", strings.Repeat("가", 342), 1)} {
		if _, err := ReadSpliceRequest([]byte(raw)); err == nil {
			t.Fatal("invalid request accepted", raw)
		}
	}
}

func TestNativeSpliceRunsActualRequestsAndSavedReplay(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	for i, tc := range []struct {
		request SpliceRequest
		want    SpliceEdit
	}{
		{SpliceRequest{Source: "a+b", Before: "a", Expected: "+", After: "b", Replacement: "***"}, SpliceEdit{Changed: true, Source: "a***b", Bytes: 5}},
		{SpliceRequest{Source: "a+b", Before: "a", Expected: "-", After: "b", Replacement: "*"}, SpliceEdit{Source: "a+b", Bytes: 3}},
		{SpliceRequest{Source: "한", Expected: "한", Replacement: "한"}, SpliceEdit{Source: "한", Bytes: 3}},
		{SpliceRequest{Source: "px", Before: "p", Expected: "x", Replacement: strings.Repeat("z", 1024)}, SpliceEdit{Source: "px", Bytes: 2}},
	} {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			o := Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "splice")}
			r, err := SpliceSource(context.Background(), o, tc.request)
			if err != nil || r.Edit != tc.want {
				t.Fatal("Gooo source splice", r.Edit, err)
			}
			if !r.Observation.ReplayVerified || r.Observation.SelectionPassed != 24 || r.Observation.NamedTotal != 0 || r.Observation.ModelCalls != 0 {
				t.Fatal("input-only request acquired an unsupported score", r.Observation)
			}
		})
	}
}

func TestSpliceOutputRequiresConstructedIdentity(t *testing.T) {
	var r result
	raw := `{"composition":{"steps":[{"generation":{"report":{"activity_id":"actual"}}}]},"runtime":{"traces":[{"deliveries":[{"activity_id":"other","actual":{"changed":true,"source":"x","bytes":1}}]}]}}`
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	var edit SpliceEdit
	if err := spliceOutput(r, &edit); err == nil {
		t.Fatal("unrelated delivery accepted")
	}
}
