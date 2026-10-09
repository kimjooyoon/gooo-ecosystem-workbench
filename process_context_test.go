package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessContextKeepsEvidenceAndTypedDecisionInputs(t *testing.T) {
	root := t.TempDir()
	s, err := ReadSnapshot(failedProcessFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	s.Processes[0].Input.Exit = 9007199254740993
	for _, name := range []string{"observation.json", "process-next.gooo", "execution.json", "replay.json", "diagnostic.json"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(strings.Repeat("large original output\n", 1000)), 0644); err != nil {
			t.Fatal(err)
		}
	}
	advice := []policyAdvice{{Code: "deadline-during-start", Message: "시작 제한", Action: "inspect-start-and-parent-budget"}}
	if err := saveProcessContext(root, s, advice, Summary{ReplayVerified: true}, "sha256:program"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "next-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result processContext
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.InputSHA != s.InputSHA || result.GeneratedSHA != "sha256:program" || len(result.Processes) != 1 ||
		result.Processes[0].Advice != advice[0] || result.Processes[0].EvidencePointer != "/native_processes/0" ||
		result.Processes[0].Input != s.Processes[0].Input || !result.Observation.ReplayVerified ||
		!bytes.Contains(raw, []byte("9007199254740993")) || bytes.Contains(raw, []byte("large original output")) {
		t.Fatal("context lost typed inputs, evidence, or included full output", string(raw))
	}
	if len(result.Artifacts) != 5 {
		t.Fatal("missing source, observation, execution, replay or diagnostic artifact")
	}
	for _, artifact := range result.Artifacts {
		b, err := os.ReadFile(filepath.Join(root, artifact.Path))
		if err != nil || artifact.SHA256 != fmt.Sprintf("sha256:%x", sha256.Sum256(b)) || artifact.Bytes != int64(len(b)) {
			t.Fatal("artifact digest differs", artifact, err)
		}
	}
	if err := saveProcessContext(root, s, nil, Summary{}, ""); err == nil {
		t.Fatal("mismatched advice count accepted")
	}
	if err := os.Remove(filepath.Join(root, "replay.json")); err != nil {
		t.Fatal(err)
	}
	if err := saveProcessContext(root, s, advice, Summary{}, ""); err == nil {
		t.Fatal("missing evidence file accepted")
	}
}
