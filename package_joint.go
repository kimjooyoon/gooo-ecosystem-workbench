package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const packageJointSchema = "gooo/workspace-caller-construction-receipt/v1"

type packageActivity struct {
	Package  string `json:"package_path"`
	Activity string `json:"activity"`
	Lowered  string `json:"lowered_name"`
}

type PackageConstructionObservation struct {
	ManifestSHA256 string            `json:"manifest_sha256"`
	SourceSHA256   string            `json:"lowered_source_sha256"`
	Entry          packageActivity   `json:"entry"`
	Activities     []packageActivity `json:"activities"`
	Scope          string            `json:"scope"`
}

type packageJointEnvelope struct {
	Schema       string `json:"schema"`
	Decision     string `json:"decision"`
	Error        string `json:"error"`
	ManifestSHA  string `json:"manifest_digest"`
	ReplayedFrom string `json:"replayed_from_sha256"`
	Result       struct {
		Schema       string `json:"schema"`
		ReplayedFrom string `json:"replayed_from_sha256"`
		Program      struct {
			Schema     string            `json:"schema"`
			Source     string            `json:"lowered_gooo_source"`
			Entry      packageActivity   `json:"entry"`
			Activities []packageActivity `json:"activities"`
		} `json:"program"`
		Cases        json.RawMessage `json:"construction_cases"`
		Construction json.RawMessage `json:"construction"`
		Evaluation   json.RawMessage `json:"evaluation"`
	} `json:"result"`
}

// This adapter recounts a supplied observation. Native source and historical
// execution validation remain the compiler's saved-construction operation.
func packageJointOutput(raw []byte) ([]byte, *PackageConstructionObservation, error) {
	var r packageJointEnvelope
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, nil, err
	}
	if r.Schema != packageJointSchema || r.Error != "" ||
		(r.Decision != "COMPLETE_FINITE" && r.Decision != "PARTIAL_FINITE") ||
		r.ManifestSHA == "" || r.Result.Schema != "gooo/workspace-caller-construction/v1" ||
		r.Result.Program.Schema != "gooo/workspace-body-program/v1" {
		return nil, nil, fmt.Errorf("package construction requires its original successful envelope")
	}
	var c struct {
		SourceSHA string          `json:"original_source_sha256"`
		Cases     json.RawMessage `json:"construction_cases"`
		Selected  struct {
			Plan struct {
				Activities []struct{ Name, ID string } `json:"activities"`
			} `json:"plan"`
		} `json:"selected"`
	}
	var e struct {
		Replayed *bool `json:"construction_replayed"`
	}
	if err := json.Unmarshal(r.Result.Construction, &c); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(r.Result.Evaluation, &e); err != nil {
		return nil, nil, err
	}
	if e.Replayed == nil || *e.Replayed != (r.ReplayedFrom != "") || *e.Replayed != (r.Result.ReplayedFrom != "") ||
		c.SourceSHA != "sha256:"+jointDigest([]byte(r.Result.Program.Source)) {
		return nil, nil, fmt.Errorf("package construction source or replay binding differs")
	}
	refs := r.Result.Program.Activities
	if len(refs) == 0 || len(refs) != len(c.Selected.Plan.Activities) {
		return nil, nil, fmt.Errorf("package activity mapping differs")
	}
	names, lowered := map[string]string{}, map[string]bool{}
	entry := false
	for _, ref := range refs {
		key := ref.Package + ":" + ref.Activity
		if ref.Package == "" || ref.Activity == "" || ref.Lowered == "" || names[key] != "" || lowered[ref.Lowered] {
			return nil, nil, fmt.Errorf("ambiguous package activity mapping")
		}
		names[key], lowered[ref.Lowered] = ref.Lowered, true
		entry = entry || ref == r.Result.Program.Entry
	}
	seen := map[string]bool{}
	for _, a := range c.Selected.Plan.Activities {
		if !lowered[a.Name] || a.ID == "" || seen[a.Name] {
			return nil, nil, fmt.Errorf("selected plan differs from package activity mapping")
		}
		seen[a.Name] = true
	}
	if !entry {
		return nil, nil, fmt.Errorf("package entry is outside the selected graph")
	}
	translated, err := translatePackageFeedback(r.Result.Cases, names)
	if err != nil {
		return nil, nil, err
	}
	original, err := decodeValue(c.Cases)
	if err != nil || !reflect.DeepEqual(translated, original) {
		return nil, nil, fmt.Errorf("original package cases differ from construction feedback")
	}
	output, err := json.Marshal(struct {
		Generated    bool            `json:"generated_now"`
		Construction json.RawMessage `json:"construction"`
		Evaluation   json.RawMessage `json:"evaluation"`
	}{!*e.Replayed, r.Result.Construction, r.Result.Evaluation})
	observation := &PackageConstructionObservation{ManifestSHA256: r.ManifestSHA, SourceSHA256: c.SourceSHA,
		Entry: r.Result.Program.Entry, Activities: refs,
		Scope: "reported package bindings and original caller rows checked against lowered observations; native source and execution are verified by compiler replay"}
	return output, observation, err
}

func translatePackageFeedback(raw []byte, names map[string]string) (any, error) {
	var suite struct {
		Schema string `json:"schema"`
		Cases  []struct {
			Inputs   map[string]json.RawMessage `json:"inputs"`
			Expected map[string]json.RawMessage `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &suite); err != nil {
		return nil, err
	}
	if suite.Schema != jointCasesSchema || len(suite.Cases) == 0 {
		return nil, fmt.Errorf("package construction cases are missing")
	}
	for i := range suite.Cases {
		for mode, values := range []map[string]json.RawMessage{suite.Cases[i].Inputs, suite.Cases[i].Expected} {
			mapped := map[string]json.RawMessage{}
			for key, value := range values {
				translated := names[key]
				if translated == "" && mode == 0 {
					for original, lowered := range names {
						if strings.HasPrefix(key, original+".") {
							if translated != "" {
								return nil, fmt.Errorf("ambiguous package input port")
							}
							translated = lowered + strings.TrimPrefix(key, original)
						}
					}
				}
				if translated == "" || mapped[translated] != nil {
					return nil, fmt.Errorf("unknown or duplicate package case activity")
				}
				mapped[translated] = value
			}
			if mode == 0 {
				suite.Cases[i].Inputs = mapped
			} else {
				suite.Cases[i].Expected = mapped
			}
		}
	}
	result, err := json.Marshal(suite)
	if err != nil {
		return nil, err
	}
	return decodeValue(result)
}

func readPackageJointSnapshot(raw []byte, inputSHA string) (Snapshot, error) {
	normalized, observation, err := packageJointOutput(raw)
	if err != nil {
		return Snapshot{}, err
	}
	s, err := readJointSnapshot(normalized, inputSHA)
	if err != nil {
		return s, err
	}
	var r packageJointEnvelope
	if err := json.Unmarshal(raw, &r); err != nil {
		return s, err
	}
	complete := s.Joint.Decision == "COMPLETE_FINITE" && s.Total > 0 && s.Passed == s.Total &&
		nativeActivityCount(s, false) == 0 && nativeActivityCount(s, true) == 0
	if (r.Decision == "COMPLETE_FINITE") != complete {
		return s, fmt.Errorf("package completion differs from recounted observations")
	}
	s.Package = observation
	return s, nil
}
