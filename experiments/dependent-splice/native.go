package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	wb "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

type observation struct {
	Generated   bool `json:"generated_now"`
	Composition struct {
		SHA   string `json:"generated_sha256"`
		Steps []struct {
			Generation struct {
				Report struct {
					Assembly json.RawMessage `json:"record_assembly"`
				} `json:"report"`
			} `json:"generation"`
		} `json:"steps"`
	} `json:"composition"`
	Runtime struct {
		Calls  int `json:"model_calls"`
		Passed int `json:"finite_passed"`
		Total  int `json:"finite_total"`
		Traces []struct {
			Index      int `json:"case_index"`
			Deliveries []struct {
				Actual wb.SpliceEdit `json:"actual"`
			} `json:"deliveries"`
		} `json:"traces"`
	} `json:"runtime"`
}

type counts struct {
	Cases        int `json:"cases"`
	Passed       int `json:"passed"`
	Fields       int `json:"fields"`
	FieldsPassed int `json:"fields_passed"`
}

func execute(ctx context.Context, compiler, output string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, compiler, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if writeErr := os.WriteFile(output, raw, 0644); writeErr != nil {
		return raw, writeErr
	}
	if stderr.Len() != 0 {
		if writeErr := os.WriteFile(output+".stderr", stderr.Bytes(), 0644); writeErr != nil {
			return raw, writeErr
		}
	}
	if err != nil {
		return raw, fmt.Errorf("%s: %w: %s", args[0], err, stderr.String())
	}
	return raw, nil
}

func recount(r observation, rows []wb.SpliceRequest) (counts, []wb.SpliceEdit, error) {
	var c counts
	if r.Runtime.Calls != 0 || len(r.Runtime.Traces) != len(rows) {
		return c, nil, fmt.Errorf("native trace count or inference differs")
	}
	actual := make([]wb.SpliceEdit, len(rows))
	seen := make([]bool, len(rows))
	for _, trace := range r.Runtime.Traces {
		i := trace.Index
		if i < 0 || i >= len(rows) || seen[i] || len(trace.Deliveries) != 1 {
			return c, nil, fmt.Errorf("invalid native case identity")
		}
		seen[i] = true
		got, want := trace.Deliveries[0].Actual, oracle(rows[i])
		actual[i] = got
		c.Cases++
		c.Fields += 3
		if got == want {
			c.Passed++
		}
		if got.Changed == want.Changed {
			c.FieldsPassed++
		}
		if got.Source == want.Source {
			c.FieldsPassed++
		}
		if got.Bytes == want.Bytes {
			c.FieldsPassed++
		}
	}
	if c.Passed != r.Runtime.Passed || c.Cases != r.Runtime.Total {
		return c, nil, fmt.Errorf("compiler counts disagree with independent oracle")
	}
	return c, actual, nil
}

func batches(root string, rows []wb.SpliceRequest) ([]string, [][]wb.SpliceRequest, error) {
	var paths []string
	var groups [][]wb.SpliceRequest
	for start := 0; start < len(rows); {
		end := start + 64
		if end > len(rows) {
			end = len(rows)
		}
		for len(suite(rows[start:end])) > 32768 && end > start+1 {
			end--
		}
		raw := suite(rows[start:end])
		if len(raw) > 32768 {
			return nil, nil, fmt.Errorf("oversized single case")
		}
		path := filepath.Join(root, fmt.Sprintf("cases-%02d.json", len(paths)))
		if err := os.WriteFile(path, raw, 0644); err != nil {
			return nil, nil, err
		}
		paths, groups = append(paths, path), append(groups, rows[start:end])
		start = end
	}
	return paths, groups, nil
}
