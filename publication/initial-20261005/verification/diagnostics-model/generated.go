package main

//gooo:generated:start id="gooo://diagnostics/diagnostic" kind="entity"
type GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568 struct {
	GoooFielda5ddfea237543761a6d8fa7e3953351a4c28b9f8b562b535985914c71ab1cefa string `json:"code"`
	GoooField4b4172b122bef7e6457a0e388c589f61e39f6835d2d4264d5eb70f1879e46023 string `json:"message"`
	GoooField0d545531b9bc128be38cece1f5b3d5aed0ab98365a92e6b1ef70e1689bf62135 string `json:"action"`
}

//gooo:generated:end id="gooo://diagnostics/diagnostic" kind="entity"

//gooo:generated:start id="diagnostics://activity/diagnose" kind="activity"
func GoooComposedActivity0(input0 int64, input1 int64, input2 int64, input3 string) GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568 {
	var action = "expand-examples"
	if input2 > 0 {
		action = "repair-and-replay"
	}
	if input1 > 0 && input0 >= 0 && input0 < input1 && input2 >= 0 {
		return GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568{GoooFielda5ddfea237543761a6d8fa7e3953351a4c28b9f8b562b535985914c71ab1cefa: ("partial"), GoooField4b4172b122bef7e6457a0e388c589f61e39f6835d2d4264d5eb70f1879e46023: (input3), GoooField0d545531b9bc128be38cece1f5b3d5aed0ab98365a92e6b1ef70e1689bf62135: action}
	}
	if input0 < 0 || input1 < 0 || input2 < 0 || input0 > input1 {
		action = "fix-counts"
		return GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568{GoooFielda5ddfea237543761a6d8fa7e3953351a4c28b9f8b562b535985914c71ab1cefa: "invalid-counts", GoooField4b4172b122bef7e6457a0e388c589f61e39f6835d2d4264d5eb70f1879e46023: input3, GoooField0d545531b9bc128be38cece1f5b3d5aed0ab98365a92e6b1ef70e1689bf62135: action}
	}
	if input1 == 0 {
		return GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568{GoooFielda5ddfea237543761a6d8fa7e3953351a4c28b9f8b562b535985914c71ab1cefa: "unobserved", GoooField4b4172b122bef7e6457a0e388c589f61e39f6835d2d4264d5eb70f1879e46023: "Add expected observations.", GoooField0d545531b9bc128be38cece1f5b3d5aed0ab98365a92e6b1ef70e1689bf62135: "add-examples"}
	}
	return GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568{GoooFielda5ddfea237543761a6d8fa7e3953351a4c28b9f8b562b535985914c71ab1cefa: "complete", GoooField4b4172b122bef7e6457a0e388c589f61e39f6835d2d4264d5eb70f1879e46023: "All observed fields matched.", GoooField0d545531b9bc128be38cece1f5b3d5aed0ab98365a92e6b1ef70e1689bf62135: "accept"}
}

//gooo:generated:end id="diagnostics://activity/diagnose" kind="activity"

//gooo:generated:start id="diagnostics://activity/explain" kind="activity"
func GoooComposedActivity1(input GoooRecord9d359c656161a57d988aded27dde9bb7bd780cc336ca66699797b584eab48568) string {
	return input.GoooFielda5ddfea237543761a6d8fa7e3953351a4c28b9f8b562b535985914c71ab1cefa + ": " + input.GoooField4b4172b122bef7e6457a0e388c589f61e39f6835d2d4264d5eb70f1879e46023 + " [" + input.GoooField0d545531b9bc128be38cece1f5b3d5aed0ab98365a92e6b1ef70e1689bf62135 + "]"
}

//gooo:generated:end id="diagnostics://activity/explain" kind="activity"

//gooo:generated:start id="diagnostics://activity/observation-echo" kind="activity"
func GoooComposedActivity2(input string) string {
	return input
}

//gooo:generated:end id="diagnostics://activity/observation-echo" kind="activity"
