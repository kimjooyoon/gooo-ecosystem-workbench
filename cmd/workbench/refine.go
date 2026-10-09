package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func runRefine(args []string) error {
	f := flag.NewFlagSet("refine", flag.ContinueOnError)
	var o workbench.RefineOptions
	f.StringVar(&o.Compiler, "compiler", "gooo", "Gooo compiler executable")
	f.StringVar(&o.Model, "model", "", "optional construction model path or builtin")
	f.StringVar(&o.Out, "out", "", "new output directory; original inputs stay unchanged")
	f.StringVar(&o.Source, "source", "", "Gooo source containing a record construction")
	f.StringVar(&o.Activity, "activity", "", "constructing activity name")
	f.StringVar(&o.Cases, "cases", "", "adaptive feedback cases with explicit expected outputs")
	f.StringVar(&o.Policy, "policy", "", "Gooo source revision policy")
	f.StringVar(&o.EvaluationCases, "evaluation-cases", "", "optional final cases withheld until source selection")
	f.IntVar(&o.MaxAttempts, "max-attempts", 8, "maximum selected activity attempt budget per refinement round")
	f.IntVar(&o.MaxRounds, "max-rounds", 4, "maximum source refinement rounds (1..8)")
	f.BoolVar(&o.SearchPolicy, "search-policy", false, "use Gooo search observations and source-declared alternatives")
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
	report, err := workbench.Refine(ctx, o)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
