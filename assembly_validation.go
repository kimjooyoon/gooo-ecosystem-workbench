package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type assemblyModel struct {
	Schema      string `json:"schema"`
	Loaded      *bool  `json:"loaded"`
	MetadataSHA string `json:"metadata_sha256"`
	WeightsSHA  string `json:"weights_sha256"`
	ModelSchema string `json:"model_schema"`
	Feature     string `json:"feature_version"`
	Arithmetic  string `json:"arithmetic_version"`
}
type assemblyPreflight struct {
	Schema        string `json:"schema"`
	SourceSHA     string `json:"original_source_sha256"`
	ContractSHA   string `json:"contract_sha256"`
	ActivityID    string `json:"activity_id"`
	Predictions   *int   `json:"model_predictions"`
	Tests         *int   `json:"candidate_tests"`
	Compatibility *struct {
		Schema string        `json:"schema"`
		Status string        `json:"status"`
		Reason string        `json:"reason"`
		Model  assemblyModel `json:"model"`
	} `json:"model_compatibility,omitempty"`
	Context struct {
		Status  string `json:"status"`
		Reason  string `json:"reason"`
		Text    string `json:"text,omitempty"`
		SHA     string `json:"sha256"`
		Feature string `json:"feature_version"`
	} `json:"context"`
}

func readAssemblyPreflight(raw []byte, sourceSHA string, requested bool) (assemblyPreflight, error) {
	var p assemblyPreflight
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	if p.Schema != "gooo/record-assembly-input-export/v1" || p.SourceSHA != sourceSHA || !nativeDigest(p.ContractSHA) ||
		p.ActivityID == "" || p.Predictions == nil || *p.Predictions != 0 || p.Tests == nil || *p.Tests != 0 {
		return p, fmt.Errorf("preflight requires a source-bound record export and explicit zero prediction/test counters")
	}
	if requested {
		if p.Compatibility == nil || p.Compatibility.Schema != "gooo/record-model-compatibility/v1" {
			return p, fmt.Errorf("requested model has no verified compatibility record")
		}
		m := p.Compatibility.Model
		if m.Schema != "gooo/retained-path-model/v1" || m.Loaded == nil || !*m.Loaded ||
			!nativeDigest("sha256:"+m.MetadataSHA) || !nativeDigest("sha256:"+m.WeightsSHA) ||
			m.ModelSchema == "" || m.Feature == "" || m.Arithmetic == "" || m.Feature != p.Context.Feature {
			return p, fmt.Errorf("preflight model identity or source input contract is incomplete")
		}
		if p.Compatibility.Status == "READY_FOR_RANKING" && (p.Context.Status != "ENCODED" ||
			p.Context.Text == "" || p.Context.SHA != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(p.Context.Text)))) {
			return p, fmt.Errorf("ready model has no encoded source input")
		}
		if p.Compatibility.Status == "DECLINED_TO_DETERMINISTIC" && (p.Context.Status != p.Compatibility.Status ||
			p.Context.Reason != p.Compatibility.Reason || p.Context.Text != "" || p.Context.SHA != "") {
			return p, fmt.Errorf("declined model differs from its source input record")
		}
	}
	return p, nil
}

func readAssemblyExecution(raw []byte, p assemblyPreflight, generated, model bool) (result, error) {
	var r result
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	var evidence struct {
		Runtime struct {
			Stage      string
			Calls      *int `json:"model_calls"`
			Passed     *int `json:"finite_passed"`
			Total      *int `json:"finite_total"`
			Projection bool `json:"projection_replayed"`
			Replay     bool `json:"runtime_replayed"`
		}
		Composition struct {
			Steps []struct {
				Generation struct {
					Report struct {
						ActivityID string          `json:"activity_id"`
						Search     json.RawMessage `json:"body_search"`
						Fill       json.RawMessage `json:"body_fill"`
						Assembly   *struct {
							Calls   *int `json:"model_calls"`
							Passed  *int `json:"fields_passed"`
							Total   *int `json:"fields_total"`
							Model   assemblyModel
							Context struct {
								SHA string `json:"sha256"`
							} `json:"model_context"`
						} `json:"record_assembly"`
					}
				}
			}
		}
	}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		return r, err
	}
	x := evidence.Runtime
	if r.Generated != generated || r.Composition.OriginalSourceSHA != p.SourceSHA || !nativeDigest(r.Composition.GeneratedSHA) ||
		x.Stage != "COMPLETE" || x.Calls == nil || *x.Calls != 0 || x.Passed == nil || x.Total == nil ||
		!x.Projection || !x.Replay || len(evidence.Composition.Steps) < 1 ||
		len(evidence.Composition.Steps) != len(r.Composition.Steps) || len(r.Composition.Preparations) != 0 {
		return r, fmt.Errorf("assembly requires complete source-bound native execution without runtime inference")
	}
	selected := -1
	seen := map[string]bool{}
	for i, step := range evidence.Composition.Steps {
		report := step.Generation.Report
		if report.ActivityID == "" || seen[report.ActivityID] || present(report.Search) || present(report.Fill) {
			return r, fmt.Errorf("assembly requires distinct fixed consumers and one record assembler")
		}
		seen[report.ActivityID] = true
		if report.Assembly != nil {
			if selected >= 0 || report.ActivityID != p.ActivityID {
				return r, fmt.Errorf("assembly has an additional or unbound record assembler")
			}
			selected = i
		}
	}
	if selected < 0 {
		return r, fmt.Errorf("assembly omitted the preflight record activity")
	}
	a := evidence.Composition.Steps[selected].Generation.Report.Assembly
	wantCalls := 0
	if model {
		wantCalls = 1
	}
	if a == nil || a.Calls == nil || *a.Calls != wantCalls || a.Passed == nil || a.Total == nil ||
		*a.Passed < 0 || *a.Total < *a.Passed {
		return r, fmt.Errorf("assembly requires explicit selection counts and model calls matching the Gooo route")
	}
	if model {
		m := p.Compatibility.Model
		if a.Model.Loaded == nil || !*a.Model.Loaded || a.Model.Schema != m.Schema || a.Model.ModelSchema != m.ModelSchema ||
			a.Model.MetadataSHA != m.MetadataSHA || a.Model.WeightsSHA != m.WeightsSHA ||
			a.Model.Feature != m.Feature || a.Model.Arithmetic != m.Arithmetic || a.Context.SHA != p.Context.SHA {
			return r, fmt.Errorf("assembly model or input digest differs from preflight")
		}
	}
	return r, nil
}
