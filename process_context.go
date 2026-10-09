package workbench

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type processContext struct {
	Schema       string               `json:"schema"`
	InputSHA     string               `json:"input_sha256"`
	GeneratedSHA string               `json:"selected_program_sha256"`
	Observation  Summary              `json:"observation"`
	Processes    []processContextItem `json:"processes"`
	Artifacts    []processArtifact    `json:"artifacts"`
	Scope        string               `json:"scope"`
}

type processContextItem struct {
	Location        string       `json:"original_json_pointer"`
	Stage           string       `json:"reported_runtime_stage"`
	Input           processInput `json:"policy_input"`
	Advice          policyAdvice `json:"advice"`
	EvidencePointer string       `json:"observation_json_pointer"`
}

type processArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// saveProcessContext projects Gooo's result for the next tool or model. Full
// evidence stays on disk; digest-bound relative references allow selective reads.
func saveProcessContext(root string, s Snapshot, advice []policyAdvice, summary Summary, program string) error {
	if len(s.Processes) != len(advice) {
		return fmt.Errorf("process context requires one Gooo advice per observation")
	}
	c := processContext{Schema: "gooo/process-next-context/v1", InputSHA: s.InputSHA,
		GeneratedSHA: program, Observation: summary, Processes: make([]processContextItem, len(advice)),
		Scope: "Reported process state and Gooo-produced next actions; full evidence is in the referenced files. " +
			"Advice is unscored on actual inputs. Authored selection checks are separate. " +
			"References bind the saved bytes, not the truth of reported identities. No action is executed."}
	for i, a := range advice {
		p := s.Processes[i]
		c.Processes[i] = processContextItem{Location: p.Location, Stage: p.RuntimeStage, Input: p.Input,
			Advice: a, EvidencePointer: fmt.Sprintf("/native_processes/%d", i)}
	}
	for _, name := range []string{"observation.json", "process-next.gooo", "execution.json", "replay.json", "diagnostic.json"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		c.Artifacts = append(c.Artifacts, processArtifact{Path: name,
			SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), Bytes: int64(len(raw))})
	}
	return save(filepath.Join(root, "next-context.json"), c)
}
