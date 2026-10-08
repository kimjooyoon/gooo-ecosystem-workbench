package main

//gooo:generated:start id="gooo://retry/plan" kind="entity"
type GoooRecord86b4379d62c779cabd28430042ed7d7455d206ef6d264504b97bae47c042c540 struct {
	GoooField83cfa71ee008cc3c1c83354e83508e402bf8545ae3febd52e48fc11b6a2d996a bool   `json:"retry"`
	GoooFielde07e4c2b5e7c32cf62380eebfbd5b64b1588a8cb1a271a66659dd4cf0f704553 int64  `json:"delay_ms"`
	GoooField39bb485fe1a7cdfd57f06b8338d27f3c9d7470638f7aef01d0b2da49d00757df string `json:"reason"`
}

//gooo:generated:end id="gooo://retry/plan" kind="entity"

//gooo:generated:start id="retry://activity/plan-retry" kind="activity"
func GoooComposedActivity0(input0 bool, input1 bool, input2 int64, input3 int64, input4 int64, input5 int64) GoooRecord86b4379d62c779cabd28430042ed7d7455d206ef6d264504b97bae47c042c540 {
	var again = false
	var delay int64 = 0
	var reason = "invalid-input"
	if input2 >= 0 && input3 >= 0 && input4 >= 0 && input5 >= 0 {
		reason = "completed"
		if !input0 {
			reason = "permanent-failure"
			if input1 {
				reason = "attempt-limit"
				if input2 < input3 {
					again = true
					reason = "retry"
					delay = input5
					if input4 <= input5-input4 {
						delay = input4 * 2
					}
					if input4 == 0 && input5 > 0 {
						delay = 1
					}
				}
			}
		}
	}
	return GoooRecord86b4379d62c779cabd28430042ed7d7455d206ef6d264504b97bae47c042c540{GoooField83cfa71ee008cc3c1c83354e83508e402bf8545ae3febd52e48fc11b6a2d996a: (again), GoooFielde07e4c2b5e7c32cf62380eebfbd5b64b1588a8cb1a271a66659dd4cf0f704553: (delay), GoooField39bb485fe1a7cdfd57f06b8338d27f3c9d7470638f7aef01d0b2da49d00757df: (reason)}
}

//gooo:generated:end id="retry://activity/plan-retry" kind="activity"
