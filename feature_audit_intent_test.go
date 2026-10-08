package workbench

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func intentFixture(t *testing.T, name string) FeatureAuditInput {
	t.Helper()
	f, err := os.Open("examples/feature-audit/" + name + ".json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	raw, err := io.ReadAll(io.LimitReader(gz, 32<<20))
	if err != nil || len(raw) == 32<<20 {
		t.Fatal("invalid bounded corpus", err)
	}
	input, err := readFeatureAuditInput(raw)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestIntentContrastsPreserveBodiesAndChangeRequiredChoices(t *testing.T) {
	input := intentFixture(t, "three-family-intent-contrasts")
	ablated := intentFixture(t, "three-family-intent-ablated")
	if len(input.Cases) != 576 || len(ablated.Cases) != 576 {
		t.Fatal("576 repeated-design rows required")
	}
	bodies := map[string]string{}
	counts := map[string]int{}
	for i, row := range input.Cases {
		id := strings.Split(row.ID, "-")
		if len(id) != 4 || id[0] != row.Family || id[1] != "ko" && id[1] != "en" && id[1] != "mixed" || !strings.HasPrefix(id[3], "request") {
			t.Fatal("lost family, language or request", row.ID)
		}
		order, e1 := strconv.Atoi(id[2])
		requested, e2 := strconv.Atoi(strings.TrimPrefix(id[3], "request"))
		if e1 != nil || e2 != nil || order < 0 || order > 7 || requested < 0 || requested > 7 ||
			len(row.AcceptedMasks) != 1 || row.AcceptedMasks[0] != uint16(order^requested) {
			t.Fatal("intent contrast label differs", row.ID)
		}
		graph := *row.Graph
		for bit := range graph.Choices {
			graph.Choices[bit].Intent = "intent withheld"
		}
		graphJSON, err := json.Marshal(graph)
		if err != nil {
			t.Fatal(err)
		}
		key := row.Family + "-" + id[2]
		if previous, exists := bodies[key]; exists && previous != string(graphJSON) {
			t.Fatal("changing the requested behavior changed the source graph", row.ID)
		}
		bodies[key] = string(graphJSON)
		counts[key]++
		row.Graph = &graph
		want, _ := json.Marshal(row)
		got, _ := json.Marshal(ablated.Cases[i])
		if string(want) != string(got) {
			t.Fatal("ablation changed labels or source structure", row.ID)
		}
	}
	if len(bodies) != 24 {
		t.Fatal("three families by eight arrangements required")
	}
	for _, count := range counts {
		if count != 24 {
			t.Fatal("three languages by eight requests required")
		}
	}
	for _, test := range []struct {
		input                             FeatureAuditInput
		distinct, conflicting, compatible int
	}{
		{input, 576, 0, 576}, {ablated, 24, 24, 72},
	} {
		report, err := auditFeatureRows(context.Background(), test.input, nil)
		if err != nil || report.Rows != 576 || report.Families != 3 || report.DistinctFeatures != test.distinct ||
			report.ConflictingGroups != test.conflicting || report.MaximumCompatible != test.compatible || report.ModelObserved || report.ModelCalls != 0 {
			t.Fatal("intent ablation measurement differs", report, err)
		}
		if test.conflicting > 0 {
			for _, group := range report.Groups {
				for _, votes := range group.Votes {
					if votes != 3 {
						t.Fatal("intent labels are not balanced within equal features")
					}
				}
			}
		}
	}
}
