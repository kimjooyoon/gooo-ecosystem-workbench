package workbench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func jointSourceInputs(request JointRequest) (map[string][]byte, error) {
	if (request.Source == "") == (request.Workspace == "") || request.Workspace != "" && request.Entry != "" {
		return nil, fmt.Errorf("construct needs either --source with optional --entry or --workspace with its declared entry")
	}
	if request.Workspace == "" {
		source, err := os.ReadFile(request.Source)
		return map[string][]byte{"source.gooo": source}, err
	}
	raw, err := os.ReadFile(request.Workspace)
	if err != nil {
		return nil, err
	}
	if len(raw) > 1<<20 {
		return nil, fmt.Errorf("workspace manifest exceeds 1 MiB")
	}
	var manifest struct {
		Schema   string `json:"schema"`
		Packages []struct {
			Sources []string `json:"sources"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	if manifest.Schema != "gooo/package-workspace-manifest/v1" || len(manifest.Packages) == 0 {
		return nil, fmt.Errorf("expected a Gooo package workspace")
	}
	files := map[string][]byte{"workspace/gooo.workspace.json": raw}
	sourceCount, sourceBytes := 0, 0
	for _, pkg := range manifest.Packages {
		if len(pkg.Sources) == 0 {
			return nil, fmt.Errorf("workspace package has no sources")
		}
		for _, name := range pkg.Sources {
			if !filepath.IsLocal(name) {
				return nil, fmt.Errorf("workspace snapshot needs manifest-relative source files")
			}
			key := filepath.Join("workspace", name)
			if key == filepath.Join("workspace", "gooo.workspace.json") {
				return nil, fmt.Errorf("workspace source overlaps the copied manifest")
			}
			if files[key] != nil {
				continue
			}
			source, err := os.ReadFile(filepath.Join(filepath.Dir(request.Workspace), name))
			if err != nil {
				return nil, err
			}
			sourceCount++
			sourceBytes += len(source)
			if sourceCount > 512 || len(source) > 1<<20 || sourceBytes > 16<<20 {
				return nil, fmt.Errorf("workspace snapshot exceeds compiler source bounds")
			}
			files[key] = source
		}
	}
	return files, nil
}

func jointRoundArgs(root, dir, cases string, budget int64, request JointRequest, model, fillModel string) []string {
	args := []string{"body-construct", "--source", filepath.Join(root, "source.gooo"), "--out", filepath.Join(root, dir)}
	if request.Workspace != "" {
		args = []string{"package", "construct", "--json"}
	}
	args = append(args, "--construction-cases", filepath.Join(root, cases), "--cases", filepath.Join(root, "evaluation-cases.json"), "--attempts", strconv.FormatInt(budget, 10))
	if request.Entry != "" {
		args = append(args, "--entry", request.Entry)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	if fillModel != "" {
		args = append(args, "--fill-model", fillModel)
	}
	if request.Workspace != "" {
		args = append(args, filepath.Join(root, "workspace", "gooo.workspace.json"))
	}
	return args
}

func jointHoldoutArgs(root, dir string, request JointRequest) []string {
	if request.Workspace != "" {
		return []string{"package", "construct", "--json", "--receipt", filepath.Join(root, dir, "package-construction.json"),
			"--cases", filepath.Join(root, "holdout-cases.json"), filepath.Join(root, "workspace", "gooo.workspace.json")}
	}
	return []string{"body-construct", "--source", filepath.Join(root, "source.gooo"), "--construction", filepath.Join(root, dir, "construction.json"), "--cases", filepath.Join(root, "holdout-cases.json")}
}

func savePackageJointRound(root, dir string, raw []byte) error {
	if err := os.Mkdir(filepath.Join(root, dir), 0755); err != nil {
		return err
	}
	// The full unmodified package receipt is replay authority. These selected source
	// files are convenient exports and do not replace its original package image.
	if err := write(filepath.Join(root, dir, "package-construction.json"), raw); err != nil {
		return err
	}
	var r struct {
		Result struct {
			Construction struct {
				Source   string `json:"selected_source"`
				Selected struct {
					Source string `json:"source"`
					Driver string `json:"driver_source"`
				} `json:"selected"`
			} `json:"construction"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	c := r.Result.Construction
	if c.Source == "" || c.Selected.Source == "" || c.Selected.Driver == "" {
		return fmt.Errorf("package construction omitted its selected source exports")
	}
	for name, value := range map[string]string{"selected.gooo": c.Source, "generated.go": c.Selected.Source, "main.go": c.Selected.Driver} {
		if err := write(filepath.Join(root, dir, name), []byte(value)); err != nil {
			return err
		}
	}
	return write(filepath.Join(root, dir, "go.mod"), []byte("module gooo.observed.composition\n\ngo 1.27.2\n"))
}
