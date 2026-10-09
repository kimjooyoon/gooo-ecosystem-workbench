package workbench

import (
	"encoding/json"
	"reflect"
	"testing"
)

func apiFactFixture() APIInterface {
	return APIInterface{Entry: APIEntry{PackagePath: "app", Activity: "Run"}, Packages: []APIPackage{
		{Path: "types", Declarations: []APIDeclaration{{Kind: "entity", Name: "Record", ID: "example://record", Shape: "record",
			Fields: []APIField{{Name: "count", ID: "example://count", TypeID: "urn:gooo:type:integer", Presence: "optional", Cardinality: "one"}}}}},
		{Path: "app", Imports: []string{"types"}, Declarations: []APIDeclaration{{Kind: "activity", Name: "Run", ID: "app://run",
			Inputs: []string{"example://record", "example://text", "example://record"}, Output: "example://result"}}},
	}}
}

func TestAPIFactsPreserveIDsAndInputOrder(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		change func(*APIInterface)
	}{
		{"field-type", func(v *APIInterface) { v.Packages[0].Declarations[0].Fields[0].TypeID = "urn:gooo:type:string" }},
		{"field-presence", func(v *APIInterface) { v.Packages[0].Declarations[0].Fields[0].Presence = "required" }},
		{"field-name", func(v *APIInterface) { v.Packages[0].Declarations[0].Fields[0].Name = "total" }},
		{"declaration-name", func(v *APIInterface) { v.Packages[0].Declarations[0].Name = "Renamed" }},
		{"inputs", func(v *APIInterface) {
			v.Packages[1].Declarations[0].Inputs = []string{"example://text", "example://record", "example://record"}
		}},
		{"shape", func(v *APIInterface) { v.Packages[0].Declarations[0].Shape = "nominal" }},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			a, b := apiFactFixture(), apiFactFixture()
			tc.change(&b)
			facts := apiChangeFacts(a, b)
			if len(facts) != 1 || facts[0].Kind != tc.kind || len(facts[0].Before) == 0 || len(facts[0].After) == 0 {
				t.Fatal(facts)
			}
			if tc.kind != "inputs" && (!facts[0].InputUsed || facts[0].OutputUsed || facts[0].DeclarationID != "example://record") {
				t.Fatal(facts)
			}
			if tc.kind == "field-presence" && (facts[0].BeforeRequired || !facts[0].AfterRequired) {
				t.Fatal(facts)
			}
		})
	}
}

func TestAPIFactsReportAddRemoveAndPackageScopedIdentity(t *testing.T) {
	a, b := apiFactFixture(), apiFactFixture()
	b.Packages[0].Declarations[0].Fields[0].ID = "example://new-count"
	facts := apiChangeFacts(a, b)
	if len(facts) != 2 || facts[0].Kind != "field-removed" || facts[1].Kind != "field-added" {
		t.Fatal(facts)
	}
	a, b = apiFactFixture(), apiFactFixture()
	duplicate := APIPackage{Path: "other", Declarations: []APIDeclaration{{Kind: "entity", Name: "Record", ID: "other://record", Shape: "nominal"}}}
	a.Packages = append(a.Packages, duplicate)
	b.Packages = append(b.Packages, duplicate)
	b.Packages[0].Declarations[0].Fields[0].Name = "new"
	facts = apiChangeFacts(a, b)
	if len(facts) != 1 || facts[0].Package != "types" {
		t.Fatal(facts)
	}
	before, _ := json.Marshal(a)
	_ = apiChangeFacts(a, b)
	after, _ := json.Marshal(a)
	if string(before) != string(after) {
		t.Fatal("comparison mutated its original input")
	}
}

func TestAPIFactsSeparateSourceEvidenceFromDeclarationChanges(t *testing.T) {
	a, b := apiFactFixture(), apiFactFixture()
	b.ImageDigest = "body changed"
	b.Packages[0].Sources = []APISource{{Filename: "moved.gooo", SourceDigest: "new body"}}
	b.Packages[0], b.Packages[1] = b.Packages[1], b.Packages[0]
	if facts := apiChangeFacts(a, b); len(facts) != 0 {
		t.Fatal("source evidence became a declaration change", facts)
	}
	a, b = apiFactFixture(), apiFactFixture()
	b.Packages[1].Declarations[0].Output = "example://record"
	b.Packages[0].Declarations[0].Fields[0].Presence = "required"
	for _, fact := range apiChangeFacts(a, b) {
		if fact.Kind == "field-presence" && (!fact.InputUsed || !fact.OutputUsed) {
			t.Fatal(fact)
		}
	}
	first := apiChangeFacts(a, b)
	second := apiChangeFacts(a, b)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("unstable comparison order")
	}
}

func TestAPIFactsKeepReorderingVisibleWhenFieldsAreAdded(t *testing.T) {
	a, b := apiFactFixture(), apiFactFixture()
	first := a.Packages[0].Declarations[0].Fields[0]
	second := first
	second.ID, second.Name = "example://other", "other"
	added := first
	added.ID, added.Name = "example://new", "new"
	a.Packages[0].Declarations[0].Fields = []APIField{first, second}
	for _, tc := range []struct {
		name      string
		fields    []APIField
		reordered bool
	}{
		{"insertion", []APIField{first, added, second}, false},
		{"insertion-and-reorder", []APIField{second, added, first}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b.Packages[0].Declarations[0].Fields = tc.fields
			facts := apiChangeFacts(a, b)
			found := false
			for _, fact := range facts {
				if fact.Kind == "field-order" {
					found = true
					if string(fact.Before) != `["example://count","example://other"]` || string(fact.After) != `["example://other","example://new","example://count"]` {
						t.Fatal("full original field sequences lost", fact)
					}
				}
			}
			if found != tc.reordered {
				t.Fatal("relative ordering of surviving fields lost", facts)
			}
		})
	}
}
