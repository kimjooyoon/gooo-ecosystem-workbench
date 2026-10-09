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

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("choose a command with gooo-workbench --help")
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return runHelp(args)
	}
	if !knownCommand(args[0]) {
		return fmt.Errorf("unknown command %q; use gooo-workbench --help", args[0])
	}
	if args[0] == "assemble" {
		return runAssemble(args[1:])
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
	before := f.String("before", "", "previous workspace manifest for api-diff")
	after := f.String("after", "", "current workspace manifest for api-diff")
	sourceFile := f.String("source", "", "Gooo source for construct")
	assemblyDir := f.String("assembly", "", "retained assemble directory for construct; source and caller cases are reused")
	workspaceFile := f.String("workspace", "", "Gooo package manifest for construct; exclusive with --source/--entry")
	constructionCases := f.String("construction-cases", "", "caller expectations used by construct")
	evaluationCases := f.String("evaluation-cases", "", "adaptive cases for construct; failures feed the next round")
	holdoutCases := f.String("holdout-cases", "", "optional final evaluation after selection stops; never fed back")
	fillModel := f.String("fill-model", "", "optional operation-classifier model for construct source_fill preparation")
	maxPrograms := f.Int64("max-program-budget", 8, "construct ceiling for whole-program attempts per round (1..64)")
	maxRounds := f.Int("max-rounds", 4, "construct ceiling for fresh construction rounds (1..16)")
	if e := f.Parse(args[1:]); e != nil {
		if e == flag.ErrHelp {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *fillModel != "" && args[0] != "construct" {
		return fmt.Errorf("--fill-model requires construct")
	}
	if *workspaceFile != "" && args[0] != "construct" {
		return fmt.Errorf("--workspace requires construct")
	}
	if *assemblyDir != "" && (args[0] != "construct" || *sourceFile != "" || *workspaceFile != "" ||
		*entry != "" || *constructionCases != "" || *evaluationCases != "" || *fillModel != "") {
		return fmt.Errorf("--assembly requires construct and is exclusive with source, workspace, entry, cases and fill-model options")
	}
	if (*before != "" || *after != "") && args[0] != "api-diff" {
		return fmt.Errorf("--before and --after require api-diff")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var value any
	var err error
	switch args[0] {
	case "api-diff":
		value, err = workbench.CompareAPI(ctx, o, *before, *after)
	case "construct":
		if *assemblyDir != "" {
			value, err = workbench.ConstructFromAssembly(ctx, o, workbench.AssemblyConstructionRequest{Assembly: *assemblyDir,
				HoldoutCases: *holdoutCases, MaxProgramBudget: *maxPrograms, MaxRounds: *maxRounds})
		} else {
			value, err = workbench.ConstructJoint(ctx, o, workbench.JointRequest{Source: *sourceFile, Workspace: *workspaceFile,
				ConstructionCases: *constructionCases, EvaluationCases: *evaluationCases, HoldoutCases: *holdoutCases, Entry: *entry,
				MaxProgramBudget: *maxPrograms, MaxRounds: *maxRounds, FillModel: *fillModel})
		}
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
		return fmt.Errorf("unknown command %q; use gooo-workbench --help", args[0])
	}
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
