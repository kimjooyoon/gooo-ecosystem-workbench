package main

//gooo:generated:start id="gooo://featureaudit/assessment" kind="entity"
type GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3 struct {
	GoooFieldf8db1e56cb84e7ab4dfda053be150b28c12c8770d222c48939ea406339f1cbf2 string `json:"code"`
	GoooField747c06003837b293651558ab85d7cb0979484565b9543d3c2e1687a86ae0483f string `json:"action"`
	GoooField7ae3bfea11690cf451d3c99258bb077588f441f0dca0d5c470f60c7264f86d57 string `json:"message"`
}

//gooo:generated:end id="gooo://featureaudit/assessment" kind="entity"
//gooo:generated:start id="gooo://featureaudit/observation" kind="entity"
type GoooRecordc1442f50e9f9e3b990518226f73cc3c326f8830b25c9b0d21eefc14c162c2f5f struct {
	GoooFieldea1b852fe7d682249af0c9429693722246dfbdeed2c5c26eb7c58ac1cfb2bced int64 `json:"rows"`
	GoooField0f926c7a174901cb9670db1adb3d0fc2d6c1d0f98cc5ea4b5161f058ce7076f8 int64 `json:"maximum"`
	GoooField753e6008586b59e07fca935e02a4282c5116dc00e707c7e84926217852e929ed bool  `json:"model_observed"`
	GoooFieldcfdf0701ac99eeae494a20224ca0edf2275dd0cd86b8af048c7970f821f0bd69 int64 `json:"model_matched"`
}

//gooo:generated:end id="gooo://featureaudit/observation" kind="entity"

//gooo:generated:start id="featureaudit://activity/observation-echo" kind="activity"
func GoooComposedActivity0(input GoooRecordc1442f50e9f9e3b990518226f73cc3c326f8830b25c9b0d21eefc14c162c2f5f) GoooRecordc1442f50e9f9e3b990518226f73cc3c326f8830b25c9b0d21eefc14c162c2f5f {
	return input
}

//gooo:generated:end id="featureaudit://activity/observation-echo" kind="activity"

//gooo:generated:start id="featureaudit://activity/assess" kind="activity"
func GoooComposedActivity1(input GoooRecordc1442f50e9f9e3b990518226f73cc3c326f8830b25c9b0d21eefc14c162c2f5f) GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3 {
	if input.GoooFieldea1b852fe7d682249af0c9429693722246dfbdeed2c5c26eb7c58ac1cfb2bced < 1 || input.GoooField0f926c7a174901cb9670db1adb3d0fc2d6c1d0f98cc5ea4b5161f058ce7076f8 < 1 || input.GoooField0f926c7a174901cb9670db1adb3d0fc2d6c1d0f98cc5ea4b5161f058ce7076f8 > input.GoooFieldea1b852fe7d682249af0c9429693722246dfbdeed2c5c26eb7c58ac1cfb2bced || input.GoooFieldcfdf0701ac99eeae494a20224ca0edf2275dd0cd86b8af048c7970f821f0bd69 < 0 || input.GoooFieldcfdf0701ac99eeae494a20224ca0edf2275dd0cd86b8af048c7970f821f0bd69 > input.GoooField0f926c7a174901cb9670db1adb3d0fc2d6c1d0f98cc5ea4b5161f058ce7076f8 {
		return GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3{GoooFieldf8db1e56cb84e7ab4dfda053be150b28c12c8770d222c48939ea406339f1cbf2: "invalid-counts", GoooField747c06003837b293651558ab85d7cb0979484565b9543d3c2e1687a86ae0483f: "recount-inputs", GoooField7ae3bfea11690cf451d3c99258bb077588f441f0dca0d5c470f60c7264f86d57: "입력과 관측 수를 다시 확인한다."}
	}
	if input.GoooField0f926c7a174901cb9670db1adb3d0fc2d6c1d0f98cc5ea4b5161f058ce7076f8 < input.GoooFieldea1b852fe7d682249af0c9429693722246dfbdeed2c5c26eb7c58ac1cfb2bced {
		return GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3{GoooFieldf8db1e56cb84e7ab4dfda053be150b28c12c8770d222c48939ea406339f1cbf2: "representation-collision", GoooField747c06003837b293651558ab85d7cb0979484565b9543d3c2e1687a86ae0483f: "preserve-distinguishing-source-facts", GoooField7ae3bfea11690cf451d3c99258bb077588f441f0dca0d5c470f60c7264f86d57: "같은 모델 입력에 서로 다른 선택이 필요하다. 구분에 필요한 연산과 값의 출처를 입력에 보존한다."}
	}
	if !input.GoooField753e6008586b59e07fca935e02a4282c5116dc00e707c7e84926217852e929ed {
		return GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3{GoooFieldf8db1e56cb84e7ab4dfda053be150b28c12c8770d222c48939ea406339f1cbf2: "input-consistent", GoooField747c06003837b293651558ab85d7cb0979484565b9543d3c2e1687a86ae0483f: "evaluate-chooser", GoooField7ae3bfea11690cf451d3c99258bb077588f441f0dca0d5c470f60c7264f86d57: "제공된 자료에서 충돌을 찾지 못했다. 후보 판단을 별도로 측정한다."}
	}
	if input.GoooFieldcfdf0701ac99eeae494a20224ca0edf2275dd0cd86b8af048c7970f821f0bd69 < input.GoooFieldea1b852fe7d682249af0c9429693722246dfbdeed2c5c26eb7c58ac1cfb2bced {
		return GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3{GoooFieldf8db1e56cb84e7ab4dfda053be150b28c12c8770d222c48939ea406339f1cbf2: "chooser-gap", GoooField747c06003837b293651558ab85d7cb0979484565b9543d3c2e1687a86ae0483f: "balance-candidate-training", GoooField7ae3bfea11690cf451d3c99258bb077588f441f0dca0d5c470f60c7264f86d57: "입력에서 구분할 수 있는 사례가 남았다. 후보 배치와 원본 프로그램별 학습 분할을 점검한다."}
	}
	return GoooRecord370c68608178e184e121eacf16513ae336a479952789c3147bf2f2f5a97c49f3{GoooFieldf8db1e56cb84e7ab4dfda053be150b28c12c8770d222c48939ea406339f1cbf2: "observed-fit", GoooField747c06003837b293651558ab85d7cb0979484565b9543d3c2e1687a86ae0483f: "evaluate-separated-programs", GoooField7ae3bfea11690cf451d3c99258bb077588f441f0dca0d5c470f60c7264f86d57: "제공된 선택을 만족했다. 별도 프로그램에서 다음 관측을 진행한다."}
}

//gooo:generated:end id="featureaudit://activity/assess" kind="activity"
