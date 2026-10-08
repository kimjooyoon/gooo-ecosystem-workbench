package main

import (
	"encoding/json"
	"testing"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func TestObserveBindsSourceGraphAndUniqueFiniteLabel(t *testing.T) {
	source := []byte("synthetic envelope test")
	graph := jointdecision.RecordGraphInput{Nodes: []jointdecision.RecordGraphNode{
		{Kind: "input", Input: 1, InputType: "bool"},
	}}
	for i := range graph.Choices {
		graph.Choices[i] = jointdecision.RecordGraphChoice{
			Field: "value", First: "input", Second: "input", Intent: "keep",
			FieldID: "test://value", Roots: [2]uint16{1, 1},
		}
	}
	text, err := jointdecision.EncodeRecordGraphThree(graph)
	if err != nil {
		t.Fatal(err)
	}
	zero, mask := 0, uint16(7)
	var exported contextExport
	exported.SourceSHA, exported.ContractSHA = digest(source), "sha256:contract"
	exported.Predictions, exported.Tests = &zero, &zero
	exported.Context.Status, exported.Context.Text, exported.Context.SHA256 = "ENCODED", text, digest([]byte(text))
	exported.Context.Feature = jointdecision.RecordGraphSharedFeatureVersion
	var finite finiteExport
	a := &finite.Report.Assembly
	a.SourceSHA, a.ContractSHA, a.Calls, a.Mask = exported.SourceSHA, exported.ContractSHA, &zero, &mask
	a.Status, a.Total, a.Passed, a.FieldsTotal, a.FieldsPass = "COMPLETE_FINITE", 1, 1, 3, 3
	for i := range 8 {
		a.Attempts = append(a.Attempts, struct {
			Mask          uint16 `json:"mask"`
			Passed, Total int
		}{Mask: uint16(i), Total: 1})
	}
	a.Attempts[7].Passed = 1
	contextRaw, _ := json.Marshal(exported)
	finiteRaw, _ := json.Marshal(finite)
	if _, row, err := observe(source, contextRaw, finiteRaw, 0); err != nil || row.Selected != 7 || row.Attempts != 8 {
		t.Fatal("bound envelope rejected", row, err)
	}
	for _, mutate := range []func(*contextExport, *finiteExport){
		func(c *contextExport, _ *finiteExport) { c.Predictions = nil },
		func(c *contextExport, _ *finiteExport) { c.Tests = nil },
		func(c *contextExport, _ *finiteExport) { c.SourceSHA = "different" },
		func(c *contextExport, _ *finiteExport) { c.Context.SHA256 = "different" },
		func(c *contextExport, _ *finiteExport) { c.Context.Feature = jointdecision.RecordSharedFeatureVersion },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.ContractSHA = "different" },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.Calls = nil },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.Mask = nil },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.Total = 0 },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.FieldsPass-- },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.Attempts[0].Passed = 1 },
		func(_ *contextExport, f *finiteExport) { f.Report.Assembly.Attempts = f.Report.Assembly.Attempts[:7] },
	} {
		var c contextExport
		var f finiteExport
		json.Unmarshal(contextRaw, &c)
		json.Unmarshal(finiteRaw, &f)
		mutate(&c, &f)
		cr, _ := json.Marshal(c)
		fr, _ := json.Marshal(f)
		if _, _, err := observe(source, cr, fr, 0); err == nil {
			t.Fatal("unbound or nonunique observation accepted")
		}
	}
}
