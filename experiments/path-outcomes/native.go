package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type nativeValue struct {
	Value   int64  `json:"value"`
	Failure string `json:"failure,omitempty"`
}

type nativeRow struct {
	Mask   int           `json:"mask"`
	Values []nativeValue `json:"values"`
}

// Candidate source is copied exactly from Program.GoSource, including its name
// and package. A separate runner imports each candidate in its own directory.
func native(ctx context.Context, out, name string, result *observation) error {
	dir := filepath.Join(out, "native")
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.invalid/gooo-observation\n\ngo 1.27.2\n"), 0644); err != nil {
		return err
	}
	var imports, calls strings.Builder
	count := 0
	for _, c := range result.Candidates {
		if c.Failure != "" {
			continue
		}
		label := fmt.Sprintf("c%02d", c.Mask)
		candidateDir := filepath.Join(dir, label)
		if err := os.Mkdir(candidateDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(candidateDir, "generated.go"), []byte(c.GoSource), 0644); err != nil {
			return err
		}
		fmt.Fprintf(&imports, "%s %q\n", label, "example.invalid/gooo-observation/"+label)
		fmt.Fprintf(&calls, "rows = append(rows, row{Mask:%d, Values:[]value{", c.Mask)
		for _, o := range c.Outcomes {
			fmt.Fprintf(&calls, "call(%s.%s, int64(%d)),", label, name, o.Input)
		}
		calls.WriteString("}})\n")
		count++
	}
	if count == 0 {
		return fmt.Errorf("no type-correct candidate has native evidence")
	}
	code := "package main\nimport(\"encoding/json\";\"fmt\";\"os\";\n" + imports.String() + `)
type value struct { Value int64 ` + "`json:\"value\"`" + `; Failure string ` + "`json:\"failure,omitempty\"`" + ` }
type row struct { Mask int ` + "`json:\"mask\"`" + `; Values []value ` + "`json:\"values\"`" + ` }
func call(f func(int64) int64, input int64) (v value) {
    defer func(){ if p := recover(); p != nil { v.Failure = fmt.Sprint(p) } }()
    v.Value = f(input); return
}
func main(){ rows := []row{}
` + calls.String() + `if err := json.NewEncoder(os.Stdout).Encode(rows); err != nil { panic(err) }
}`
	if err := os.Mkdir(filepath.Join(dir, "runner"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "runner", "main.go"), []byte(code), 0644); err != nil {
		return err
	}
	raw, err := run(ctx, dir, "go", "run", "./runner")
	if writeErr := os.WriteFile(filepath.Join(out, "native.stdout.json"), raw, 0644); writeErr != nil {
		return writeErr
	}
	if err != nil {
		_ = os.WriteFile(filepath.Join(out, "native.failure.txt"), []byte(err.Error()), 0644)
		return err
	}
	return compareNative(raw, result)
}

func compareNative(raw []byte, result *observation) error {
	result.NativeParity, result.NativeChecked = false, 0
	var rows []nativeRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	seen, checked := map[int]bool{}, 0
	for _, row := range rows {
		if row.Mask < 0 || row.Mask >= len(result.Candidates) || seen[row.Mask] {
			return fmt.Errorf("native mask missing, repeated or outside palette")
		}
		seen[row.Mask] = true
		c := result.Candidates[row.Mask]
		if c.Failure != "" || len(row.Values) != len(c.Outcomes) {
			return fmt.Errorf("native candidate case count differs")
		}
		for i, v := range row.Values {
			o := c.Outcomes[i]
			if (v.Failure != "") != (o.Failure != "") || (v.Failure == "" && (o.Actual == nil || v.Value != *o.Actual)) {
				return fmt.Errorf("native disagreement at mask %d input %d", row.Mask, o.Input)
			}
			checked++
		}
	}
	for _, c := range result.Candidates {
		if c.Failure == "" && !seen[c.Mask] {
			return fmt.Errorf("native result omitted candidate %d", c.Mask)
		}
	}
	result.NativeChecked, result.NativeParity = checked, true
	return nil
}
