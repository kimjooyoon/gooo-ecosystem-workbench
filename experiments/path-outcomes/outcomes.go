package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
	"github.com/kimjooyoon/gooo-decision-runtime/pathplan"
)

type exportedInput struct {
	ID   string `json:"decision_id"`
	SHA  string `json:"input_sha256"`
	Text string `json:"text"`
}

type sourceExport struct {
	Schema    string `json:"schema"`
	SourceSHA string `json:"original_source_sha256"`
	Binding   struct {
		Decision   string `json:"decision"`
		Equivalent bool   `json:"equivalent"`
	} `json:"source_binding"`
	Context struct {
		Status  string `json:"status"`
		Feature string `json:"feature_version"`
		PlanSHA string `json:"original_plan_sha256"`
	} `json:"context"`
	Plan        pathplan.Plan   `json:"expanded_plan"`
	Inputs      []exportedInput `json:"inputs"`
	Predictions int             `json:"model_predictions"`
	Tests       int             `json:"candidate_tests"`
	Emission    bool            `json:"selected_emission"`
	Writes      int             `json:"repository_writes"`
}

type outcome struct {
	Input    int64  `json:"input"`
	Expected int64  `json:"expected"`
	Actual   *int64 `json:"actual,omitempty"`
	Failure  string `json:"failure,omitempty"`
	Passed   bool   `json:"passed"`
}

type candidate struct {
	Mask       int               `json:"mask"`
	Choices    map[string]string `json:"choices"`
	Failure    string            `json:"compile_failure,omitempty"`
	Body       string            `json:"gooo_body,omitempty"`
	GoSHA      string            `json:"go_source_sha256,omitempty"`
	Outcomes   []outcome         `json:"outcomes"`
	Compatible bool              `json:"compatible"`
	GoSource   string            `json:"-"`
}

type marginal struct {
	ID     string         `json:"decision_id"`
	Counts map[string]int `json:"compatible_mask_counts"`
}

type observation struct {
	Schema            string          `json:"schema"`
	SourceSHA         string          `json:"source_sha256"`
	PlanSHA           string          `json:"plan_sha256"`
	CasesSHA          string          `json:"consumed_cases_sha256"`
	Inputs            []exportedInput `json:"model_inputs"`
	Declared          int             `json:"declared_combinations"`
	Candidates        []candidate     `json:"candidates"`
	Compatible        []int           `json:"compatible_masks"`
	Marginals         []marginal      `json:"site_marginals"`
	MarginalProduct   int             `json:"marginal_product_combinations"`
	InvalidProduct    int             `json:"incompatible_marginal_combinations"`
	FiniteGroups      [][]int         `json:"finite_output_groups"`
	NativeChecked     int             `json:"native_outcomes_checked"`
	NativeParity      bool            `json:"native_parity"`
	ModelCalls        int             `json:"model_calls"`
	TrainingPerformed bool            `json:"training_performed"`
	Scope             string          `json:"scope"`
}

func digest(raw []byte) string {
	value := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(value[:])
}

func bindExport(raw, source []byte) (sourceExport, *pathplan.PreparedPlan, error) {
	var e sourceExport
	if err := json.Unmarshal(raw, &e); err != nil {
		return e, nil, err
	}
	if (e.Schema != "gooo/compiler-path-input-export/v2" && e.Schema != "gooo/compiler-path-model-input-export/v1") ||
		e.SourceSHA != digest(source) || e.Binding.Decision != "PASS" || !e.Binding.Equivalent ||
		e.Context.Status != "ENCODED" || e.Context.Feature != decision.SemanticContextIntentFeatureVersion ||
		e.Predictions != 0 || e.Tests != 0 || e.Emission || e.Writes != 0 {
		return e, nil, fmt.Errorf("source-bound zero-execution semantic context required")
	}
	p, err := pathplan.Prepare(e.Plan)
	if err != nil {
		return e, nil, err
	}
	if p.PlanSHA256() != e.Context.PlanSHA || len(e.Inputs) != len(e.Plan.Decisions) {
		return e, nil, fmt.Errorf("expanded plan or input count differs from source receipt")
	}
	for i, choice := range e.Plan.Decisions {
		fields, err := p.SourceFeatures(choice.ID)
		if err != nil {
			return e, nil, err
		}
		intent := choice.Intent
		if j := strings.LastIndex(intent, "intent: "); j >= 0 {
			intent = intent[j+len("intent: "):]
		}
		text, err := decision.EncodeSemanticContextInput(fields, intent)
		input := e.Inputs[i]
		if err != nil || input.ID != choice.ID || input.Text != text || input.SHA != digest([]byte(text)) {
			return e, nil, fmt.Errorf("source-derived input differs at choice %s", choice.ID)
		}
	}
	return e, p, nil
}

func decodeCases(raw []byte) ([]pathplan.TestCase, error) {
	var cases []pathplan.TestCase
	if len(raw) > 128<<10 {
		return nil, fmt.Errorf("case byte bound exceeded")
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, err
	}
	if len(cases) == 0 || len(cases) > 128 {
		return nil, fmt.Errorf("1..128 explicit consumed cases required")
	}
	seen := map[int64]bool{}
	for _, c := range cases {
		if seen[c.Input] {
			return nil, fmt.Errorf("duplicate input cannot inflate target evidence")
		}
		seen[c.Input] = true
	}
	return cases, nil
}

func observe(ctx context.Context, e sourceExport, prepared *pathplan.PreparedPlan, rawCases []byte) (observation, error) {
	result := observation{Schema: "gooo/finite-path-targets/v1", SourceSHA: e.SourceSHA,
		PlanSHA: prepared.PlanSHA256(), CasesSHA: digest(rawCases), Inputs: e.Inputs,
		Compatible: []int{}, Scope: "Consumed finite activity cases; SDK typed candidates and native projections. Joint masks are finite targets, not whole-domain equivalence or caller correctness. No held-out score, training or inference."}
	if len(e.Plan.Decisions) > 6 {
		return result, fmt.Errorf("exhaustive audit requires at most 64 combinations; no partial targets emitted")
	}
	cases, err := decodeCases(rawCases)
	if err != nil {
		return result, err
	}
	result.Declared = 1 << len(e.Plan.Decisions)
	for mask := range result.Declared {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		row := candidate{Mask: mask, Choices: map[string]string{}, Outcomes: []outcome{}}
		for i, choice := range e.Plan.Decisions {
			row.Choices[choice.ID] = choice.Options[(mask>>i)&1].Label
		}
		program, err := prepared.Compile(row.Choices)
		if err != nil {
			row.Failure = err.Error()
		} else {
			row.Body, row.GoSource = program.GoooBody(), program.GoSource()
			row.GoSHA, row.Compatible = digest([]byte(row.GoSource)), true
			for _, test := range cases {
				v, err := program.Evaluate(test.Input)
				item := outcome{Input: test.Input, Expected: test.Expected}
				if err != nil {
					item.Failure = err.Error()
				} else if v.Type != decision.TypeInt {
					return result, fmt.Errorf("integer activity required")
				} else {
					item.Actual = &v.Int
					item.Passed = v.Int == test.Expected
				}
				row.Compatible = row.Compatible && item.Passed
				row.Outcomes = append(row.Outcomes, item)
			}
		}
		if row.Compatible {
			result.Compatible = append(result.Compatible, mask)
		}
		result.Candidates = append(result.Candidates, row)
	}
	result.summarize(e.Plan)
	return result, nil
}

func (r *observation) summarize(plan pathplan.Plan) {
	r.MarginalProduct = 1
	for _, choice := range plan.Decisions {
		m := marginal{ID: choice.ID, Counts: map[string]int{}}
		for _, option := range choice.Options {
			m.Counts[option.Label] = 0
		}
		for _, mask := range r.Compatible {
			m.Counts[r.Candidates[mask].Choices[choice.ID]]++
		}
		allowed := 0
		for _, n := range m.Counts {
			if n > 0 {
				allowed++
			}
		}
		r.MarginalProduct *= allowed
		r.Marginals = append(r.Marginals, m)
	}
	r.InvalidProduct = r.MarginalProduct - len(r.Compatible)
	groupIDs := map[string]int{}
	for _, c := range r.Candidates {
		if c.Failure != "" {
			continue
		}
		valid := true
		values := make([]int64, 0, len(c.Outcomes))
		for _, o := range c.Outcomes {
			if o.Actual == nil || o.Failure != "" {
				valid = false
				break
			}
			values = append(values, *o.Actual)
		}
		if !valid {
			continue
		}
		raw, _ := json.Marshal(values)
		index, exists := groupIDs[string(raw)]
		if !exists {
			index = len(r.FiniteGroups)
			groupIDs[string(raw)] = index
			r.FiniteGroups = append(r.FiniteGroups, []int{})
		}
		r.FiniteGroups[index] = append(r.FiniteGroups[index], c.Mask)
	}
}
