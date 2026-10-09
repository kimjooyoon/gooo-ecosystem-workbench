package workbench

import (
	"encoding/json"
	"fmt"
)

// sourceAssemblyCases changes only the envelope of the source-owned test rows.
// Raw input and expected JSON values preserve int64 precision and original oracles.
func sourceAssemblyCases(raw []byte, activityID string) ([]byte, string, error) {
	var saved struct {
		Composition struct {
			Plan struct {
				Schema        string
				EntryActivity string `json:"entry_activity"`
				Preparations  []json.RawMessage
				Activities    []struct {
					Name, ID string
					Inputs   []struct {
						Port string
						From int
					}
				}
			}
			Preparations []json.RawMessage
			Steps        []struct {
				Generation struct {
					Report struct {
						ActivityID string `json:"activity_id"`
						Assembly   *struct {
							Cases []struct {
								Inputs   []json.RawMessage
								Expected json.RawMessage
							}
						} `json:"record_assembly"`
					}
				}
			}
		}
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return nil, "", err
	}
	c := saved.Composition
	if c.Plan.Schema != "gooo/body-composition-plan/v1" || len(c.Plan.Activities) < 1 ||
		len(c.Plan.Preparations) != 0 || len(c.Preparations) != 0 || len(c.Steps) != len(c.Plan.Activities) {
		return nil, "", fmt.Errorf("assembly feedback requires one root record assembler and fixed bound consumers without prepared assembly helpers")
	}
	selected, entryFound := -1, false
	names, ids := map[string]bool{}, map[string]bool{}
	for i, a := range c.Plan.Activities {
		if a.Name == "" || a.ID == "" || names[a.Name] || ids[a.ID] || c.Steps[i].Generation.Report.ActivityID != a.ID {
			return nil, "", fmt.Errorf("assembly feedback graph activity identities are ambiguous")
		}
		names[a.Name], ids[a.ID] = true, true
		entryFound = entryFound || c.Plan.EntryActivity == a.Name
		if a.ID == activityID {
			selected = i
		} else if c.Steps[i].Generation.Report.Assembly != nil {
			return nil, "", fmt.Errorf("assembly feedback requires exactly one record assembler")
		}
		for _, input := range a.Inputs {
			if a.ID != activityID && (input.From < 0 || input.From >= i) {
				return nil, "", fmt.Errorf("assembly feedback cannot infer additional root inputs from local source cases")
			}
		}
	}
	if !entryFound || selected < 0 {
		return nil, "", fmt.Errorf("assembly feedback omitted its graph entry or record activity")
	}
	a := c.Plan.Activities[selected]
	record := c.Steps[selected].Generation.Report.Assembly
	if len(a.Inputs) == 0 || record == nil || len(record.Cases) == 0 {
		return nil, "", fmt.Errorf("assembly feedback needs the original entry, input ports and source cases")
	}
	ports := map[string]bool{}
	for _, input := range a.Inputs {
		if input.Port == "" || input.From != -1 || ports[input.Port] {
			return nil, "", fmt.Errorf("assembly feedback input ports are ambiguous or dependent")
		}
		ports[input.Port] = true
	}
	doc := jointCases{Schema: jointCasesSchema}
	for _, row := range record.Cases {
		if len(row.Inputs) != len(a.Inputs) || len(row.Expected) == 0 {
			return nil, "", fmt.Errorf("source assembly case omitted inputs or its expected value")
		}
		inputs := map[string]json.RawMessage{}
		for i, value := range row.Inputs {
			inputs[a.Name+"."+a.Inputs[i].Port] = value
		}
		encoded, err := json.Marshal(struct {
			Inputs   map[string]json.RawMessage `json:"inputs"`
			Expected map[string]json.RawMessage `json:"expected"`
		}{inputs, map[string]json.RawMessage{a.Name: row.Expected}})
		if err != nil {
			return nil, "", err
		}
		doc.Cases = append(doc.Cases, encoded)
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return nil, "", err
	}
	encoded = append(encoded, '\n')
	if _, err = readJointCases(encoded, false); err != nil {
		return nil, "", err
	}
	return encoded, c.Plan.EntryActivity, nil
}
