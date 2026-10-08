package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestVariantsPreserveCasesAndSwapOnlyDeclaredChoices(t *testing.T) {
	for _, family := range families {
		raw, err := os.ReadFile(filepath.Join("../../examples/graph-choice-corpus", family.source))
		if err != nil {
			t.Fatal(err)
		}
		original := string(raw)
		cases := original[strings.Index(original, "    value_case "):]
		for _, language := range []string{"ko", "en", "mixed"} {
			for order := range 8 {
				source, err := variant(original, family, order, language)
				if err != nil || !strings.HasSuffix(source, cases) {
					t.Fatal("variant changed finite cases or budget", family.name, language, order, err)
				}
				lines := declaration.FindAllStringSubmatch(source, -1)
				for bit, choice := range family.choices {
					first, second := choice.first, choice.second
					if order&(1<<bit) != 0 {
						first, second = second, first
					}
					if lines[bit][3] != strconv.Quote(second) || strings.Count(source, choice.field+": "+first) != 1 {
						t.Fatal("body and declaration were not swapped together", family.name, order, bit)
					}
				}
			}
		}
		for _, invalid := range []string{original + original, strings.Replace(original, `at "1"`, `at "2"`, 1)} {
			if _, err := variant(invalid, family, 0, "ko"); err == nil {
				t.Fatal("ambiguous fixture accepted")
			}
		}
		for _, order := range []int{-1, 8} {
			if _, err := variant(original, family, order, "ko"); err == nil {
				t.Fatal("invalid order accepted")
			}
		}
		if _, err := variant(original, family, 0, "unknown"); err == nil {
			t.Fatal("invalid language accepted")
		}
	}
}
