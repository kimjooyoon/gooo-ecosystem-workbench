package main

import (
	"encoding/json"
	"os"
)

func main() {
	var in [][6]json.RawMessage
	if json.NewDecoder(os.Stdin).Decode(&in) != nil || len(in) > 128 {
		os.Exit(2)
	}
	out := make([][1]json.RawMessage, len(in))
	for c, row := range in {
		var input0 bool
		if json.Unmarshal(row[0], &input0) != nil {
			os.Exit(2)
		}
		var input1 bool
		if json.Unmarshal(row[1], &input1) != nil {
			os.Exit(2)
		}
		var input2 int64
		if json.Unmarshal(row[2], &input2) != nil {
			os.Exit(2)
		}
		var input3 int64
		if json.Unmarshal(row[3], &input3) != nil {
			os.Exit(2)
		}
		var input4 int64
		if json.Unmarshal(row[4], &input4) != nil {
			os.Exit(2)
		}
		var input5 int64
		if json.Unmarshal(row[5], &input5) != nil {
			os.Exit(2)
		}
		v0 := GoooComposedActivity0(input0, input1, input2, input3, input4, input5)
		out[c][0], _ = json.Marshal(v0)
	}
	if json.NewEncoder(os.Stdout).Encode(out) != nil {
		os.Exit(3)
	}
}
