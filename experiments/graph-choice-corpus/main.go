// Export source-owned choices and separately observed finite labels. Family
// groups remain explicit so later training can hold out entire programs.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func main() {
	compiler := flag.String("compiler", "", "Gooo compiler supporting graph input")
	expected := flag.String("expected-compiler", "", "required clean compiler source SHA")
	sources := flag.String("sources", "examples/graph-choice-corpus", "three Gooo source families")
	out := flag.String("out", "", "new observation directory")
	contrasts := flag.Bool("intent-contrasts", false, "request all eight behaviors while preserving source choices")
	native := flag.String("native-contrasts", "", "existing contrast corpus for separate native inputs and saved replay")
	nativeModel := flag.String("native-model", "", "one model for native contrast construction")
	nativeFolds := flag.String("native-model-folds", "", "root with family/qat_ternary/model.json held-out models")
	budgetProbes := flag.Bool("budget-probes", false, "also observe actual source-case completion at budgets 1, 2, 4 and 8")
	flag.Parse()
	if *compiler == "" || len(*expected) != 40 || *out == "" {
		panic("compiler, expected-compiler and new out directory required")
	}
	if *nativeModel != "" && *nativeFolds != "" || *native == "" && (*nativeModel != "" || *nativeFolds != "" || *budgetProbes) {
		panic("native mode and one explicit model source required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	build := command(ctx, *compiler, "version", "--build", "--json")
	var identity struct {
		Source string `json:"compiler_source_sha"`
		Status string `json:"source_status"`
	}
	must(json.Unmarshal(build, &identity))
	if identity.Source != *expected || identity.Status != "CLEAN_VCS" {
		panic("exact clean compiler source required")
	}
	must(os.Mkdir(*out, 0755))
	write(*out, "compiler.json", build)
	if *native != "" {
		must(nativeContrasts(ctx, *compiler, *expected, *native, *out, nativeOptions{*nativeModel, *nativeFolds, *budgetProbes}))
		must(archiveEvidence(*out))
		return
	}
	input := workbench.FeatureAuditInput{Schema: "gooo/record-feature-audit-input/v1",
		FeatureVersion: jointdecision.RecordGraphSharedFeatureVersion,
		LabelSource:    "Separate finite source-case observations; compiler " + *expected + "; see corpus-index.json and rows/"}
	requests := []int{7}
	if *contrasts {
		requests = []int{0, 1, 2, 3, 4, 5, 6, 7}
	}
	rows := make([]rowReceipt, 0, 72*len(requests))
	for _, family := range families {
		raw, err := os.ReadFile(filepath.Join(*sources, family.source))
		must(err)
		for _, requested := range requests {
			for _, language := range []string{"ko", "en", "mixed"} {
				for order := range 8 {
					id := fmt.Sprintf("%s-%s-%d", family.name, language, order)
					if *contrasts {
						id = fmt.Sprintf("%s-request%d", id, requested)
					}
					directory := filepath.Join(*out, "rows", id)
					must(os.MkdirAll(directory, 0755))
					text, err := variant(string(raw), family, order, language)
					if *contrasts {
						text, err = contrastSource(string(raw), family, order, language, uint16(requested))
					}
					must(err)
					write(directory, "source.gooo", []byte(text))
					source := filepath.Join(directory, "source.gooo")
					exported := command(ctx, *compiler, "body-context", "--value-flow", "--activity", family.activity,
						"--feature-version", input.FeatureVersion, source)
					write(directory, "context.json", exported)
					finite := command(ctx, *compiler, "body-codegen", "--json", "--activity", family.activity, source)
					write(directory, "finite.json", finite)
					graph, row, err := observeMask([]byte(text), exported, finite, uint16(requested^order))
					must(err)
					row.ID, row.Family, row.Language, row.Arrangement = id, family.name, language, order
					if *contrasts {
						mask := uint16(requested)
						row.Requested = &mask
					}
					rows = append(rows, row)
					input.Cases = append(input.Cases, workbench.FeatureAuditCase{ID: id, Family: family.name,
						Graph: &graph, AcceptedMasks: []uint16{row.Selected}})
				}
			}
		}
	}
	writeJSON(*out, "audit-input.json", input)
	index := map[string]any{
		"schema": "gooo/graph-choice-corpus/v1", "compiler_source_sha": *expected,
		"source_families": len(families), "arrangements_per_source": 8, "intent_forms": 3, "rows": rows,
		"model_calls": 0, "trained": false,
		"scope": "Three source families with repeated arrangements and wording; finite source-case labels; no held-out execution or learned accuracy claim.",
	}
	if *contrasts {
		index["schema"] = "gooo/graph-intent-contrast-corpus/v1"
		index["requested_behaviors_per_source"] = 8
		index["scope"] = "Three source families; eight source-declared output policies, arrangements and wording; independent Go oracles for finite labels; no learned accuracy or native holdout claim."
	}
	writeJSON(*out, "corpus-index.json", index)
	if *contrasts {
		// Copy each fixed-size choice array; retain the source graph and labels.
		for i := range input.Cases {
			graph := *input.Cases[i].Graph
			for bit := range graph.Choices {
				graph.Choices[bit].Intent = "intent withheld"
			}
			input.Cases[i].Graph = &graph
		}
		input.LabelSource += "; intent ablation preserves graph, ordering and finite labels"
		writeJSON(*out, "intent-ablated-input.json", input)
		must(archiveEvidence(*out))
	}
	fmt.Printf("%d rows, %d source families; zero model calls and no training\n", len(rows), len(families))
}

func command(ctx context.Context, compiler string, args ...string) []byte {
	raw, err := exec.CommandContext(ctx, compiler, args...).Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			panic(fmt.Sprintf("%v: %s", err, exit.Stderr))
		}
		panic(err)
	}
	return raw
}

func write(root, name string, raw []byte) { must(os.WriteFile(filepath.Join(root, name), raw, 0644)) }
func writeJSON(root, name string, value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	must(err)
	write(root, name, append(raw, '\n'))
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
