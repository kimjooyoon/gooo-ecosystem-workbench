package main

import (
	"strings"
	"testing"
)

func TestAssembleHelpAndMissingInputs(t *testing.T) {
	if err := run([]string{"assemble", "--help"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"assemble"}); err == nil || !strings.Contains(err.Error(), "--source, --entry and --cases") {
		t.Fatal("missing inputs did not point to the supported flags", err)
	}
}
