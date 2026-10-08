package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func budgetObservations(ctx context.Context, compiler string, source []byte, family familySpec, requested uint16, model, root string) ([]map[string]any, error) {
	if strings.Count(string(source), `attempts "8"`) != 1 {
		return nil, fmt.Errorf("one original eight-attempt budget required")
	}
	var observations []map[string]any
	for _, budget := range []int{1, 2, 4, 8} {
		dir := filepath.Join(root, fmt.Sprintf("budget-%d", budget))
		if err := os.Mkdir(dir, 0755); err != nil {
			return nil, err
		}
		revised := []byte(strings.Replace(string(source), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1))
		write(dir, "source.gooo", revised)
		args := []string{"body-codegen", "--json", "--activity", family.activity}
		calls := 0
		if model != "" {
			args = append(args, "--path-model", model)
			calls = 1
		}
		args = append(args, filepath.Join(dir, "source.gooo"))
		raw := command(ctx, compiler, args...)
		write(dir, "generation.json", raw)
		observation, err := checkBudget(raw, revised, family, requested, budget, calls)
		if err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func checkBudget(raw, source []byte, family familySpec, requested uint16, budget, calls int) (map[string]any, error) {
	var result finiteExport
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	a := result.Report.Assembly
	declared := valueCase.FindAllStringSubmatch(string(source), -1)
	if a.SourceSHA != digest(source) || a.Mask == nil || *a.Mask > 7 || a.Calls == nil || *a.Calls != calls ||
		a.Budget != budget || len(a.Attempts) == 0 || len(a.Attempts) > budget || a.Total <= 0 || len(a.Cases) != a.Total ||
		len(declared) != a.Total || a.FieldsTotal != a.Total*3 || len(a.Ranking) != 8 {
		return nil, fmt.Errorf("bounded source-case observation required")
	}
	if calls == 1 && (a.Prediction == nil || a.Prediction.Mask > 7 || a.Context == nil ||
		a.Context.Feature != jointdecision.RecordGraphSharedFeatureVersion || a.Ranking[0] != a.Prediction.Mask) {
		return nil, fmt.Errorf("source graph model ranking required")
	}
	seen, selected := uint16(0), false
	for i, mask := range a.Ranking {
		if mask > 7 || seen&(1<<mask) != 0 {
			return nil, fmt.Errorf("complete distinct candidate ranking required")
		}
		seen |= 1 << mask
		if i < len(a.Attempts) {
			attempt := a.Attempts[i]
			if attempt.Mask != mask || attempt.Total != a.Total {
				return nil, fmt.Errorf("attempt history differs from ranked candidates")
			}
			selected = selected || attempt.Mask == *a.Mask
		}
	}
	if !selected {
		return nil, fmt.Errorf("selected candidate was not attempted")
	}
	passed, fields := 0, 0
	for i, c := range a.Cases {
		input, inputErr := strconv.Unquote(declared[i][1])
		expected, expectedErr := strconv.Unquote(declared[i][2])
		if inputErr != nil || expectedErr != nil || !sameJSON([]byte(input), c.Inputs) || !sameJSON([]byte(expected), c.Expected) {
			return nil, fmt.Errorf("observed case differs from source roster")
		}
		want, err := oracle(family.name, c.Inputs, requested)
		if err != nil || !sameJSON(want, c.Expected) {
			return nil, fmt.Errorf("source expectation differs from requested behavior")
		}
		actual, err := oracle(family.name, c.Inputs, *a.Mask)
		if err != nil || !sameJSON(actual, c.Actual) {
			return nil, fmt.Errorf("observed candidate values differ from independent oracle")
		}
		matches := sameJSON(c.Expected, c.Actual)
		if matches != c.Passed {
			return nil, fmt.Errorf("source-case match flag differs")
		}
		if matches {
			passed++
		}
		var left, right map[string]json.RawMessage
		if err = json.Unmarshal(c.Actual, &left); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(c.Expected, &right); err != nil {
			return nil, err
		}
		if len(left) != 3 || len(right) != 3 {
			return nil, fmt.Errorf("three complete record fields required")
		}
		for name, value := range right {
			if sameJSON(left[name], value) {
				fields++
			}
		}
	}
	if passed != a.Passed || fields != a.FieldsPass || budget == 8 && passed != a.Total {
		return nil, fmt.Errorf("actual finite completeness recount differs")
	}
	status := "PARTIAL_FINITE"
	if passed == a.Total {
		status = "COMPLETE_FINITE"
	}
	if a.Status != status {
		return nil, fmt.Errorf("finite status differs from actual case matches")
	}
	return map[string]any{"budget": budget, "attempts": len(a.Attempts), "selected_mask": *a.Mask, "requested_mask": requested,
		"cases_matched": passed, "cases_total": a.Total, "fields_matched": fields, "fields_total": a.FieldsTotal, "model_calls": calls, "source_sha256": a.SourceSHA, "generation_sha256": digest(raw)}, nil
}
