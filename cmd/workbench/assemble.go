package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func runAssemble(args []string) error {
	f := flag.NewFlagSet("assemble", flag.ContinueOnError)
	var o workbench.Options
	var request workbench.AssemblyRequest
	jsonOutput := f.Bool("json", false, "emit the complete machine-readable report; originals are always saved")
	graph := f.Bool("graph", false, "retain the full native graph, multiple assemblers and caller roots; requires body-plan")
	f.StringVar(&o.Compiler, "compiler", "gooo", "Gooo executable; --graph also requires body-plan")
	f.StringVar(&o.Model, "model", "", "optional local record model or builtin; omission is deterministic")
	f.StringVar(&o.Out, "out", "", "new output directory containing result, generated Go and original records")
	f.StringVar(&request.Source, "source", "", "Gooo source with assemblers, bound consumers and helpers")
	f.StringVar(&request.Entry, "entry", "", "final activity to execute, including its bound producers")
	f.StringVar(&request.AssemblyActivity, "assembly-activity", "", "root record activity to check; defaults to --entry")
	f.StringVar(&request.Cases, "cases", "", "caller cases; --graph also accepts an input-only document")
	if err := f.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *graph {
		report, err := workbench.AssembleGraph(ctx, o, request)
		if err != nil {
			return err
		}
		if *jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}
		root, err := filepath.Abs(o.Out)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "그래프 조립: %s · 본문 %d개\n호출 기대값: %d/%d · 필드: %d/%d\n조립 추론: %d회 · 저장 재생: 확인\n계획: %s\n생성 코드: %s\n다음 작업: %s\n",
			report.Entry, len(report.Bodies), report.Observation.NamedPassed, report.Observation.NamedTotal,
			report.Observation.FieldsPassed, report.Observation.FieldsTotal, report.Observation.ModelCalls,
			filepath.Join(root, "plan.json"), filepath.Join(root, "composition/generated.go"), report.Next.Message)
		if report.Next.Action == "add-counterexamples-to-construction" {
			fmt.Fprintf(os.Stdout, "이어서 실행: gooo-workbench construct --assembly %q --compiler %q --out %q\n", root, o.Compiler, root+"-next")
		}
		return nil
	}
	report, err := workbench.Assemble(ctx, o, request)
	if err != nil {
		return err
	}
	if !*jsonOutput {
		root, err := filepath.Abs(o.Out)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%s\n경로: %s\n조립용 사례의 필드: %d/%d\n호출 사례: %d/%d · 필드: %d/%d\n조립 추론: %d회 · 저장 재생: 확인\n결과: %s\n생성 코드: %s\n",
			report.Route.Message, report.Route.Action, report.Observation.SelectionPassed, report.Observation.SelectionTotal,
			report.Observation.NamedPassed, report.Observation.NamedTotal, report.Observation.FieldsPassed, report.Observation.FieldsTotal,
			report.Observation.ModelCalls, filepath.Join(root, "report.json"), filepath.Join(root, "composition", "generated.go"))
		fmt.Fprintf(os.Stdout, "다음 작업: %s\n후속 입력: %s\n", report.Next.Message, filepath.Join(root, report.NextContext))
		if report.Next.Action == "add-counterexamples-to-construction" {
			fmt.Fprintf(os.Stdout, "이어서 실행: gooo-workbench construct --assembly %q --out %q\n", root, root+"-next")
		}
		return nil
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
