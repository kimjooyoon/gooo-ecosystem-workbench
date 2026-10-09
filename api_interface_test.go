package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
)

func signedAPIReceipt(t *testing.T, view APIInterface) []byte {
	t.Helper()
	view.Digest = ""
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	view.Digest = fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	raw, err = json.Marshal(map[string]any{"schema": "gooo/package-workspace-interface-receipt/v1", "decision": "PASS", "manifest": "fixture.json", "manifest_digest": "sha256:fixture", "interface": view})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validAPIInterface() APIInterface {
	view := apiFactFixture()
	view.Packages[0].Declarations = append(view.Packages[0].Declarations,
		APIDeclaration{Kind: "entity", Name: "Text", ID: "example://text", Shape: "nominal"},
		APIDeclaration{Kind: "entity", Name: "Result", ID: "example://result", Shape: "nominal"})
	view.Schema = "gooo/package-interface/v1"
	view.Scope = "DECLARED_PACKAGE_INTERFACES"
	view.ImageDigest = "sha256:image"
	for i := range view.Packages {
		pkg := &view.Packages[i]
		pkg.Name, pkg.Namespace = pkg.Path, pkg.Path
		pkg.Sources = []APISource{{Filename: "source.gooo", SourceDigest: "sha256:source", SemanticDigest: "sha256:semantic"}}
		for j := range pkg.Declarations {
			pkg.Declarations[j].Source = "source.gooo"
		}
	}
	return view
}

func TestAPIInterfaceRejectsIncompleteOrChangedProjection(t *testing.T) {
	raw := signedAPIReceipt(t, validAPIInterface())
	if _, err := readAPIInterface(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := readAPIInterface(append(raw, raw...)); err == nil {
		t.Fatal("trailing projection accepted")
	}
	for _, change := range []func(*APIInterface){
		func(v *APIInterface) { v.Schema = "unknown" },
		func(v *APIInterface) { v.Entry.Activity = "Missing" },
		func(v *APIInterface) { v.Packages[1].Declarations[0].Inputs[0] = "missing://input" },
		func(v *APIInterface) { v.Packages[0].Declarations[0].Fields[0].TypeID = "" },
		func(v *APIInterface) { v.Packages[0].Declarations[0].Fields[0].ID = v.Packages[0].Declarations[0].ID },
		func(v *APIInterface) { v.Packages[0].Declarations[0].Fields[0].Presence = "unknown" },
	} {
		v := validAPIInterface()
		change(&v)
		if _, err := readAPIInterface(signedAPIReceipt(t, v)); err == nil {
			t.Fatal("incomplete projection accepted", v)
		}
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	object["interface"].(map[string]any)["image_digest"] = "changed"
	changed, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = readAPIInterface(changed); err == nil {
		t.Fatal("altered interface digest accepted")
	}
}
