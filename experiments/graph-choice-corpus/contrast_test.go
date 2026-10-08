package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestContrastsChangeIntentAndExpectationsOnly(t *testing.T) {
	for _, family := range families {
		raw, err := os.ReadFile(filepath.Join("../../examples/graph-choice-corpus", family.source))
		if err != nil {
			t.Fatal(err)
		}
		for _, language := range []string{"ko", "en", "mixed"} {
			for order := range 8 {
				baseline, err := variant(string(raw), family, order, language)
				if err != nil {
					t.Fatal(err)
				}
				for mask := range uint16(8) {
					got, err := contrastSource(string(raw), family, order, language, mask)
					if err != nil {
						t.Fatal(family.name, language, order, mask, err)
					}
					strip := func(s string) string {
						s = declaration.ReplaceAllString(s, "$1 $2 $3")
						return valueCase.ReplaceAllString(s, "$1")
					}
					if strip(got) != strip(baseline) {
						t.Fatal("requested behavior changed choices, source body or input rows")
					}
					for bit, line := range declaration.FindAllStringSubmatch(got, -1) {
						if mask&(1<<bit) == 0 && !strings.Contains(line[0], firstIntents[family.name][bit][map[bool]int{true: 1}[language == "en"]]) {
							t.Fatal("first-choice intent missing", family.name, mask, bit)
						}
					}
					for _, line := range valueCase.FindAllStringSubmatch(got, -1) {
						input, _ := strconv.Unquote(line[1])
						expected, _ := strconv.Unquote(line[2])
						want, err := oracle(family.name, []byte(input), mask)
						if err != nil || !sameJSON([]byte(expected), want) {
							t.Fatal("case does not follow requested policy")
						}
					}
				}
			}
		}
		wrong := valueCase.ReplaceAllString(string(raw), "    value_case $1 -> \"{}\"")
		if _, err := contrastSource(wrong, family, 0, "en", 7); err == nil {
			t.Fatal("source/oracle disagreement was accepted")
		}
	}
}
