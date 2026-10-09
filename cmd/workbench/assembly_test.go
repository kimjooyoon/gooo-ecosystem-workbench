package main

import (
	"strings"
	"testing"
)

func TestAssemblyConstructRejectsConflictingInputs(t *testing.T) {
	for _, option := range []string{"--source", "--workspace", "--entry", "--construction-cases", "--evaluation-cases", "--fill-model"} {
		if err := run([]string{"construct", "--assembly", "unused", option, "unused"}); err == nil ||
			!strings.Contains(err.Error(), "exclusive") {
			t.Fatal(option, err)
		}
	}
	if err := run([]string{"verify", "--assembly", "unused"}); err == nil || !strings.Contains(err.Error(), "requires construct") {
		t.Fatal(err)
	}
}
