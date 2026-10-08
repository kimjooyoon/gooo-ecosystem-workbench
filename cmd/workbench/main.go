package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("use verify, scaffold, splice, construct, diagnose, refine, reference, discover, receipt, or feature-audit; each command accepts --help")
	}
	if args[0] == "refine" {
		return runRefine(args[1:])
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var o workbench.Options
	f.StringVar(&o.Compiler, "compiler", "gooo", "Gooo compiler executable")
	f.StringVar(&o.Model, "model", "", "optional model.json path or builtin; omission is deterministic")
	f.StringVar(&o.Out, "out", "", "new output directory")
	profile := f.String("profile", "record", "starter profile: scalar, record, or library")
	input := f.String("input", "", "splice/diagnostic/feature-audit input file, or completed verify output directory for receipt")
	packageDir := f.String("package", "", "Gooo package directory for reference")
	entry := f.String("entry", "", "entry activity for reference or construct")
	query := f.String("query", "", "natural-language capability question for discover")
	declaration := f.String("declaration", "", "optional .gooo declaration file to bind to the discovery")
	sourceFile := f.String("source", "", "Gooo source for construct")
	constructionCases := f.String("construction-cases", "", "caller expectations used by construct")
	evaluationCases := f.String("evaluation-cases", "", "subsequent evaluation cases for construct")
	maxPrograms := f.Int64("max-program-budget", 8, "construct ceiling for whole-program attempts per round (1..64)")
	maxRounds := f.Int("max-rounds", 4, "construct ceiling for fresh construction rounds (1..16)")
	if e := f.Parse(args[1:]); e != nil {
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	ctx := context.Background()
	var value any
	var err error
	switch args[0] {
	case "construct":
		value, err = workbench.ConstructJoint(ctx, o, workbench.JointRequest{Source: *sourceFile,
			ConstructionCases: *constructionCases, EvaluationCases: *evaluationCases, Entry: *entry,
			MaxProgramBudget: *maxPrograms, MaxRounds: *maxRounds})
	case "splice":
		if *input == "" {
			return fmt.Errorf("splice requires --input")
		}
		raw, e := os.ReadFile(*input)
		if e != nil {
			return e
		}
		request, e := workbench.ReadSpliceRequest(raw)
		if e != nil {
			return e
		}
		value, err = workbench.SpliceSource(ctx, o, request)
	case "feature-audit":
		if *input == "" {
			return fmt.Errorf("feature-audit requires --input")
		}
		raw, e := os.ReadFile(*input)
		if e != nil {
			return e
		}
		value, err = workbench.AuditRecordFeatures(ctx, o, raw)
	case "verify":
		value, err = workbench.Verify(ctx, o)
	case "scaffold":
		value, err = workbench.Scaffold(ctx, o, *profile)
	case "diagnose":
		if *input == "" {
			return fmt.Errorf("diagnose requires --input")
		}
		b, e := os.ReadFile(*input)
		if e != nil {
			return e
		}
		s, e := workbench.ReadSnapshot(b)
		if e != nil {
			return e
		}
		value, err = workbench.Diagnose(ctx, o, s)
		if err == nil {
			err = os.WriteFile(filepath.Join(o.Out, "captured-input.json"), b, 0644)
		}
	case "reference":
		value, err = workbench.Reference(ctx, o, *packageDir, *entry)
	case "discover":
		if *query == "" {
			return fmt.Errorf("discover requires --query")
		}
		var source string
		if *declaration != "" {
			b, readErr := os.ReadFile(*declaration)
			if readErr != nil {
				return readErr
			}
			source = string(b)
		}
		value, err = workbench.DiscoverCapability(ctx, o, *query, source)
	case "receipt":
		value, err = workbench.CompletenessReceiptFor(ctx, o, *input)
	default:
		return fmt.Errorf("unknown command %q; use verify, scaffold, splice, construct, diagnose, refine, reference, discover, receipt, or feature-audit", args[0])
	}
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
