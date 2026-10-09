package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
	"github.com/kimjooyoon/gooo-decision-runtime/pathplan"
)

func corpus(root string) ([]example, []example, error) {
	raw, err := os.ReadFile(root + "/source-context.json")
	if err != nil {
		return nil, nil, err
	}
	var context struct {
		Schema        string
		OriginalSHA   string                    `json:"original_source_sha256"`
		SourceBinding struct{ Decision string } `json:"source_binding"`
		Context       struct {
			Feature string `json:"feature_version"`
		}
		Inputs []struct {
			ID   string `json:"decision_id"`
			Text string
		}
	}
	if err := json.Unmarshal(raw, &context); err != nil {
		return nil, nil, err
	}
	source, err := os.ReadFile(root + "/source.gooo.fixture")
	if err != nil {
		return nil, nil, err
	}
	if context.Schema != "gooo/compiler-path-input-export/v2" || context.SourceBinding.Decision != "PASS" ||
		context.OriginalSHA != "sha256:"+digest(source) || context.Context.Feature != decision.SemanticContextIntentFeatureVersion || len(context.Inputs) != 6 {
		return nil, nil, fmt.Errorf("source-bound six-choice semantic export required")
	}
	raw, err = os.ReadFile(root + "/source-plan.json")
	if err != nil {
		return nil, nil, err
	}
	var document pathplan.Document
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, nil, err
	}
	if _, err := document.Prepare(); err != nil {
		return nil, nil, err
	}
	choices := document.Plan.Decisions
	labelNames := decision.PathLabels()
	var training, evaluation []example
	for _, input := range context.Inputs {
		var choice *pathplan.Choice
		for i := range choices {
			if choices[i].ID == input.ID {
				choice = &choices[i]
				break
			}
		}
		if choice == nil {
			return nil, nil, fmt.Errorf("exported decision absent from original plan")
		}
		separator := strings.LastIndex(input.Text, ";intent: ")
		if separator < 0 {
			return nil, nil, fmt.Errorf("complete semantic header required")
		}
		header := input.Text[:separator+len(";intent: ")]
		for direction := range 2 {
			label := -1
			for i, name := range labelNames {
				if name == choice.Options[direction].Label {
					label = i
				}
			}
			if label < 0 {
				return nil, nil, fmt.Errorf("undeclared path label")
			}
			trainIntents, testIntents := intents(choice.Kind, direction)
			for _, split := range []struct {
				name    string
				phrases []string
			}{{"train", trainIntents}, {"test", testIntents}} {
				for i, phrase := range split.phrases {
					e := example{Label: label, Text: header + phrase, ID: fmt.Sprintf("%s/%s/%d/%d", input.ID, split.name, direction, i)}
					if err := decision.SemanticContextFeaturesInto(e.Text, &e.X); err != nil {
						return nil, nil, err
					}
					for i, v := range e.X {
						if v != 0 {
							e.Active = append(e.Active, i)
						}
					}
					if split.name == "train" {
						training = append(training, e)
					} else {
						evaluation = append(evaluation, e)
					}
				}
			}
		}
	}
	return training, evaluation, nil
}

func intents(kind string, direction int) ([]string, []string) {
	first, second := "first", "second"
	koFirst, koSecond := "첫 번째", "두 번째"
	if direction == 1 {
		first, second = second, first
		koFirst, koSecond = koSecond, koFirst
	}
	noun := "local reference"
	switch kind {
	case pathplan.AssignmentTarget:
		noun = "assignment destination"
	case pathplan.OperandOrder:
		noun = "operand ordering"
	case pathplan.BranchLayout:
		noun = "branch layout"
	case pathplan.RootOrder:
		noun = "declaration ordering"
	}
	return []string{
		"Select the " + first + " option for " + noun + ".", "Use " + first + " instead of " + second + " in " + noun + ".",
		koFirst + " 선택지를 사용한다. " + noun, koSecond + " 대신 " + koFirst + " 후보를 고른다. " + noun,
	}, []string{"Choose option " + first + " for this " + noun + ".", "이 " + noun + "에서는 " + koFirst + " 것을 선택해줘."}
}

func digest(raw []byte) string { s := sha256.Sum256(raw); return hex.EncodeToString(s[:]) }
