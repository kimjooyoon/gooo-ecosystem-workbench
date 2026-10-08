package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

var nativeInputs = map[string][]string{
	"filenames": {`["x.gooo"]`, `[".x.gooo"]`, `["文🙂.gooo.gooo"]`, `["module.GOOO"]`, `["x.gooo "]`},
	"division":  {`[9007199254740993,7]`, `[-9223372036854775808,-1]`, `[9223372036854775807,-3]`, `[0,5]`, `[11,0]`},
	"retry": {`[false,true,0,4,0,10]`, `[false,true,0,4,9223372036854775807,9223372036854775807]`,
		`[false,true,4,4,2,10]`, `[true,false,1,4,2,10]`, `[false,true,0,4,-1,10]`, `[false,false,0,4,2,10]`},
}

func nativeContrasts(ctx context.Context, compiler, expected, corpus, out string) error {
	var index struct {
		Compiler string `json:"compiler_source_sha"`
		Rows     []rowReceipt
	}
	if err := readJSONFile(filepath.Join(corpus, "corpus-index.json"), &index); err != nil {
		return err
	}
	if index.Compiler != expected || len(index.Rows) != 576 {
		return fmt.Errorf("matching contrast corpus required")
	}
	byID := map[string]rowReceipt{}
	for _, row := range index.Rows {
		byID[row.ID] = row
	}
	var observations []map[string]any
	for _, family := range families {
		for requested := range uint16(8) {
			id := fmt.Sprintf("%s-mixed-0-request%d", family.name, requested)
			row := byID[id]
			sourcePath := filepath.Join(corpus, "rows", id, "source.gooo")
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				return err
			}
			if row.Requested == nil || *row.Requested != requested || row.Selected != requested || row.SourceSHA != digest(source) {
				return fmt.Errorf("native input source differs from recorded construction")
			}
			cases, err := nativeCases(source, family, requested)
			if err != nil {
				return err
			}
			dir := filepath.Join(out, id)
			if err := os.Mkdir(dir, 0755); err != nil {
				return err
			}
			writeJSON(dir, "cases.json", map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases})
			construction := filepath.Join(dir, "construction")
			raw := command(ctx, compiler, "body-compose", "--entry", family.activity, "--source", sourcePath, "--cases", filepath.Join(dir, "cases.json"), "--out", construction)
			write(dir, "result.json", raw)
			if err := checkNative(raw, source, family, requested, expected, true); err != nil {
				return fmt.Errorf("%s construction: %w", id, err)
			}
			replay := command(ctx, compiler, "body-compose", "--source", filepath.Join(construction, "original.gooo"), "--composition", filepath.Join(construction, "composition.json"), "--cases", filepath.Join(dir, "cases.json"))
			write(dir, "replay.json", replay)
			if err := checkNative(replay, source, family, requested, expected, false); err != nil {
				return fmt.Errorf("%s replay: %w", id, err)
			}
			observations = append(observations, map[string]any{"id": id, "source_sha256": row.SourceSHA, "requested_behavior_mask": requested, "native_cases": len(cases), "native_fields": len(cases) * 3, "construction_sha256": digest(raw), "replay_sha256": digest(replay)})
		}
	}
	writeJSON(out, "native-summary.json", map[string]any{"schema": "gooo/intent-contrast-native/v1", "compiler_source_sha": expected, "observations": observations, "scope": "Three families, eight requested policies, mixed wording and one fixed arrangement. Actual native outputs independently recounted against disjoint finite inputs and a Go oracle, then saved replay. No learned-model evaluation."})
	fmt.Println("24 requested-policy constructions and saved replays independently checked")
	return nil
}

func nativeCases(source []byte, family familySpec, requested uint16) ([]map[string]any, error) {
	var cases []map[string]any
	for _, input := range nativeInputs[family.name] {
		for _, line := range valueCase.FindAllStringSubmatch(string(source), -1) {
			selection, _ := strconv.Unquote(line[1])
			if sameJSON([]byte(selection), []byte(input)) {
				return nil, fmt.Errorf("native input overlaps source selection")
			}
		}
		var values []json.RawMessage
		if err := json.Unmarshal([]byte(input), &values); err != nil {
			return nil, err
		}
		inputs := map[string]json.RawMessage{}
		for i, value := range values {
			key := family.activity
			if len(values) > 1 {
				key += fmt.Sprintf(".input%d", i)
			}
			inputs[key] = value
		}
		want, err := oracle(family.name, []byte(input), requested)
		if err != nil {
			return nil, err
		}
		cases = append(cases, map[string]any{"inputs": inputs, "expected": map[string]json.RawMessage{family.activity: want}})
	}
	return cases, nil
}

func readJSONFile(path string, value any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}
