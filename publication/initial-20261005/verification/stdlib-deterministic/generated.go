package main

//gooo:generated:start id="gooolib://activity/abs-saturating" kind="activity"
func GoooComposedActivity0(input int64) int64 {
	if input == -9223372036854775808 {
		return 9223372036854775807
	}
	if input < 0 {
		return -input
	}
	return input
}

//gooo:generated:end id="gooolib://activity/abs-saturating" kind="activity"

//gooo:generated:start id="gooolib://activity/and" kind="activity"
func GoooComposedActivity1(input0 bool, input1 bool) bool {
	return input0 && input1
}

//gooo:generated:end id="gooolib://activity/and" kind="activity"

//gooo:generated:start id="gooolib://activity/choose-text" kind="activity"
func GoooComposedActivity2(input0 string, input1 string, input2 bool) string {
	if input2 {
		return input0
	}
	return input1
}

//gooo:generated:end id="gooolib://activity/choose-text" kind="activity"

//gooo:generated:start id="gooolib://activity/clamp" kind="activity"
func GoooComposedActivity3(input0 int64, input1 int64, input2 int64) int64 {
	var lower = input1
	var upper = input2
	if lower > upper {
		lower = input2
		upper = input1
	}
	if input0 < lower {
		return lower
	}
	if input0 > upper {
		return upper
	}
	return input0
}

//gooo:generated:end id="gooolib://activity/clamp" kind="activity"

//gooo:generated:start id="gooolib://activity/coalesce-text" kind="activity"
func GoooComposedActivity4(input0 string, input1 string) string {
	if input0 != "" {
		return input0
	}
	return input1
}

//gooo:generated:end id="gooolib://activity/coalesce-text" kind="activity"

//gooo:generated:start id="gooolib://activity/in-range" kind="activity"
func GoooComposedActivity5(input0 int64, input1 int64, input2 int64) bool {
	var lower = input1
	var upper = input2
	if lower > upper {
		lower = input2
		upper = input1
	}
	return input0 >= lower && input0 <= upper
}

//gooo:generated:end id="gooolib://activity/in-range" kind="activity"

//gooo:generated:start id="gooolib://activity/is-zero" kind="activity"
func GoooComposedActivity6(input int64) bool {
	return input == 0
}

//gooo:generated:end id="gooolib://activity/is-zero" kind="activity"

//gooo:generated:start id="gooolib://activity/max" kind="activity"
func GoooComposedActivity7(input0 int64, input1 int64) int64 {
	if input0 > input1 {
		return input0
	}
	return input1
}

//gooo:generated:end id="gooolib://activity/max" kind="activity"

//gooo:generated:start id="gooolib://activity/min" kind="activity"
func GoooComposedActivity8(input0 int64, input1 int64) int64 {
	if input0 < input1 {
		return input0
	}
	return input1
}

//gooo:generated:end id="gooolib://activity/min" kind="activity"

//gooo:generated:start id="gooolib://activity/non-negative" kind="activity"
func GoooComposedActivity9(input int64) bool {
	return input >= 0
}

//gooo:generated:end id="gooolib://activity/non-negative" kind="activity"

//gooo:generated:start id="gooolib://activity/not" kind="activity"
func GoooComposedActivity10(input bool) bool {
	return !input
}

//gooo:generated:end id="gooolib://activity/not" kind="activity"

//gooo:generated:start id="gooolib://activity/or" kind="activity"
func GoooComposedActivity11(input0 bool, input1 bool) bool {
	return input0 || input1
}

//gooo:generated:end id="gooolib://activity/or" kind="activity"

//gooo:generated:start id="gooolib://activity/sign" kind="activity"
func GoooComposedActivity12(input int64) int64 {
	if input < 0 {
		return -1
	}
	if input > 0 {
		return 1
	}
	return 0
}

//gooo:generated:end id="gooolib://activity/sign" kind="activity"
