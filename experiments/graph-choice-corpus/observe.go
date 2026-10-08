package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type contextExport struct {
	SourceSHA   string `json:"original_source_sha256"`
	ContractSHA string `json:"contract_sha256"`
	Predictions *int   `json:"model_predictions"`
	Tests       *int   `json:"candidate_tests"`
	Context     struct {
		Status, Text, SHA256 string
		Feature              string `json:"feature_version"`
	}
}

type finiteExport struct {
	Report struct {
		Assembly struct {
			SourceSHA   string  `json:"original_source_sha256"`
			ContractSHA string  `json:"contract_sha256"`
			Mask        *uint16 `json:"selected_mask"`
			Status      string
			Calls       *int `json:"model_calls"`
			Passed      int
			Total       int
			FieldsPass  int `json:"fields_passed"`
			FieldsTotal int `json:"fields_total"`
			Attempts    []struct {
				Mask   uint16 `json:"mask"`
				Passed int
				Total  int
			} `json:"attempts"`
		} `json:"record_assembly"`
	}
}

type rowReceipt struct {
	ID          string  `json:"id"`
	Family      string  `json:"family"`
	Language    string  `json:"intent_language"`
	Arrangement int     `json:"arrangement"`
	SourceSHA   string  `json:"source_sha256"`
	ContractSHA string  `json:"contract_sha256"`
	ContextSHA  string  `json:"context_sha256"`
	Selected    uint16  `json:"selected_mask"`
	Cases       int     `json:"selection_cases"`
	Fields      int     `json:"selection_fields"`
	Attempts    int     `json:"candidate_attempts"`
	Requested   *uint16 `json:"requested_behavior_mask,omitempty"`
}

func observe(source, contextRaw, finiteRaw []byte, order int) (jointdecision.RecordGraphInput, rowReceipt, error) {
	return observeMask(source, contextRaw, finiteRaw, uint16(7^order))
}

func observeMask(source, contextRaw, finiteRaw []byte, wanted uint16) (jointdecision.RecordGraphInput, rowReceipt, error) {
	var exported contextExport
	var finite finiteExport
	var row rowReceipt
	var graph jointdecision.RecordGraphInput
	if err := json.Unmarshal(contextRaw, &exported); err != nil {
		return graph, row, err
	}
	if err := json.Unmarshal(finiteRaw, &finite); err != nil {
		return graph, row, err
	}
	a := finite.Report.Assembly
	if wanted > 7 || exported.SourceSHA != digest(source) || exported.ContractSHA == "" ||
		exported.Predictions == nil || *exported.Predictions != 0 || exported.Tests == nil || *exported.Tests != 0 ||
		exported.Context.Status != "ENCODED" || exported.Context.Feature != jointdecision.RecordGraphSharedFeatureVersion ||
		exported.Context.SHA256 != digest([]byte(exported.Context.Text)) ||
		a.SourceSHA != exported.SourceSHA || a.ContractSHA != exported.ContractSHA ||
		a.Calls == nil || *a.Calls != 0 || a.Mask == nil || *a.Mask != wanted ||
		a.Status != "COMPLETE_FINITE" || a.Total < 1 || a.Passed != a.Total || a.FieldsTotal != a.Total*3 || a.FieldsPass != a.FieldsTotal {
		return graph, row, fmt.Errorf("source-only context and complete finite label must share exact identities")
	}
	graph, err := jointdecision.DecodeRecordGraphThree(exported.Context.Text)
	if err != nil {
		return graph, row, err
	}
	// Each requested behavior has an arrangement that puts the accepted mask
	// last. Its eight attempts establish a unique accepted finite label.
	if wanted == 7 {
		if len(a.Attempts) != 8 {
			return graph, row, fmt.Errorf("baseline must observe all eight combinations")
		}
		for i, attempt := range a.Attempts {
			if attempt.Mask != uint16(i) || attempt.Total != a.Total || (attempt.Passed == attempt.Total) != (i == 7) {
				return graph, row, fmt.Errorf("baseline finite labels are not unique")
			}
		}
	}
	row = rowReceipt{SourceSHA: exported.SourceSHA, ContractSHA: exported.ContractSHA,
		ContextSHA: exported.Context.SHA256, Selected: *a.Mask, Cases: a.Total, Fields: a.FieldsTotal, Attempts: len(a.Attempts)}
	return graph, row, nil
}

func digest(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }
