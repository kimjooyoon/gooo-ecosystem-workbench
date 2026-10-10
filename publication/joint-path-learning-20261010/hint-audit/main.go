package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strconv"
)

func readSource(path string) []byte {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		raw, err = os.ReadFile(path + ".gz")
		if err != nil {
			panic(err)
		}
		z, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			panic(err)
		}
		defer z.Close()
		raw, err = io.ReadAll(z)
	}
	if err != nil {
		panic(err)
	}
	return raw
}

func integer(e ast.Expr, input int64) int64 {
	switch v := e.(type) {
	case *ast.Ident:
		if v.Name == "input" {
			return input
		}
	case *ast.BasicLit:
		if v.Kind == token.INT {
			n, err := strconv.ParseInt(v.Value, 0, 64)
			if err == nil {
				return n
			}
		}
	case *ast.ParenExpr:
		return integer(v.X, input)
	}
	panic("diagnostic requires an input/int64 literal operand")
}

func main() {
	if len(os.Args) != 2 {
		panic("original native directory required")
	}
	rows := []map[string]any{}
	for _, arm := range []string{"previous", "control", "joint"} {
		raw := readSource(os.Args[1] + "/" + arm + "/generated.go")
		files := token.NewFileSet()
		file, err := parser.ParseFile(files, "generated.go", raw, 0)
		if err != nil {
			panic(err)
		}
		var conditions []ast.Expr
		ast.Inspect(file, func(n ast.Node) bool {
			if v, ok := n.(*ast.IfStmt); ok {
				conditions = append(conditions, v.Cond)
			}
			return true
		})
		if len(conditions) != 1 {
			panic("diagnostic is scoped to the retained single-condition fixture")
		}
		condition, ok := conditions[0].(*ast.BinaryExpr)
		if !ok || condition.Op != token.LSS {
			panic("diagnostic requires a less-than condition")
		}
		var rendered bytes.Buffer
		if err := format.Node(&rendered, files, condition); err != nil {
			panic(err)
		}
		cases := []map[string]any{}
		passed := 0
		for _, input := range []int64{-1, 0, 1} {
			actual := integer(condition.X, input) < integer(condition.Y, input)
			expected := input < 0
			if actual == expected {
				passed++
			}
			cases = append(cases, map[string]any{"input": input, "expected_negative_predicate": expected, "actual_condition": actual, "matched": actual == expected})
		}
		rows = append(rows, map[string]any{"arm": arm, "generated_sha256": fmt.Sprintf("sha256:%x", sha256.Sum256(raw)),
			"condition": rendered.String(), "matched": passed, "total": len(cases), "cases": cases})
	}
	out := map[string]any{"schema": "gooo/posthoc-path-hint-audit/v1",
		"authored_hint": "음수 입력인지 비교한다. Compare whether input is negative.",
		"scope":         "Post-hoc exact AST observation of the retained generated condition and three authored truth-table probes. This local hint was not a formal enforced assembly constraint and these are not held-out function cases. No new generation or training.", "results": rows}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
}
