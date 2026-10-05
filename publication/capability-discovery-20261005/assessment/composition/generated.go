package main

//gooo:generated:start id="gooo://workbench/capability-assessment/assessment" kind="entity"
type GoooRecordec19a4175b927832d03badc3b3e40359814b779c645f95c7f0136e2b4e42749d struct {
	GoooFieldbec5fca68d37c66506612f11a7484833bcbb111205f6d407305da38306890027 string `json:"catalog_state"`
	GoooField82f1af0f45915bf3ea3ca47058d6c9567604c70884b1c2ff5a82f6576d98fddb string `json:"first_unresolved_stage"`
	GoooField3fa41a58f4b5b244e3cd0df8a9932bed6a029cc56fd6601f52942fc119cb9e9f string `json:"intent"`
	GoooFieldbae41678438b34c1e48131e888a1195a19cde7c081801c3f900d7f91493f056a string `json:"next_operation"`
	GoooFielda73dd6b48c36f632e6abe9e1569c8a9e6455cd524ec217f564ea546a8ceab3e0 string `json:"real_use_case_coverage"`
	GoooFieldafee8ad39a01e6befd2e38f8364c20475cdc9a6a5caf2be17413db9d0220f06a string `json:"query_digest"`
	GoooField80c2da3529adf06b49338f273be33dbe4f16a6bcb578a5e9826a4510b77cdead string `json:"declaration_source_digest"`
	GoooFieldacdf8136296812d86a7cc34ef323c9441734e1088a9a5cc8d210d987e378bc91 string `json:"declaration_bound"`
	GoooFieldc8b816108ba3edd468ab8f1ceef4af9c8426b866ffefebd831c052893fa801f7 string `json:"execution_attempted"`
	GoooField99351fe637e6a11d116b2b5b15dbc556bbb0b378318462acd9dd363b931f28e5 string `json:"provider_invocations"`
	GoooField2972dab22d2a54ab58c2185541333a2c1e06b2d477fc89d756842bbe77795ea1 string `json:"reason"`
}

//gooo:generated:end id="gooo://workbench/capability-assessment/assessment" kind="entity"

//gooo:generated:start id="workbench-capability-assessment://activity/assess-capability" kind="activity"
func GoooComposedActivity0(input0 string, input1 string, input2 string, input3 string, input4 string, input5 string) GoooRecordec19a4175b927832d03badc3b3e40359814b779c645f95c7f0136e2b4e42749d {
	var coverage = "UNKNOWN"
	var unresolved = "capability_catalog"
	if input0 == "DEFERRED" {
		unresolved = "external_boundary"
	}
	if input0 == "AVAILABLE" && input2 == "false" {
		unresolved = "declaration_binding"
	}
	if input0 == "AVAILABLE" && input2 == "true" {
		coverage = "PROGRESS"
		unresolved = "generation"
	}
	return GoooRecordec19a4175b927832d03badc3b3e40359814b779c645f95c7f0136e2b4e42749d{GoooFieldbec5fca68d37c66506612f11a7484833bcbb111205f6d407305da38306890027: input0, GoooField82f1af0f45915bf3ea3ca47058d6c9567604c70884b1c2ff5a82f6576d98fddb: unresolved, GoooField3fa41a58f4b5b244e3cd0df8a9932bed6a029cc56fd6601f52942fc119cb9e9f: input1, GoooFieldbae41678438b34c1e48131e888a1195a19cde7c081801c3f900d7f91493f056a: input3, GoooFielda73dd6b48c36f632e6abe9e1569c8a9e6455cd524ec217f564ea546a8ceab3e0: coverage, GoooFieldafee8ad39a01e6befd2e38f8364c20475cdc9a6a5caf2be17413db9d0220f06a: input4, GoooField80c2da3529adf06b49338f273be33dbe4f16a6bcb578a5e9826a4510b77cdead: input5, GoooFieldacdf8136296812d86a7cc34ef323c9441734e1088a9a5cc8d210d987e378bc91: input2, GoooFieldc8b816108ba3edd468ab8f1ceef4af9c8426b866ffefebd831c052893fa801f7: "false", GoooField99351fe637e6a11d116b2b5b15dbc556bbb0b378318462acd9dd363b931f28e5: "0", GoooField2972dab22d2a54ab58c2185541333a2c1e06b2d477fc89d756842bbe77795ea1: "catalog discovery is not generation, reverse observation, or execution evidence"}
}

//gooo:generated:end id="workbench-capability-assessment://activity/assess-capability" kind="activity"

//gooo:generated:start id="workbench-capability-assessment://activity/preserve-assessment" kind="activity"
func GoooComposedActivity1(input GoooRecordec19a4175b927832d03badc3b3e40359814b779c645f95c7f0136e2b4e42749d) GoooRecordec19a4175b927832d03badc3b3e40359814b779c645f95c7f0136e2b4e42749d {
	return input
}

//gooo:generated:end id="workbench-capability-assessment://activity/preserve-assessment" kind="activity"
