package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func read(path string) any {
	raw, e := os.ReadFile(path)
	if e != nil {
		panic(e)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if e := d.Decode(&v); e != nil {
		panic(e)
	}
	return v
}
func field(v any, names ...string) any {
	for _, name := range names {
		m, ok := v.(map[string]any)
		if !ok {
			panic(name)
		}
		var found bool
		v, found = m[name]
		if !found {
			panic("missing " + name)
		}
	}
	return v
}
func eq(a, b any, name string) {
	if a != b {
		panic("mismatch " + name)
	}
}
func num(v any) string {
	n, ok := v.(json.Number)
	if !ok {
		panic("number absent")
	}
	return n.String()
}

func main() {
	if len(os.Args) != 4 {
		panic("preflight native replay required")
	}
	pre, native, replay := read(os.Args[1]), read(os.Args[2]), read(os.Args[3])
	eq(num(field(pre, "model_predictions")), "0", "inspection predictions")
	eq(num(field(pre, "candidate_tests")), "0", "inspection tests")
	eq(field(pre, "model_compatibility", "status"), "READY_FOR_RANKING", "per-choice representation")
	input := field(pre, "inputs").([]any)[0]
	for _, v := range []any{native, replay} {
		r := field(v, "evaluation", "runtime")
		eq(field(r, "producer_source_sha"), "426caecb711e47da26fe659237a117f301d112f4", "actual compiler")
		eq(num(field(r, "finite_passed")), "3", "finite passed")
		eq(num(field(r, "finite_total")), "3", "finite total")
		eq(num(field(v, "evaluation", "new_model_calls")), "0", "fresh evaluation inference")
		preparations := field(v, "construction", "initial", "preparations").([]any)
		eq(len(preparations), 1, "one typed preparation")
		body := field(preparations[0], "generation", "report", "body_paths")
		eq(num(field(body, "search", "selection", "local_model_predictions")), "1", "real own model call")
		actual := field(body, "model_context", "inputs").([]any)[0]
		eq(field(actual, "input_sha256"), field(input, "input_sha256"), "inspection input parity")
		for _, key := range []string{"metadata_sha256", "weights_sha256", "model_schema", "feature_version"} {
			eq(field(body, "model_retention", key), field(pre, "model_compatibility", "model", key), "model "+key)
		}
		traces := field(r, "traces").([]any)
		eq(len(traces), 3, "trace count")
		d := field(traces[2], "deliveries").([]any)[0]
		eq(num(field(d, "input")), "9007199254740993", "exact integer input")
		eq(num(field(d, "actual")), "18014398509481986", "exact integer output")
		eq(num(field(d, "expected")), "18014398509481986", "exact integer expectation")
	}
	eq(field(replay, "generated_now"), false, "saved projection")
	eq(field(replay, "evaluation", "construction_replayed"), true, "saved history")
	fmt.Println("source/model/input parity, finite3/3, zero new replay and exact integers verified")
}
