package main

import "testing"

func TestIndependentOraclesBoundaries(t *testing.T) {
	for _, test := range []struct {
		family, input, want string
		mask                uint16
	}{
		{"filenames", `["가.gooo"]`, `{"source":true,"stem":"가","bytes":8}`, 7},
		{"filenames", `[".gooo"]`, `{"source":false,"stem":"","bytes":5}`, 7},
		{"filenames", `[""]`, `{"source":false,"stem":"","bytes":0}`, 7},
		{"filenames", `["🙂.gooo.gooo"]`, `{"source":true,"stem":"🙂.gooo","bytes":14}`, 7},
		{"filenames", `["name.GOOO"]`, `{"source":true,"stem":"name.GOOO","bytes":0}`, 0},
		{"filenames", `[".hidden"]`, `{"source":false,"stem":".hidden","bytes":7}`, 4},
		{"division", `[-7,3]`, `{"valid":true,"quotient":-2,"remainder":-1}`, 7},
		{"division", `[7,-3]`, `{"valid":true,"quotient":-2,"remainder":1}`, 7},
		{"division", `[7,0]`, `{"valid":false,"quotient":0,"remainder":0}`, 7},
		{"division", `[7,0]`, `{"valid":true,"quotient":0,"remainder":0}`, 0},
		{"division", `[9007199254740993,1]`, `{"valid":true,"quotient":9007199254740993,"remainder":0}`, 7},
		{"division", `[-9223372036854775808,-1]`, `{"valid":true,"quotient":-9223372036854775808,"remainder":0}`, 7},
		{"division", `[-9223372036854775808,-1]`, `{"valid":false,"quotient":-9223372036854775808,"remainder":0}`, 0},
		{"retry", `[true,true,-1,4,2,10]`, `{"retry":false,"delay_ms":0,"reason":"invalid-input"}`, 7},
		{"retry", `[true,true,0,4,2,10]`, `{"retry":false,"delay_ms":0,"reason":"completed"}`, 7},
		{"retry", `[false,false,0,4,2,10]`, `{"retry":false,"delay_ms":0,"reason":"permanent-failure"}`, 7},
		{"retry", `[false,true,4,4,2,10]`, `{"retry":false,"delay_ms":0,"reason":"attempt-limit"}`, 7},
		{"retry", `[false,true,0,4,0,10]`, `{"retry":true,"delay_ms":1,"reason":"retry"}`, 7},
		{"retry", `[false,true,0,4,0,0]`, `{"retry":true,"delay_ms":0,"reason":"retry"}`, 7},
		{"retry", `[false,true,0,4,2,10]`, `{"retry":true,"delay_ms":4,"reason":"retry"}`, 7},
		{"retry", `[false,true,0,4,9223372036854775807,9223372036854775807]`, `{"retry":true,"delay_ms":9223372036854775807,"reason":"retry"}`, 7},
		{"retry", `[false,true,0,4,2,3]`, `{"retry":false,"delay_ms":3,"reason":"pending"}`, 2},
	} {
		t.Run(test.family+test.input, func(t *testing.T) {
			got, err := oracle(test.family, []byte(test.input), test.mask)
			if err != nil || !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s: %v", got, test.want, err)
			}
		})
	}
}

func TestOracleRejectsIllTypedInputs(t *testing.T) {
	for _, family := range []string{"filenames", "division", "retry", "unknown"} {
		for _, input := range []string{`null`, `[]`, `[null]`, `[null,1]`, `[1.5,1]`, `[9223372036854775808,1]`,
			`["1",1]`, `[false,true,0,1,null,2]`, `[false,true,0,1,0.5,2]`, `[false,true,0,1,0,9223372036854775808]`} {
			if _, err := oracle(family, []byte(input), 7); err == nil {
				t.Fatal("ill-typed input accepted", family, input)
			}
		}
	}
	if _, err := oracle("filenames", []byte(`["x"]`), 8); err == nil {
		t.Fatal("out-of-range requested behavior accepted")
	}
	if sameJSON([]byte(`9007199254740993`), []byte(`9007199254740992`)) ||
		sameJSON([]byte(`1 2`), []byte(`1`)) || sameJSON([]byte(`broken`), []byte(`broken`)) {
		t.Fatal("JSON comparison lost exact integers or accepted malformed input")
	}
}
