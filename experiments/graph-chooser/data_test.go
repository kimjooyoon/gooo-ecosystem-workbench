package main

import (
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func TestWholeFamilySplitsAndIntentPairDenominators(t *testing.T) {
	rows, sha, err := loadRows("../../examples/feature-audit/three-family-intent-contrasts.json.gz")
	if err != nil || len(sha) != 64 {
		t.Fatal(err)
	}
	for _, family := range []string{"filenames", "division", "retry"} {
		train, test := split(rows, family)
		if len(train) != 384 || len(test) != 192 {
			t.Fatal("whole-family split differs")
		}
		for _, i := range train {
			if rows[i].Family == family {
				t.Fatal("held-out family leaked into training")
			}
		}
		for _, i := range test {
			if rows[i].Family != family {
				t.Fatal("mixed evaluation family")
			}
		}
	}
	path, err := exportModel(filepath.Join(t.TempDir(), "constant"), network{}, "fp32", .5)
	if err != nil {
		t.Fatal(err)
	}
	m, err := jointdecision.LoadRecordGraphSharedThree(path)
	if err != nil {
		t.Fatal(err)
	}
	all, heldout := split(rows, "all-data-demonstration")
	if len(heldout) != 0 {
		t.Fatal("in-sample demonstration mislabeled")
	}
	got, err := evaluate(rows, all, m, "full")
	if err != nil || got.Rows != 576 || got.Correct != 72 || got.Fields != 864 || got.FieldTotal != 1728 || got.Pairs != 864 || got.ExactFlips != 0 || got.BothCorrect != 0 {
		t.Fatal("balanced labels and intent-pair counts differ", got, err)
	}
}
