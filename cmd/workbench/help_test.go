package main

import (
	"strings"
	"testing"
)

func TestHelpDoesNotExecuteWork(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		if err := run(args); err != nil {
			t.Errorf("top-level help %v: %v", args, err)
		}
	}
	names := "verify scaffold splice assemble construct diagnose refine reference discover " +
		"receipt feature-audit api-diff"
	for _, name := range strings.Fields(names) {
		for _, args := range [][]string{{name, "--help"}, {"help", name}} {
			if err := run(args); err != nil {
				t.Errorf("command help %v executed work or returned an error: %v", args, err)
			}
		}
	}
}

func TestHelpRejectsUnknownCommandAndExtraArguments(t *testing.T) {
	for _, args := range [][]string{{"unknown", "--help"}, {"help", "unknown"}} {
		if err := run(args); err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("unknown command %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"help", "assemble", "extra"}, {"--help", "extra"}} {
		if err := run(args); err == nil {
			t.Errorf("extra help arguments were ignored: %v", args)
		}
	}
}
