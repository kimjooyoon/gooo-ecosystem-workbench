// Command assembly-graph-observe runs four explicit origin/next model routes.
// Raw native observations stay in --out; stdout is a path-free public projection.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

type observation struct {
	OriginModel     bool                   `json:"origin_model_requested"`
	NextModel       bool                   `json:"next_model_requested"`
	Origin          workbench.Summary      `json:"origin"`
	Added           []int                  `json:"original_caller_indices_added"`
	Rounds          []workbench.JointRound `json:"rounds"`
	FreshCalls      int                    `json:"fresh_construction_model_calls"`
	Passed          int64                  `json:"final_fields_passed"`
	Total           int64                  `json:"final_fields_total"`
	Other           int64                  `json:"final_inputs_outside_construction"`
	FinalCalls      int64                  `json:"final_new_model_calls"`
	OriginUnchanged bool                   `json:"origin_bytes_unchanged"`
	RawSHA          string                 `json:"raw_construction_observation_sha256"`
	RawBytes        int                    `json:"raw_construction_observation_bytes"`
}

func main() {
	compiler := flag.String("compiler", "gooo", "public 0.6.22 compiler")
	out := flag.String("out", "", "new directory for original observations")
	flag.Parse()
	if err := run(*compiler, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(compiler, out string) error {
	if out == "" {
		return fmt.Errorf("--out is required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		return fmt.Errorf("output exists or cannot be inspected")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	const model = "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json"
	ctx := context.Background()
	rows := []observation{}
	for _, originModel := range []string{"", model} {
		name := "fixed"
		if originModel != "" {
			name = "model"
		}
		origin := filepath.Join(out, name, "origin")
		r, err := workbench.Assemble(ctx, workbench.Options{Compiler: compiler, Model: originModel, Out: origin},
			workbench.AssemblyRequest{Source: "examples/assembly-graph-feedback/source.gooo", Entry: "Present", AssemblyActivity: "Describe", Cases: "examples/assembly-graph-feedback/adaptive-cases.json"})
		if err != nil {
			return err
		}
		for _, nextModel := range []string{"", model} {
			nextName := "next-fixed"
			if nextModel != "" {
				nextName = "next-model"
			}
			next := filepath.Join(out, name, nextName)
			c, err := workbench.ConstructFromAssembly(ctx, workbench.Options{Compiler: compiler, Model: nextModel, Out: next},
				workbench.AssemblyConstructionRequest{Assembly: origin, HoldoutCases: "examples/assembly-graph-feedback/holdout-cases.json", MaxProgramBudget: 8, MaxRounds: 5})
			if err != nil {
				return err
			}
			if c.Feedback == nil || !c.Feedback.Consumed || c.Loop == nil || c.Loop.FinalEvaluation == nil || c.Loop.FinalEvaluation.Joint == nil {
				return fmt.Errorf("expected consumed graph feedback and final observation")
			}
			f := c.Loop.FinalEvaluation
			row := observation{OriginModel: originModel != "", NextModel: nextModel != "", Origin: r.Observation, Added: c.Feedback.AddedIndices,
				Rounds: c.Loop.Rounds, Passed: f.Passed, Total: f.Total, Other: *f.Joint.Inputs.Other, FinalCalls: f.Joint.NewModelCalls, OriginUnchanged: true}
			for _, round := range c.Loop.Rounds {
				raw, err := os.ReadFile(filepath.Join(next, "construction", round.Result))
				if err != nil {
					return err
				}
				var doc struct {
					Construction struct {
						Initial struct {
							Steps []struct {
								Generation struct {
									Report struct {
										Assembly *struct {
											Calls int `json:"model_calls"`
										} `json:"record_assembly"`
									}
								}
							}
						}
					}
				}
				if err = json.Unmarshal(raw, &doc); err != nil {
					return err
				}
				for _, step := range doc.Construction.Initial.Steps {
					if step.Generation.Report.Assembly != nil {
						row.FreshCalls += step.Generation.Report.Assembly.Calls
					}
				}
			}
			var context struct{ Artifacts []struct{ Path string } }
			raw, err := os.ReadFile(filepath.Join(origin, "next-context.json"))
			if err != nil {
				return err
			}
			if err = json.Unmarshal(raw, &context); err != nil {
				return err
			}
			for _, a := range context.Artifacts {
				before, e1 := os.ReadFile(filepath.Join(origin, a.Path))
				after, e2 := os.ReadFile(filepath.Join(next, "origin", a.Path))
				if e1 != nil || e2 != nil || string(before) != string(after) {
					return fmt.Errorf("origin artifact changed: %s", a.Path)
				}
			}
			raw, err = os.ReadFile(filepath.Join(next, "assembly-construction.json"))
			if err != nil {
				return err
			}
			row.RawSHA, row.RawBytes = fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), len(raw)
			rows = append(rows, row)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(struct {
		Schema       string        `json:"schema"`
		Observations []observation `json:"observations"`
		Scope        string        `json:"scope"`
	}{"gooo/bound-assembly-feedback-observation/v1", rows,
		"two bound activities and a fixed pure helper; original root and caller oracles retained; four explicit origin/next routes; final two inputs used only after selection; training exposure unknown; finite fields and repeated fresh rounds, no general accuracy or timing claim"})
}
