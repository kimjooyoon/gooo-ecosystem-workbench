package workbench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var assemblyOriginNames = []string{"source.gooo", "cases.json", "preflight.json", "assembly.json", "replay.json",
	"composition/generated.go", "composition/composition.json", "follow-up/assembly-next.gooo",
	"follow-up/inputs.json", "follow-up/execution.json", "follow-up/replay.json"}

// readAssemblyOrigin snapshots the exact bytes referenced by the retained context.
// Native replay subsequently checks the saved composition against its source.
func readAssemblyOrigin(directory string) (map[string][]byte, assemblyContext, error) {
	var c assemblyContext
	raw, err := os.ReadFile(filepath.Join(directory, "next-context.json"))
	if err != nil {
		return nil, c, err
	}
	if len(raw) > 1<<20 {
		return nil, c, fmt.Errorf("assembly context exceeds 1 MiB")
	}
	if err = json.Unmarshal(raw, &c); err != nil {
		return nil, c, err
	}
	if c.Schema != "gooo/assembly-next-context/v1" || len(c.Artifacts) != len(assemblyOriginNames) {
		return nil, c, fmt.Errorf("expected a complete retained assemble directory")
	}
	wanted := map[string]bool{}
	for _, name := range assemblyOriginNames {
		wanted[name] = true
	}
	files := map[string][]byte{"next-context.json": raw}
	total := 0
	for _, artifact := range c.Artifacts {
		if !wanted[artifact.Path] || files[artifact.Path] != nil {
			return nil, c, fmt.Errorf("assembly context has an unknown or duplicate artifact")
		}
		value, err := os.ReadFile(filepath.Join(directory, artifact.Path))
		if err != nil {
			return nil, c, err
		}
		total += len(value)
		if len(value) > 32<<20 || total > 64<<20 || int64(len(value)) != artifact.Bytes ||
			"sha256:"+jointDigest(value) != artifact.SHA256 {
			return nil, c, fmt.Errorf("retained assembly bytes differ: %s", artifact.Path)
		}
		files[artifact.Path] = value
	}
	if c.SourceSHA != "sha256:"+jointDigest(files["source.gooo"]) ||
		c.GeneratedSHA != "sha256:"+jointDigest(files["composition/generated.go"]) {
		return nil, c, fmt.Errorf("assembly context source or generated program differs")
	}
	return files, c, nil
}
