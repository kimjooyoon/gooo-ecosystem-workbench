// Import the compiler's source-only graph exports and separately observed
// finite selection labels. It never trains a model or mixes labels into graphs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func main() {
	root := flag.String("observation", "", "published source-graph-context-20261008 directory")
	out := flag.String("out", "", "new fixture filename")
	flag.Parse()
	if *root == "" || *out == "" {
		panic("observation and out required")
	}
	input := workbench.FeatureAuditInput{Schema: "gooo/record-feature-audit-input/v1",
		FeatureVersion: jointdecision.RecordGraphSharedFeatureVersion,
		LabelSource:    "https://github.com/kimjooyoon/meta-ontology-go/wiki/Source-Graph-Model-Input; compiler dea641f709b37bfd3f8655a242bebaa6cdc0d18b; separately observed deterministic selected masks"}
	for order := range 8 {
		var exported struct {
			Context     struct{ Status, Text string }
			SourceSHA   string `json:"original_source_sha256"`
			ContractSHA string `json:"contract_sha256"`
			Predictions int    `json:"model_predictions"`
			Tests       int    `json:"candidate_tests"`
		}
		read(filepath.Join(*root, "context", fmt.Sprintf("%s-%d.json", input.FeatureVersion, order)), &exported)
		if exported.Context.Status != "ENCODED" || exported.Predictions != 0 || exported.Tests != 0 {
			panic("expected source-only graph export")
		}
		graph, err := jointdecision.DecodeRecordGraphThree(exported.Context.Text)
		must(err)
		var generation struct {
			Report struct {
				Assembly struct {
					SourceSHA   string `json:"original_source_sha256"`
					ContractSHA string `json:"contract_sha256"`
					Mask        uint16 `json:"selected_mask"`
					Status      string `json:"status"`
					Calls       int    `json:"model_calls"`
				} `json:"record_assembly"`
			} `json:"report"`
		}
		read(filepath.Join(*root, "finite", fmt.Sprintf("order-%d.json", order)), &generation)
		a := generation.Report.Assembly
		if exported.SourceSHA == "" || exported.ContractSHA == "" ||
			a.SourceSHA != exported.SourceSHA || a.ContractSHA != exported.ContractSHA {
			panic("finite label and graph export have different source/contract digests")
		}
		if a.Mask != uint16(7^order) || a.Status != "COMPLETE_FINITE" || a.Calls != 0 {
			panic("unexpected finite label observation")
		}
		input.Cases = append(input.Cases, workbench.FeatureAuditCase{
			ID: fmt.Sprintf("filename-order-%d", order), Family: "filename-classifier", Graph: &graph,
			AcceptedMasks: []uint16{a.Mask}})
	}
	raw, err := json.MarshalIndent(input, "", "  ")
	must(err)
	f, err := os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	must(err)
	_, err = f.Write(append(raw, '\n'))
	must(err)
	must(f.Close())
}

func read(name string, value any) {
	raw, err := os.ReadFile(name)
	must(err)
	must(json.Unmarshal(raw, value))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
