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
		return fmt.Errorf("use verify, scaffold, diagnose, reference, or receipt; each command accepts --help")
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var o workbench.Options
	f.StringVar(&o.Compiler, "compiler", "gooo", "Gooo compiler executable")
	f.StringVar(&o.Model, "model", "", "optional model.json path or builtin; omission is deterministic")
	f.StringVar(&o.Out, "out", "", "new output directory")
	profile := f.String("profile", "record", "starter profile: scalar or record")
	input := f.String("input", "", "diagnostic input file, or completed verify output directory for receipt")
	packageDir := f.String("package", "", "Gooo package directory for reference")
	entry := f.String("entry", "", "public activity name for reference")
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
	case "receipt":
		value, err = workbench.CompletenessReceiptFor(ctx, o, *input)
	default:
		return fmt.Errorf("unknown command %q; use verify, scaffold, diagnose, reference, or receipt", args[0])
	}
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
