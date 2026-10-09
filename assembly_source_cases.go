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
	if c.Plan.Schema != "gooo/body-composition-plan/v1" || len(c.Plan.Activities) != 1 ||
		len(c.Plan.Preparations) != 0 || len(c.Preparations) != 0 || len(c.Steps) != 1 {
		return nil, "", fmt.Errorf("assembly feedback currently requires one root record activity without prepared helpers")
	}
	a := c.Plan.Activities[0]
	record := c.Steps[0].Generation.Report.Assembly
	if a.Name == "" || a.ID != activityID || c.Steps[0].Generation.Report.ActivityID != a.ID ||
		c.Plan.EntryActivity != a.Name || len(a.Inputs) == 0 || record == nil || len(record.Cases) == 0 {
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
	return encoded, a.Name, nil
}
