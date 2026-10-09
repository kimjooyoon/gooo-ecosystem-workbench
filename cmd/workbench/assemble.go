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
	f.StringVar(&o.Compiler, "compiler", "gooo", "Gooo compiler executable with body-context --model")
	f.StringVar(&o.Model, "model", "", "optional local record model or builtin; omission is deterministic")
	f.StringVar(&o.Out, "out", "", "new output directory containing result, generated Go and original records")
	f.StringVar(&request.Source, "source", "", "Gooo source with one record assembling activity")
	f.StringVar(&request.Entry, "entry", "", "activity name to assemble")
	f.StringVar(&request.Cases, "cases", "", "caller native cases with expected outputs")
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
		return nil
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
