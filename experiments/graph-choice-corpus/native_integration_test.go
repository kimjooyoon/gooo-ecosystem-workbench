package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrainedGraphModelsConstructAndReplayNativePrograms(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("GOOO_COMPILER required for actual native construction")
	}
	compiler, err := filepath.Abs(compiler)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var identity struct {
		Source string `json:"compiler_source_sha"`
	}
	if err := json.Unmarshal(command(ctx, compiler, "version", "--build", "--json"), &identity); err != nil || identity.Source == "" {
		t.Fatal("compiler identity required", err)
	}
	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("../../examples/graph-choice-corpus", family.source))
			if err != nil {
				t.Fatal(err)
			}
			source, err := contrastSource(string(raw), family, 0, "mixed", 7)
			if err != nil {
				t.Fatal(err)
			}
			cases, err := nativeCases([]byte(source), family, 7)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			write(root, "source.gooo", []byte(source))
			writeJSON(root, "cases.json", map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases})
			model, err := filepath.Abs(filepath.Join("../../models/graph-chooser-20261008", family.name, "qat_ternary/model.json"))
			if err != nil {
				t.Fatal(err)
			}
			construction := filepath.Join(root, "construction")
			result := command(ctx, compiler, "body-compose", "--entry", family.activity, "--source", filepath.Join(root, "source.gooo"), "--cases", filepath.Join(root, "cases.json"), "--model", model, "--out", construction)
			if err := checkNativeModel(result, []byte(source), family, 7, identity.Source, true, 1); err != nil {
				t.Fatal(err)
			}
			replay := command(ctx, compiler, "body-compose", "--source", filepath.Join(construction, "original.gooo"), "--composition", filepath.Join(construction, "composition.json"), "--cases", filepath.Join(root, "cases.json"))
			if err := checkNativeModel(replay, []byte(source), family, 7, identity.Source, false, 1); err != nil {
				t.Fatal(err)
			}
			if _, err := budgetObservations(ctx, compiler, []byte(source), family, 7, model, root); err != nil {
				t.Fatal(err)
			}
		})
	}
}
