package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type choiceSpec struct{ id, field, first, second, korean, english string }
type familySpec struct {
	name, source, activity string
	choices                [3]choiceSpec
}

var families = []familySpec{
	{"filenames", "filenames.gooo", "Classify", [3]choiceSpec{
		{"source", "source", "suffix || visible", "suffix && visible", "확장자와 보이는 이름 조건을 모두 만족한다.", "Require both the suffix and visible-name conditions."},
		{"stem", "stem", "input", "stem", "접미사를 제거한 이름을 반환한다.", "Return the name after removing the suffix."},
		{"bytes", "bytes", "0", "bytes", "입력 이름의 UTF-8 바이트 길이를 반환한다.", "Return the UTF-8 byte length of the input name."},
	}},
	{"division", "division.gooo", "Divide", [3]choiceSpec{
		{"valid", "valid", "!valid", "valid", "분모가 0이 아닌지 계산한 결과를 유지한다.", "Keep the computed nonzero-divisor result."},
		{"quotient", "quotient", "-quotient", "quotient", "계산된 정수 몫의 부호와 값을 유지한다.", "Keep the sign and value of the computed integer quotient."},
		{"remainder", "remainder", "-remainder", "remainder", "계산된 나머지의 부호와 값을 유지한다.", "Keep the sign and value of the computed remainder."},
	}},
	{"retry", "retry.gooo", "PlanRetry", [3]choiceSpec{
		{"retry", "retry", "false", "again", "분기에서 계산한 재시도 여부를 반환한다.", "Return the retry decision computed by the branches."},
		{"delay", "delay_ms", "0", "delay", "상한 안에서 계산한 대기 시간을 반환한다.", "Return the computed delay within its cap."},
		{"reason", "reason", "\"pending\"", "reason", "분기에서 선택한 이유를 반환한다.", "Return the reason selected by the branches."},
	}},
}

const quoted = `"(?:[^"\\]|\\.)*"`

var declaration = regexp.MustCompile(`(?m)^    choice (` + quoted + `) field_value at (` + quoted + `) alternative (` + quoted + `) intent (` + quoted + `)$`)

func variant(source string, family familySpec, order int, language string) (string, error) {
	if order < 0 || order > 7 || language != "ko" && language != "en" && language != "mixed" {
		return "", fmt.Errorf("eight arrangements and ko/en/mixed intent forms required")
	}
	lines := declaration.FindAllStringSubmatch(source, -1)
	if len(lines) != 3 {
		return "", fmt.Errorf("three exact field-choice declarations required")
	}
	for bit, choice := range family.choices {
		line := lines[bit]
		if line[1] != strconv.Quote(choice.id) || line[2] != strconv.Quote(strconv.Itoa(bit)) || line[3] != strconv.Quote(choice.second) {
			return "", fmt.Errorf("declared choice %d differs from fixture specification", bit)
		}
		first, second := choice.first, choice.second
		if order&(1<<bit) != 0 {
			first, second = second, first
		}
		old := choice.field + ": " + choice.first
		if strings.Count(source, old) != 1 {
			return "", fmt.Errorf("field %s baseline is not unique", choice.field)
		}
		source = strings.Replace(source, old, choice.field+": "+first, 1)
		intent := choice.korean + " " + choice.english
		if language == "ko" {
			intent = choice.korean
		} else if language == "en" {
			intent = choice.english
		}
		next := fmt.Sprintf("    choice %s field_value at %s alternative %s intent %s",
			line[1], line[2], strconv.Quote(second), strconv.Quote(intent))
		source = strings.Replace(source, line[0], next, 1)
	}
	return source, nil
}
