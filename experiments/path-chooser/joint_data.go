package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
	"github.com/kimjooyoon/gooo-decision-runtime/pathplan"
)

const jointCorpusRoot = "publication/compatible-path-targets-20261010/original/"
const jointTargetsSHA = "c5969c1328012a6e113adec164d23fa1b97b11217d0781383a73523639cb1586"
const jointContextSHA = "20a1cef0140e4af264b9641961b77fb840f4a792f577399d531e9cd846319d0b"
const originalFP32Root = "publication/per-choice-decision-20261010/fp32/"
const originalFP32SHA = "8d5f63c6e27abd703eb88b13b7487c9ff57d48f6ebe6b68ca119182309482f06"

// This first study consumes one named, immutable, previously native-checked
// corpus. A general trainer must independently validate newly supplied corpora.
func loadJointCorpus(root string) (jointExample, error) {
	var result jointExample
	raw, err := readPinned(root+"targets.json.gz", jointTargetsSHA)
	if err != nil {
		return result, err
	}
	var target struct {
		Schema  string   `json:"schema"`
		PlanSHA string   `json:"plan_sha256"`
		Masks   []uint16 `json:"compatible_masks"`
		Inputs  []struct {
			ID   string `json:"decision_id"`
			Text string `json:"text"`
		} `json:"model_inputs"`
		NativeParity bool `json:"native_parity"`
	}
	if err := json.Unmarshal(raw, &target); err != nil {
		return result, err
	}
	raw, err = readPinned(root+"context.json.gz", jointContextSHA)
	if err != nil {
		return result, err
	}
	var exported struct {
		Plan pathplan.Plan `json:"expanded_plan"`
	}
	if err := json.Unmarshal(raw, &exported); err != nil {
		return result, err
	}
	p, err := pathplan.Prepare(exported.Plan)
	if err != nil {
		return result, err
	}
	if target.Schema != "gooo/finite-path-targets/v1" || !target.NativeParity || p.PlanSHA256() != target.PlanSHA || len(target.Inputs) != len(exported.Plan.Decisions) {
		return result, fmt.Errorf("pinned source plan and native target corpus differ")
	}
	names := decision.PathLabels()
	for i, choice := range exported.Plan.Decisions {
		input := target.Inputs[i]
		if input.ID != choice.ID {
			return result, fmt.Errorf("source site ordering differs")
		}
		e := example{ID: choice.ID, Text: input.Text}
		if err := decision.SemanticContextFeaturesInto(e.Text, &e.X); err != nil {
			return result, err
		}
		for at, v := range e.X {
			if v != 0 {
				e.Active = append(e.Active, at)
			}
		}
		pair := [2]int{-1, -1}
		for option, v := range choice.Options {
			for label, name := range names {
				if v.Label == name {
					pair[option] = label
				}
			}
		}
		result.Sites = append(result.Sites, e)
		result.Allowed = append(result.Allowed, pair)
	}
	result.Masks = target.Masks
	return result, result.validate()
}

func loadOriginalFP32(root string) (network, error) {
	var n network
	// Pin metadata as well as its weight file so byte interpretation is fixed.
	if _, err := readPinned(root+"model.json", "48c12b4df499e2a2081958fbf1145c3de5b5989f9fc75517fa88b71045d28128"); err != nil {
		return n, err
	}
	if _, err := decision.LoadPath(root + "model.json"); err != nil {
		return n, err
	}
	raw, err := readPinned(root+"weights.bin", originalFP32SHA)
	if err != nil {
		return n, err
	}
	if len(raw) != 4*parameterCount {
		return n, fmt.Errorf("original FP32 size differs")
	}
	for i := range n {
		n[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[4*i:]))
		if math.IsNaN(float64(n[i])) || math.IsInf(float64(n[i]), 0) {
			return n, fmt.Errorf("nonfinite original weight")
		}
	}
	return n, nil
}

func readPinned(path, sha string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) > 1<<20 || digest(raw) != sha {
		return nil, fmt.Errorf("frozen corpus or checkpoint differs: %s", path)
	}
	if !strings.HasSuffix(path, ".gz") {
		return raw, nil
	}
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	decoded, err := io.ReadAll(io.LimitReader(z, (1<<20)+1))
	if err != nil || len(decoded) > 1<<20 {
		return nil, fmt.Errorf("bounded corpus decompression failed: %v", err)
	}
	return decoded, nil
}
