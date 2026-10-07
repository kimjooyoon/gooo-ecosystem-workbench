// construction-loop dispatches Gooo's next-step proposal against an explicitly
// supplied workspace, saved receipt and continuation policy. It never edits source.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type event struct {
	Round   int      `json:"round"`
	Receipt string   `json:"receipt"`
	Actions []string `json:"actions"`
	Resumed bool     `json:"resumed"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	compiler := flag.String("compiler", "gooo", "Gooo compiler executable")
	workbench := flag.String("workbench", "workbench", "workbench executable")
	workspace := flag.String("workspace", "", "original workspace manifest")
	cases := flag.String("cases", "", "native expectation cases")
	policy := flag.String("policy", "", "explicit continuation policy workspace")
	receipt := flag.String("receipt", "", "saved package receipt")
	model := flag.String("model", "", "optional diagnostic model path or builtin")
	out := flag.String("out", "", "new experiment output directory")
	rounds := flag.Int("rounds", 2, "maximum diagnostic rounds; the last round only observes")
	flag.Parse()
	if *workspace == "" || *cases == "" || *policy == "" || *receipt == "" || *out == "" || *rounds < 1 || *rounds > 16 {
		return fmt.Errorf("provide workspace, cases, policy, receipt, new out, and 1..16 rounds")
	}
	if err := os.Mkdir(*out, 0755); err != nil {
		return err
	}
	raw, err := os.ReadFile(*receipt)
	if err != nil {
		return err
	}
	current := filepath.Join(*out, "initial.json")
	if err = os.WriteFile(current, raw, 0644); err != nil {
		return err
	}
	var events []event
	reason := "round-limit"
	for round := 0; round < *rounds; round++ {
		dir := filepath.Join(*out, fmt.Sprintf("diagnosis-%d", round))
		args := []string{"diagnose", "--compiler", *compiler, "--input", current, "--out", dir}
		if *model != "" {
			args = append(args, "--model", *model)
		}
		diagnostic, err := command(*workbench, args...)
		if err != nil {
			return err
		}
		var result struct {
			Plans []struct {
				Action string `json:"action"`
			} `json:"construction_next_steps"`
		}
		if err = json.Unmarshal(diagnostic, &result); err != nil {
			return err
		}
		e := event{Round: round, Receipt: filepath.Base(current)}
		resume := false
		for _, plan := range result.Plans {
			e.Actions = append(e.Actions, plan.Action)
			resume = resume || plan.Action == "resume-candidates"
		}
		if !resume || round == *rounds-1 {
			if !resume {
				reason = "no-resume-proposal"
			}
			events = append(events, e)
			break
		}
		next, err := command(*compiler, "package", "resume", "--json", "--receipt", current,
			"--cases", *cases, "--assembly-policy-workspace", *policy, *workspace)
		if err != nil {
			return err
		}
		current = filepath.Join(*out, fmt.Sprintf("resumed-%d.json", round))
		if err = os.WriteFile(current, next, 0644); err != nil {
			return err
		}
		e.Resumed = true
		events = append(events, e)
	}
	result := map[string]any{"schema": "gooo/construction-loop-experiment/v1", "stop_reason": reason,
		"events": events, "final_receipt": filepath.Base(current),
		"scope": "bounded dispatch of Gooo proposals with an explicit workspace and policy; no source changes or model training"}
	raw, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*out, "loop.json"), append(raw, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

func command(binary string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stderr = os.Stderr
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", filepath.Base(binary), args[0], err)
	}
	return raw, nil
}
