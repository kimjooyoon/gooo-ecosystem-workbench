package main

//gooo:generated:start id="gooo://starter/project" kind="entity"
type GoooRecord2fb13f89f6312dcdb30eab3d8c95616a6238b5eae01737d285c12df4ad4c310b struct {
	GoooFielde54d871d58bd98d8a0476fcd5e0684d2638c56ecf6e64136fd94eb1748a24b4d string `json:"filename"`
	GoooField2cf9088b98b68456d40f435660aeaaa4d92f1dcd0b73f1b3fb464dfaedd2b5f3 string `json:"source"`
	GoooField427259b74f9d75ef2bbfa0ce311803e73088408750734bb5c1d43d897cc48751 string `json:"next"`
}

//gooo:generated:end id="gooo://starter/project" kind="entity"

//gooo:generated:start id="starterplanner://activity/plan" kind="activity"
func GoooComposedActivity0(input string) GoooRecord2fb13f89f6312dcdb30eab3d8c95616a6238b5eae01737d285c12df4ad4c310b {
	var scalarSource = "package starter\nnamespace example\nentity Integer id \"example://integer\"\nactivity Identity(Integer) -> Integer computes \x60return input\x60\n"
	var recordSource = "package starter\nnamespace example\nentity Item id \"example://item\" fields {\n    field title id \"example://item/title\" type string required one\n}\nactivity Identity(Item) -> Item computes \x60return input\x60\n"
	if input == "record" && recordSource != "" {
		return GoooRecord2fb13f89f6312dcdb30eab3d8c95616a6238b5eae01737d285c12df4ad4c310b{GoooFielde54d871d58bd98d8a0476fcd5e0684d2638c56ecf6e64136fd94eb1748a24b4d: ("main.gooo"), GoooField2cf9088b98b68456d40f435660aeaaa4d92f1dcd0b73f1b3fb464dfaedd2b5f3: (recordSource), GoooField427259b74f9d75ef2bbfa0ce311803e73088408750734bb5c1d43d897cc48751: ("gooo body-codegen --activity Identity main.gooo")}
	}
	if input == "scalar" {
		return GoooRecord2fb13f89f6312dcdb30eab3d8c95616a6238b5eae01737d285c12df4ad4c310b{GoooFielde54d871d58bd98d8a0476fcd5e0684d2638c56ecf6e64136fd94eb1748a24b4d: "main.gooo", GoooField2cf9088b98b68456d40f435660aeaaa4d92f1dcd0b73f1b3fb464dfaedd2b5f3: scalarSource, GoooField427259b74f9d75ef2bbfa0ce311803e73088408750734bb5c1d43d897cc48751: "gooo body-codegen --activity Identity main.gooo"}
	}
	return GoooRecord2fb13f89f6312dcdb30eab3d8c95616a6238b5eae01737d285c12df4ad4c310b{GoooFielde54d871d58bd98d8a0476fcd5e0684d2638c56ecf6e64136fd94eb1748a24b4d: "", GoooField2cf9088b98b68456d40f435660aeaaa4d92f1dcd0b73f1b3fb464dfaedd2b5f3: "", GoooField427259b74f9d75ef2bbfa0ce311803e73088408750734bb5c1d43d897cc48751: "Choose scalar or record."}
}

//gooo:generated:end id="starterplanner://activity/plan" kind="activity"

//gooo:generated:start id="starterplanner://activity/preview" kind="activity"
func GoooComposedActivity1(input GoooRecord2fb13f89f6312dcdb30eab3d8c95616a6238b5eae01737d285c12df4ad4c310b) string {
	return input.GoooField2cf9088b98b68456d40f435660aeaaa4d92f1dcd0b73f1b3fb464dfaedd2b5f3
}

//gooo:generated:end id="starterplanner://activity/preview" kind="activity"
