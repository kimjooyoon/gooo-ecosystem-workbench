package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
)

// These types mirror the compiler's versioned declaration projection. They do
// not parse Gooo or infer type structure from names.
type APIInterface struct {
	Schema      string       `json:"schema"`
	Scope       string       `json:"scope"`
	Entry       APIEntry     `json:"entry"`
	Packages    []APIPackage `json:"packages"`
	ImageDigest string       `json:"image_digest"`
	Digest      string       `json:"digest"`
}
type APIEntry struct {
	PackagePath string `json:"package_path"`
	Activity    string `json:"activity"`
}
type APIPackage struct {
	Path         string           `json:"path"`
	Name         string           `json:"name"`
	Namespace    string           `json:"namespace"`
	Imports      []string         `json:"imports"`
	Sources      []APISource      `json:"sources"`
	Declarations []APIDeclaration `json:"declarations"`
}
type APISource struct {
	Filename       string `json:"filename"`
	SourceDigest   string `json:"source_digest"`
	SemanticDigest string `json:"semantic_digest"`
	Declarations   int    `json:"declarations"`
}
type APIDeclaration struct {
	Kind   string     `json:"kind"`
	Name   string     `json:"name"`
	ID     string     `json:"id"`
	Source string     `json:"source"`
	Shape  string     `json:"shape,omitempty"`
	Fields []APIField `json:"fields,omitempty"`
	Inputs []string   `json:"inputs,omitempty"`
	Output string     `json:"output,omitempty"`
}
type APIField struct {
	Name        string   `json:"name"`
	ID          string   `json:"id"`
	Aliases     []string `json:"aliases,omitempty"`
	TypeID      string   `json:"type_id"`
	Presence    string   `json:"presence"`
	Cardinality string   `json:"cardinality"`
}

func readAPIInterface(raw []byte) (APIInterface, error) {
	var receipt struct {
		Schema         string        `json:"schema"`
		Decision       string        `json:"decision"`
		Manifest       string        `json:"manifest"`
		ManifestDigest string        `json:"manifest_digest"`
		Interface      *APIInterface `json:"interface"`
		Error          string        `json:"error,omitempty"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&receipt); err != nil {
		return APIInterface{}, err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return APIInterface{}, fmt.Errorf("one compiler interface receipt required")
	}
	if receipt.Schema != "gooo/package-workspace-interface-receipt/v1" || receipt.Decision != "PASS" ||
		receipt.Error != "" || receipt.ManifestDigest == "" || receipt.Interface == nil {
		return APIInterface{}, fmt.Errorf("complete compiler package interface required")
	}
	view := *receipt.Interface
	if view.Schema != "gooo/package-interface/v1" || view.Scope != "DECLARED_PACKAGE_INTERFACES" ||
		view.ImageDigest == "" || len(view.Packages) == 0 {
		return APIInterface{}, fmt.Errorf("unsupported package interface")
	}
	digest := view.Digest
	view.Digest = ""
	data, err := json.Marshal(view)
	if err != nil {
		return APIInterface{}, err
	}
	if digest != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
		return APIInterface{}, fmt.Errorf("package interface digest differs")
	}
	view.Digest = digest
	return view, validateAPIInterface(view)
}

func validateAPIInterface(view APIInterface) error {
	paths := map[string]bool{}
	entities := map[string]bool{}
	for _, pkg := range view.Packages {
		for _, declaration := range pkg.Declarations {
			if declaration.Kind == "entity" {
				entities[declaration.ID] = true
			}
		}
	}
	entry := false
	for _, pkg := range view.Packages {
		if pkg.Path == "" || paths[pkg.Path] || pkg.Name == "" || pkg.Namespace == "" || len(pkg.Sources) == 0 {
			return fmt.Errorf("incomplete or repeated package identity")
		}
		paths[pkg.Path] = true
		ids := map[string]bool{}
		for _, declaration := range pkg.Declarations {
			if declaration.ID == "" || ids[declaration.ID] || declaration.Name == "" || declaration.Source == "" {
				return fmt.Errorf("incomplete or repeated declaration identity")
			}
			ids[declaration.ID] = true
			if declaration.Kind == "activity" {
				if !entities[declaration.Output] {
					return fmt.Errorf("activity output identity required")
				}
				for _, id := range declaration.Inputs {
					if !entities[id] {
						return fmt.Errorf("activity input identity is unresolved")
					}
				}
				if pkg.Path == view.Entry.PackagePath && declaration.Name == view.Entry.Activity {
					entry = true
				}
			} else if declaration.Kind != "entity" || (declaration.Shape != "record" && declaration.Shape != "nominal") {
				return fmt.Errorf("unknown declaration kind or shape")
			}
		}
		for _, declaration := range pkg.Declarations {
			for _, field := range declaration.Fields {
				if field.ID == "" || ids[field.ID] || field.Name == "" || field.TypeID == "" ||
					(field.Presence != "required" && field.Presence != "optional") ||
					(field.Cardinality != "one" && field.Cardinality != "many") {
					return fmt.Errorf("incomplete field contract")
				}
				ids[field.ID] = true
			}
		}
	}
	if !entry {
		return fmt.Errorf("interface entry is missing")
	}
	return nil
}
