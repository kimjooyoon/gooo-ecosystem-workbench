package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	wb "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

// oracle checks the intervening fragment by its positions in the original.
// The Gooo implementation instead compares a complete concatenated context.
func oracle(r wb.SpliceRequest) wb.SpliceEdit {
	out := wb.SpliceEdit{Source: r.Source, Bytes: int64(len(r.Source))}
	if !strings.HasPrefix(r.Source, r.Before) || !strings.HasSuffix(r.Source, r.After) {
		return out
	}
	end := len(r.Source) - len(r.After)
	if end < len(r.Before) || r.Source[len(r.Before):end] != r.Expected || r.Expected == r.Replacement {
		return out
	}
	text := r.Source[:len(r.Before)] + r.Replacement + r.Source[end:]
	if len(text) <= 1024 {
		out = wb.SpliceEdit{Changed: true, Source: text, Bytes: int64(len(text))}
	}
	return out
}

func roster() []wb.SpliceRequest {
	var rows []wb.SpliceRequest
	seen := map[wb.SpliceRequest]bool{}
	add := func(r wb.SpliceRequest) {
		if !seen[r] {
			seen[r] = true
			rows = append(rows, r)
		}
	}
	for _, before := range []string{"", "pre:", "한글:", "line\n", `"quoted":`} {
		for _, expected := range []string{"x", "값", "🙂", ""} {
			for _, after := range []string{"", ":tail", ":끝", "\n"} {
				for _, replacement := range []string{"", "replacement", "교체"} {
					r := wb.SpliceRequest{Source: before + expected + after, Before: before, Expected: expected, After: after, Replacement: replacement}
					add(r)
					r.Source = "!" + r.Source
					add(r)
				}
				add(wb.SpliceRequest{Source: before + expected + after, Before: before, Expected: expected, After: after, Replacement: expected})
			}
		}
	}
	for _, n := range []int{1023, 1024} {
		add(wb.SpliceRequest{Source: "px", Before: "p", Expected: "x", Replacement: strings.Repeat("z", n)})
	}
	return rows
}

func assertDisjoint(source []byte, rows []wb.SpliceRequest) error {
	selection := map[wb.SpliceRequest]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(source)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "value_case ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "value_case "), " -> ", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid source selection case")
		}
		input, err := strconv.Unquote(parts[0])
		var values []string
		if err != nil || json.Unmarshal([]byte(input), &values) != nil || len(values) != 5 {
			return fmt.Errorf("invalid selection input tuple")
		}
		selection[wb.SpliceRequest{Source: values[0], Before: values[1], Expected: values[2], After: values[3], Replacement: values[4]}] = true
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(selection) != 8 {
		return fmt.Errorf("source selection roster changed: %d", len(selection))
	}
	for _, row := range rows {
		if selection[row] {
			return fmt.Errorf("native tuple overlaps source selection")
		}
	}
	return nil
}

func suite(rows []wb.SpliceRequest) []byte {
	cases := make([]any, 0, len(rows))
	for _, r := range rows {
		inputs := map[string]any{"Splice.input0": r.Source, "Splice.input1": r.Before, "Splice.input2": r.Expected, "Splice.input3": r.After, "Splice.input4": r.Replacement}
		cases = append(cases, map[string]any{"inputs": inputs, "expected": map[string]any{"Splice": oracle(r)}})
	}
	raw, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases})
	if err != nil {
		panic(err) // Only concrete string, bool and integer values are encoded.
	}
	return raw
}
