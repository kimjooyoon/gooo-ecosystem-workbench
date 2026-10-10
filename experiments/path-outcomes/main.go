package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	compiler := flag.String("compiler", "", "Gooo compiler executable")
	sourcePath := flag.String("source", "", "authoritative Gooo source")
	activity := flag.String("activity", "", "integer activity to observe")
	casesPath := flag.String("cases", "", "consumed finite activity target cases")
	out := flag.String("out", "", "fresh directory outside the checkout")
	flag.Parse()
	if flag.NArg() != 0 || *compiler == "" || *sourcePath == "" || *activity == "" || *casesPath == "" || *out == "" {
		panic("compiler, source, activity, cases and out are required")
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	must(err)
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	if len(dirty) != 0 {
		panic("clean committed producer required")
	}
	absolute, err := filepath.Abs(*out)
	must(err)
	relative, err := filepath.Rel(strings.TrimSpace(string(root)), absolute)
	must(err)
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		panic("observation output must be outside the source checkout")
	}
	source, err := os.ReadFile(*sourcePath)
	must(err)
	cases, err := os.ReadFile(*casesPath)
	must(err)
	_, err = decodeCases(cases)
	must(err)
	must(os.Mkdir(absolute, 0755))
	must(os.WriteFile(filepath.Join(absolute, "source.gooo"), source, 0644))
	must(os.WriteFile(filepath.Join(absolute, "consumed-cases.json"), cases, 0644))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build, err := run(ctx, "", *compiler, "version", "--build", "--json")
	must(os.WriteFile(filepath.Join(absolute, "compiler-build.json"), build, 0644))
	must(err)
	args := []string{"body-context", "--feature-version", "semantic_context_intent_v3", "--include-plan", "--activity", *activity, filepath.Join(absolute, "source.gooo")}
	export, err := run(ctx, "", *compiler, args...)
	must(os.WriteFile(filepath.Join(absolute, "context.json"), export, 0644))
	must(err)
	e, prepared, err := bindExport(export, source)
	must(err)
	started := time.Now()
	result, err := observe(ctx, e, prepared, cases)
	must(err)
	evaluateNS := time.Since(started).Nanoseconds()
	// Retain interpreted observations even if native compilation/parity fails.
	must(save(filepath.Join(absolute, "interpreted.json"), result))
	must(native(ctx, absolute, e.Plan.Base.Name, &result))
	must(save(filepath.Join(absolute, "targets.json"), result))
	must(save(filepath.Join(absolute, "provenance.json"), map[string]any{
		"schema": "gooo/finite-path-target-producer/v1", "producer": strings.TrimSpace(string(head)),
		"go_version": runtime.Version(), "source_sha256": digest(source), "context_sha256": digest(export),
		"compiler_build_sha256": digest(build), "consumed_cases_sha256": digest(cases),
		"sdk_enumeration_ns": evaluateNS, "sdk_version": "v0.2.26-experimental",
		"host_cpu_utilization": "UNMEASURED", "gpu_utilization": "NOT_USED", "training": "NOT_PERFORMED",
	}))
	fmt.Printf("%d declared paths; %d finite-compatible; %d incompatible marginal combinations; %d native outcomes agree\n",
		result.Declared, len(result.Compatible), result.InvalidProduct, result.NativeChecked)
}

func run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=go1.27.2")
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	raw, err := cmd.Output()
	if err != nil {
		return raw, fmt.Errorf("%s: %w: %s", name, err, diagnostic.String())
	}
	return raw, nil
}

func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
