package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var firstIntents = map[string][3][2]string{
	"filenames": {
		{"확장자나 보이는 이름 조건 중 하나 이상을 만족한다.", "Require either the suffix or visible-name condition."},
		{"입력 이름을 접미사까지 그대로 반환한다.", "Return the unchanged input name including its suffix."},
		{"길이를 세지 않고 항상 0을 반환한다.", "Return zero without counting the length."},
	},
	"division": {
		{"계산된 유효 여부를 반전해 반환한다.", "Return the negation of the computed validity."},
		{"계산된 정수 몫의 부호를 반전한다.", "Negate the computed integer quotient."},
		{"계산된 나머지의 부호를 반전한다.", "Negate the computed remainder."},
	},
	"retry": {
		{"계산된 판단에 관계없이 재시도를 끈다.", "Disable retry regardless of the computed decision."},
		{"계산된 시간에 관계없이 대기 시간을 0으로 둔다.", "Set the delay to zero regardless of the computed delay."},
		{"분기 결과에 관계없이 이유를 pending으로 둔다.", "Set the reason to pending regardless of the branch result."},
	},
}

var valueCase = regexp.MustCompile(`(?m)^    value_case (` + quoted + `) -> (` + quoted + `)$`)

func contrastSource(original string, family familySpec, order int, language string, wanted uint16) (string, error) {
	if wanted > 7 {
		return "", fmt.Errorf("requested behavior mask must be in 0..7")
	}
	source, err := variant(original, family, order, language)
	if err != nil {
		return "", err
	}
	for bit, line := range declaration.FindAllStringSubmatch(source, -1) {
		if wanted&(1<<bit) != 0 {
			continue
		}
		words := firstIntents[family.name][bit]
		intent := words[0] + " " + words[1]
		if language == "ko" {
			intent = words[0]
		} else if language == "en" {
			intent = words[1]
		}
		next := fmt.Sprintf("    choice %s field_value at %s alternative %s intent %s", line[1], line[2], line[3], strconv.Quote(intent))
		source = strings.Replace(source, line[0], next, 1)
	}
	lines := valueCase.FindAllStringSubmatch(source, -1)
	if len(lines) == 0 {
		return "", fmt.Errorf("source-owned cases required")
	}
	for _, line := range lines {
		inputs, err := strconv.Unquote(line[1])
		if err != nil {
			return "", err
		}
		baseline, err := oracle(family.name, []byte(inputs), 7)
		if err != nil {
			return "", err
		}
		declared, _ := strconv.Unquote(line[2])
		if !sameJSON([]byte(declared), baseline) {
			return "", fmt.Errorf("independent oracle differs from original %s case", family.name)
		}
		want, err := oracle(family.name, []byte(inputs), wanted)
		if err != nil {
			return "", err
		}
		source = strings.Replace(source, line[0], "    value_case "+line[1]+" -> "+strconv.Quote(string(want)), 1)
	}
	return source, nil
}

func sameJSON(a, b []byte) bool {
	decode := func(raw []byte) any {
		var value any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if d.Decode(&value) != nil {
			return nil
		}
		var trailing any
		if d.Decode(&trailing) != io.EOF {
			return nil
		}
		return value
	}
	left, right := decode(a), decode(b)
	return left != nil && right != nil && reflect.DeepEqual(left, right)
}
