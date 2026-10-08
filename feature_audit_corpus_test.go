package workbench

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestGraphChoiceCorpusKeepsFamiliesAndCompleteArrays(t *testing.T) {
	raw, err := os.ReadFile("examples/feature-audit/three-family-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	input, err := readFeatureAuditInput(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string]int{}
	for _, row := range input.Cases {
		id := strings.Split(row.ID, "-")
		if len(id) != 3 || id[0] != row.Family || id[1] != "ko" && id[1] != "en" && id[1] != "mixed" {
			t.Fatal("source family or wording identity lost", row.ID)
		}
		order, err := strconv.Atoi(id[2])
		if err != nil || order < 0 || order > 7 || len(row.AcceptedMasks) != 1 || row.AcceptedMasks[0] != uint16(7^order) {
			t.Fatal("separately observed finite label differs", row.ID)
		}
		groups[row.Family+"-"+id[1]]++
	}
	if len(groups) != 9 {
		t.Fatal("three families by three wording forms required", groups)
	}
	for _, count := range groups {
		if count != 8 {
			t.Fatal("all eight arrangements required", groups)
		}
	}
	report, err := auditFeatureRows(context.Background(), input, nil)
	if err != nil || report.Rows != 72 || report.Families != 3 || report.DistinctFeatures != 72 ||
		report.MaximumCompatible != 72 || report.ConflictingGroups != 0 || report.ModelObserved || report.ModelCalls != 0 {
		t.Fatal("cross-family representation audit differs", report, err)
	}
}
