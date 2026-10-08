package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func TestPublishedTrainedArtifactsReproduceRecordedScores(t *testing.T) {
	rows, _, err := loadRows("../../examples/feature-audit/three-family-intent-contrasts.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("../../publication/graph-chooser-20261008/training-evidence.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	archive := tar.NewReader(gz)
	checked := 0
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimPrefix(header.Name, "./")
		parts := strings.Split(name, "/")
		if len(parts) != 3 || parts[2] != "assessment.json" {
			continue
		}
		if header.Size > 2<<20 {
			t.Fatal("unexpected assessment size")
		}
		var recorded struct {
			Train      score
			Evaluation score
			Ablated    score  `json:"intent_ablated"`
			Weights    string `json:"model_weights_sha256"`
			Metadata   string `json:"model_metadata_sha256"`
		}
		if err := json.NewDecoder(archive).Decode(&recorded); err != nil {
			t.Fatal(err)
		}
		model, err := jointdecision.LoadRecordGraphSharedThree(filepath.Join("../../models/graph-chooser-20261008", parts[0], parts[1], "model.json"))
		if err != nil || model.WeightsSHA256() != recorded.Weights || model.MetadataSHA256() != recorded.Metadata {
			t.Fatal("published model identity differs", name, err)
		}
		training, evaluation := split(rows, parts[0])
		ablation := evaluation
		if len(ablation) == 0 {
			ablation = training
		}
		for _, test := range []struct {
			indices  []int
			mode     string
			expected score
		}{
			{training, "full", recorded.Train}, {evaluation, "full", recorded.Evaluation}, {ablation, "ablated", recorded.Ablated},
		} {
			got, err := evaluate(rows, test.indices, model, test.mode)
			if err != nil || !reflect.DeepEqual(got, test.expected) {
				t.Fatal("published predictions differ", name, test.mode, err)
			}
		}
		checked++
	}
	if checked != 12 {
		t.Fatal("twelve trained artifacts required", checked)
	}
}
