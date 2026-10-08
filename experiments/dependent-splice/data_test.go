package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	wb "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func TestFrozenSpliceRosterAndIndependentOracle(t *testing.T) {
	rows := roster()
	if len(rows) != 542 {
		t.Fatalf("roster changed: %d", len(rows))
	}
	source, err := os.ReadFile("../../recipes/splice.gooo")
	if err != nil {
		t.Fatal(err)
	}
	if err = assertDisjoint(source, rows); err != nil {
		t.Fatal(err)
	}
	seen := map[wb.SpliceRequest]bool{}
	for _, r := range rows {
		if seen[r] {
			t.Fatal("duplicate native tuple")
		}
		seen[r] = true
		if len(r.Source) > 1024 || len(r.Replacement) > 1024 {
			t.Fatal("invalid Text input")
		}
		got := oracle(r)
		if got.Bytes != int64(len(got.Source)) || got.Bytes > 1024 {
			t.Fatal("invalid oracle output")
		}
	}
	if !oracle(rows[len(rows)-2]).Changed || oracle(rows[len(rows)-2]).Bytes != 1024 || oracle(rows[len(rows)-1]).Changed {
		t.Fatal("output boundary changed")
	}
	paths, groups, err := batches(t.TempDir(), rows)
	if err != nil || len(paths) != len(groups) {
		t.Fatal(err)
	}
	total := 0
	for i, path := range paths {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil || len(raw) > 32768 || len(groups[i]) > 128 {
			t.Fatal("batch boundary", err)
		}
		total += len(groups[i])
	}
	if total != len(rows) {
		t.Fatal("batch dropped rows")
	}
}

func TestSpliceOracleDistinguishesNoopStaleAndUTF8(t *testing.T) {
	for _, tc := range []struct {
		r    wb.SpliceRequest
		want wb.SpliceEdit
	}{
		{wb.SpliceRequest{Source: "한🙂끝", Before: "한", Expected: "🙂", After: "끝", Replacement: "글"}, wb.SpliceEdit{Changed: true, Source: "한글끝", Bytes: 9}},
		{wb.SpliceRequest{Source: "x+x", Before: "x+", Expected: "x", Replacement: "y"}, wb.SpliceEdit{Changed: true, Source: "x+y", Bytes: 3}},
		{wb.SpliceRequest{Source: "xx", Before: "xx", Expected: "", After: "xx", Replacement: "y"}, wb.SpliceEdit{Source: "xx", Bytes: 2}},
		{wb.SpliceRequest{Source: "x", Expected: "y", Replacement: "z"}, wb.SpliceEdit{Source: "x", Bytes: 1}},
		{wb.SpliceRequest{Source: "x", Expected: "x", Replacement: "x"}, wb.SpliceEdit{Source: "x", Bytes: 1}},
		{wb.SpliceRequest{Source: "x", Expected: "x", Replacement: strings.Repeat("z", 1024)}, wb.SpliceEdit{Changed: true, Source: strings.Repeat("z", 1024), Bytes: 1024}},
	} {
		if got := oracle(tc.r); got != tc.want {
			t.Fatalf("got %+v want %+v", got, tc.want)
		}
	}
}
