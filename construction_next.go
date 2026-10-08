package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// ConstructionObservation transports recorded counts, not a continuation decision.
// BudgetKnown is false when neither a source nor a policy budget was recorded.
type ConstructionObservation struct {
	ActivityID   string `json:"activity_id"`
	Kind         string `json:"kind,omitempty"`
	Matched      int64  `json:"matched"`
	Total        int64  `json:"total"`
	Scored       int64  `json:"scored"`
	Rejected     int64  `json:"rejected"`
	Ranked       int64  `json:"ranked"`
	Budget       int64  `json:"budget"`
	Observed     bool   `json:"observed"`
	BudgetKnown  bool   `json:"budget_known"`
	BudgetSource string `json:"budget_source"`
	Consistent   bool   `json:"consistent"`
	Omitted      int64  `json:"omitted"`
	SpaceKnown   bool   `json:"space_known"`
}

type constructionControl struct {
	Decisions []constructionDecision `json:"decisions"`
	Entry     *constructionDecision  `json:"entry"`
}

type constructionDecision struct {
	Input struct {
		Scored *int64 `json:"scored"`
		Budget *int64 `json:"budget"`
	} `json:"input"`
}

type ConstructionNextStep struct {
	Observation ConstructionObservation `json:"observation"`
	Code        string                  `json:"code"`
	Action      string                  `json:"action"`
	Message     string                  `json:"message"`
}

func constructionObservations(r result) []ConstructionObservation {
	var observations []ConstructionObservation
	for _, steps := range [][]constructionStep{r.Composition.Preparations, r.Composition.Steps} {
		for _, step := range steps {
			if fill := step.Generation.Report.Fill; fill != nil {
				o := fillObservation(step.Generation.Report.ActivityID, fill)
				if step.Generation.Report.Assembly != nil || step.Generation.Report.Search != nil {
					o.Consistent = false
				}
				observations = append(observations, o)
				continue
			}
			if search := step.Generation.Report.Search; search != nil {
				o := searchObservation(step.Generation.Report.ActivityID, search)
				if step.Generation.Report.Assembly != nil {
					o.Consistent = false
				}
				observations = append(observations, o)
				continue
			}
			a := step.Generation.Report.Assembly
			if a == nil {
				continue
			}
			o := ConstructionObservation{ActivityID: step.Generation.Report.ActivityID,
				Scored: int64(len(a.Attempts)), Ranked: int64(len(a.Ranking)), Consistent: true,
				BudgetSource: "unavailable", Kind: "record_choices", SpaceKnown: true}
			if a.AttemptBudget != nil {
				o.Budget, o.BudgetKnown, o.BudgetSource = *a.AttemptBudget, true, "source_contract"
			}
			o.Observed = a.CasePassed != nil && a.CaseTotal != nil
			if o.Observed {
				o.Matched, o.Total = *a.CasePassed, *a.CaseTotal
			}
			seen := make(map[uint16]bool, len(a.Ranking))
			for _, mask := range a.Ranking {
				if seen[mask] {
					o.Consistent = false
				}
				seen[mask] = true
			}
			for i, attempt := range a.Attempts {
				if attempt.Mask == nil || i >= len(a.Ranking) || *attempt.Mask != a.Ranking[i] {
					o.Consistent = false
				}
			}
			if o.ActivityID == "" {
				o.Consistent = false
			}
			if a.Control != nil {
				observe := func(d constructionDecision) {
					if d.Input.Budget == nil || d.Input.Scored == nil {
						o.Consistent = false
						return
					}
					if o.BudgetKnown && o.Budget != *d.Input.Budget || *d.Input.Scored < 1 || *d.Input.Scored > o.Scored {
						o.Consistent = false
					}
					if !o.BudgetKnown {
						o.Budget, o.BudgetKnown, o.BudgetSource = *d.Input.Budget, true, "policy_observation"
					}
				}
				if a.Control.Entry != nil {
					observe(*a.Control.Entry)
				}
				for _, d := range a.Control.Decisions {
					observe(d)
				}
			}
			observations = append(observations, o)
		}
	}
	return observations
}

func constructionInput(s Snapshot, o ConstructionObservation) map[string]any {
	return map[string]any{"native_passed": s.Passed, "native_total": s.Total,
		"matched": o.Matched, "total": o.Total, "scored": o.Scored, "rejected": o.Rejected, "ranked": o.Ranked,
		"budget": o.Budget, "observed": o.Observed, "budget_known": o.BudgetKnown, "consistent": o.Consistent,
		"omitted": o.Omitted, "space_known": o.SpaceKnown}
}

// All action selection is executed from recipes/next-steps.gooo. No commands in
// an input receipt are executed, and the caller must explicitly run any resume.
func constructionNextSteps(ctx context.Context, o Options, root string, s Snapshot) ([]ConstructionNextStep, error) {
	rows := make([]any, len(s.Construction))
	for i, observation := range s.Construction {
		rows[i] = map[string]any{"inputs": map[string]any{"ObservationEcho": constructionInput(s, observation)}, "expected": map[string]any{"ObservationEcho": constructionInput(s, observation)}}
	}
	cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": rows})
	if err != nil {
		return nil, err
	}
	_, r, err := runRecipe(ctx, o, root, "next-steps", "construction-next", "", cases)
	if err != nil {
		return nil, err
	}
	if r.Runtime.Calls != 0 || r.Runtime.Passed != len(rows) || r.Runtime.Total != len(rows) || len(r.Runtime.Traces) != len(rows) {
		return nil, fmt.Errorf("construction next steps lost observations or called a model")
	}
	plans := make([]ConstructionNextStep, len(rows))
	seen := make([]bool, len(rows))
	for _, trace := range r.Runtime.Traces {
		i := trace.CaseIndex
		if i < 0 || i >= len(rows) || seen[i] || len(trace.Deliveries) != 2 {
			return nil, fmt.Errorf("construction next steps returned unexpected activity observations")
		}
		seen[i] = true
		found := false
		for _, delivery := range trace.Deliveries {
			if delivery.ID != "nextsteps://activity/next" {
				continue
			}
			if found {
				return nil, fmt.Errorf("duplicate construction next-step output")
			}
			found = true
			if err = json.Unmarshal(delivery.Actual, &plans[i]); err != nil {
				return nil, err
			}
		}
		if !found || plans[i].Code == "" || plans[i].Action == "" || plans[i].Message == "" {
			return nil, fmt.Errorf("construction next step did not return its typed result")
		}
		plans[i].Observation = s.Construction[i]
	}
	source, err := assets.ReadFile("recipes/next-steps.gooo")
	if err != nil {
		return nil, err
	}
	err = save(filepath.Join(root, "construction-next-steps.json"), map[string]any{
		"schema": "gooo/construction-next-steps/v1", "input_sha256": s.InputSHA,
		"source_sha256": fmt.Sprintf("%x", sha256.Sum256(source)), "compiler_source": r.Runtime.Source,
		"native_unit": s.Unit, "native_passed": s.Passed, "native_total": s.Total,
		"plans": plans, "new_model_calls": 0,
		"scope": "Gooo proposals from recorded construction counts and recounted native expectations; per-activity plans do not establish which helper caused a native mismatch; explicit compiler resume revalidates the saved source and records",
	})
	return plans, err
}
