package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// These small Go functions are independent of compiler candidate output.
// They preserve exact int64 JSON values and the Gooo examples' byte semantics.
func oracle(family string, raw []byte, wanted uint16) ([]byte, error) {
	if wanted > 7 {
		return nil, fmt.Errorf("requested behavior mask must be in 0..7")
	}
	var inputs []json.RawMessage
	if err := json.Unmarshal(raw, &inputs); err != nil {
		return nil, err
	}
	for _, input := range inputs {
		if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
			return nil, fmt.Errorf("typed source inputs cannot be null")
		}
	}
	var first, second [3]any
	var names [3]string
	switch family {
	case "filenames":
		var name string
		if len(inputs) != 1 || json.Unmarshal(inputs[0], &name) != nil {
			return nil, fmt.Errorf("one filename required")
		}
		suffix, visible := strings.HasSuffix(name, ".gooo"), !strings.HasPrefix(name, ".")
		names = [3]string{"source", "stem", "bytes"}
		first = [3]any{suffix || visible, name, int64(0)}
		second = [3]any{suffix && visible, strings.TrimSuffix(name, ".gooo"), int64(len(name))}
	case "division":
		var numerator, denominator int64
		if len(inputs) != 2 || json.Unmarshal(inputs[0], &numerator) != nil || json.Unmarshal(inputs[1], &denominator) != nil {
			return nil, fmt.Errorf("two exact int64 inputs required")
		}
		var quotient, remainder int64
		valid := denominator != 0
		if valid {
			quotient, remainder = numerator/denominator, numerator%denominator
		}
		names = [3]string{"valid", "quotient", "remainder"}
		first, second = [3]any{!valid, -quotient, -remainder}, [3]any{valid, quotient, remainder}
	case "retry":
		var done, temporary bool
		var attempt, limit, base, cap int64
		if len(inputs) != 6 || json.Unmarshal(inputs[0], &done) != nil || json.Unmarshal(inputs[1], &temporary) != nil {
			return nil, fmt.Errorf("retry requires two booleans and four integers")
		}
		for i, output := range []*int64{&attempt, &limit, &base, &cap} {
			if json.Unmarshal(inputs[i+2], output) != nil {
				return nil, fmt.Errorf("retry requires exact int64 inputs")
			}
		}
		again, delay, reason := retryOracle(done, temporary, attempt, limit, base, cap)
		names = [3]string{"retry", "delay_ms", "reason"}
		first, second = [3]any{false, int64(0), "pending"}, [3]any{again, delay, reason}
	default:
		return nil, fmt.Errorf("unknown source family %s", family)
	}
	result := make(map[string]any, 3)
	for bit, name := range names {
		result[name] = first[bit]
		if wanted&(1<<bit) != 0 {
			result[name] = second[bit]
		}
	}
	return json.Marshal(result)
}

func retryOracle(done, temporary bool, attempt, limit, base, cap int64) (bool, int64, string) {
	if attempt < 0 || limit < 0 || base < 0 || cap < 0 {
		return false, 0, "invalid-input"
	}
	if done {
		return false, 0, "completed"
	}
	if !temporary {
		return false, 0, "permanent-failure"
	}
	if attempt >= limit {
		return false, 0, "attempt-limit"
	}
	delay := cap
	if base <= cap-base {
		delay = base * 2
	}
	if base == 0 && cap > 0 {
		delay = 1
	}
	return true, delay, "retry"
}
