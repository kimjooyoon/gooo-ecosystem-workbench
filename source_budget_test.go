package workbench

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeGenerationReportsBudgetWithoutPolicy(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual Gooo/native verification")
	}
	template, err := os.ReadFile("examples/source-budget/assembly.gooo")
	if err != nil {
		t.Fatal(err)
	}
	for _, budget := range []int{1, 3, 8, 16} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "source.gooo")
			if err := os.WriteFile(source, []byte(strings.Replace(string(template), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1)), 0644); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			raw, err := exec.CommandContext(ctx, compiler, "body-compose", "--source", source, "--cases", "examples/source-budget/cases.json", "--out", filepath.Join(root, "composition")).Output()
			if err != nil {
				t.Fatal("native construction", err)
			}
			s, err := ReadSnapshot(raw)
			if err != nil || len(s.Construction) != 1 {
				t.Fatal(s, err)
			}
			o := s.Construction[0]
			if !o.BudgetKnown || o.Budget != int64(budget) || o.Ranked != 8 || o.Scored > min(o.Budget, o.Ranked) || o.BudgetSource != "source_contract" || !o.Consistent {
				t.Fatal("native source limit lost", o)
			}
			plans, err := constructionNextSteps(ctx, Options{Compiler: compiler}, root, s)
			want := "raise-attempt-budget"
			if budget >= 8 {
				want = "observe-new-inputs"
			}
			if err != nil || len(plans) != 1 || plans[0].Action != want {
				t.Fatal("Gooo next action", plans, err)
			}
		})
	}
}
