package workbench

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type assemblyNextInput struct {
	CallerPassed    int `json:"caller_passed"`
	CallerTotal     int `json:"caller_total"`
	FieldsPassed    int `json:"fields_passed"`
	FieldsTotal     int `json:"fields_total"`
	SelectionPassed int `json:"selection_passed"`
	SelectionTotal  int `json:"selection_total"`
}

type assemblyFailure struct {
	CaseIndex int    `json:"case_index"`
	Pointer   string `json:"original_json_pointer"`
}

type assemblyContext struct {
	Schema       string            `json:"schema"`
	SourceSHA    string            `json:"source_sha256"`
	ActivityID   string            `json:"activity_id"`
	Entry        string            `json:"entry_activity,omitempty"`
	GeneratedSHA string            `json:"selected_program_sha256"`
	Observation  Summary           `json:"observation"`
	Next         policyAdvice      `json:"next"`
	Mismatches   int               `json:"mismatched_deliveries"`
	Failures     []assemblyFailure `json:"first_mismatches"`
	Artifacts    []processArtifact `json:"artifacts"`
	Scope        string            `json:"scope"`
}

// saveAssemblyContext connects a finite observation to the next tool. It leaves
// original expectations intact and retains at most eight mismatch locations.
func saveAssemblyContext(root string, report AssemblyReport, execution result) error {
	c := assemblyContext{Schema: "gooo/assembly-next-context/v1", SourceSHA: report.Preflight.SourceSHA,
		ActivityID: report.Preflight.ActivityID, GeneratedSHA: report.GeneratedSHA, Observation: report.Observation,
		Next: report.Next, Failures: []assemblyFailure{},
		Scope: "Finite caller and source selection observations remain separate. Gooo advice is unscored and replayed. " +
			"References bind saved bytes; at most eight mismatch locations are shown. No next action or model call is executed."}
	if err := assemblyContextFailures(&c, execution); err != nil {
		return err
	}
	if c.Mismatches != report.Observation.NamedTotal-report.Observation.NamedPassed {
		return fmt.Errorf("context mismatch locations differ from the finite caller observation")
	}
	return saveAssemblyContextArtifacts(root, report.NextContext, c, assemblyOriginNames)
}

func assemblyContextFailures(c *assemblyContext, execution result) error {
	for ti, trace := range execution.Runtime.Traces {
		for di, delivery := range trace.Deliveries {
			if len(delivery.Expected) == 0 {
				continue
			}
			actual, err := decodeValue(delivery.Actual)
			if err != nil {
				return err
			}
			expected, err := decodeValue(delivery.Expected)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(actual, expected) {
				c.Mismatches++
				if len(c.Failures) < 8 {
					c.Failures = append(c.Failures, assemblyFailure{trace.CaseIndex,
						fmt.Sprintf("/runtime/traces/%d/deliveries/%d", ti, di)})
				}
			}
		}
	}
	return nil
}

func saveAssemblyContextArtifacts(root, name string, c assemblyContext, names []string) error {
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		c.Artifacts = append(c.Artifacts, processArtifact{Path: name,
			SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), Bytes: int64(len(raw))})
	}
	return save(filepath.Join(root, name), c)
}
