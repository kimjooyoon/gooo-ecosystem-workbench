package main

import (
	"encoding/json"
	"os"
)

func main() {
	var in [][1]json.RawMessage
	if json.NewDecoder(os.Stdin).Decode(&in) != nil || len(in) > 128 {
		os.Exit(2)
	}
	out := make([][2]json.RawMessage, len(in))
	for c, row := range in {
		var input0 GoooRecordc1442f50e9f9e3b990518226f73cc3c326f8830b25c9b0d21eefc14c162c2f5f
		if json.Unmarshal(row[0], &input0) != nil {
			os.Exit(2)
		}
		v0 := GoooComposedActivity0(input0)
		out[c][0], _ = json.Marshal(v0)
		v1 := GoooComposedActivity1(v0)
		out[c][1], _ = json.Marshal(v1)
	}
	if json.NewEncoder(os.Stdout).Encode(out) != nil {
		os.Exit(3)
	}
}
