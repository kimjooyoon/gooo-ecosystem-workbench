package workbench

import (
	"encoding/json"
	"reflect"
	"sort"
)

type APIChange struct {
	Package        string          `json:"package"`
	DeclarationID  string          `json:"declaration_id,omitempty"`
	FieldID        string          `json:"field_id,omitempty"`
	Kind           string          `json:"kind"`
	Before         json.RawMessage `json:"before,omitempty"`
	After          json.RawMessage `json:"after,omitempty"`
	InputUsed      bool            `json:"input_used"`
	OutputUsed     bool            `json:"output_used"`
	BeforeRequired bool            `json:"before_required"`
	AfterRequired  bool            `json:"after_required"`
	Assessment     *APIAssessment  `json:"assessment,omitempty"`
}
type APIAssessment struct {
	Code     string `json:"code"`
	Action   string `json:"action"`
	Priority int64  `json:"priority"`
}
type apiUsage struct{ input, output bool }

func apiChangeFacts(before, after APIInterface) []APIChange {
	usage := map[string]apiUsage{}
	for _, view := range []APIInterface{before, after} {
		for _, pkg := range view.Packages {
			for _, decl := range pkg.Declarations {
				if decl.Kind != "activity" {
					continue
				}
				for _, id := range decl.Inputs {
					u := usage[id]
					u.input = true
					usage[id] = u
				}
				u := usage[decl.Output]
				u.output = true
				usage[decl.Output] = u
			}
		}
	}
	var changes []APIChange
	add := func(base APIChange, kind string, old, next any) {
		if reflect.DeepEqual(old, next) {
			return
		}
		base.Kind = kind
		if old != nil {
			base.Before, _ = json.Marshal(old)
		}
		if next != nil {
			base.After, _ = json.Marshal(next)
		}
		changes = append(changes, base)
	}
	add(APIChange{}, "entry", before.Entry, after.Entry)
	a, b := map[string]APIPackage{}, map[string]APIPackage{}
	for _, pkg := range before.Packages {
		a[pkg.Path] = pkg
	}
	for _, pkg := range after.Packages {
		b[pkg.Path] = pkg
	}
	for _, path := range unionKeys(a, b) {
		old, had := a[path]
		next, has := b[path]
		base := APIChange{Package: path}
		if !had {
			add(base, "package-added", nil, next)
		}
		if !has {
			add(base, "package-removed", old, nil)
		}
		if had && has {
			add(base, "package-name", old.Name, next.Name)
			add(base, "package-namespace", old.Namespace, next.Namespace)
			add(base, "imports", old.Imports, next.Imports)
		}
		left, right := map[string]APIDeclaration{}, map[string]APIDeclaration{}
		for _, decl := range old.Declarations {
			left[decl.ID] = decl
		}
		for _, decl := range next.Declarations {
			right[decl.ID] = decl
		}
		for _, id := range unionKeys(left, right) {
			prev, existed := left[id]
			current, exists := right[id]
			u := usage[id]
			item := APIChange{Package: path, DeclarationID: id, InputUsed: u.input, OutputUsed: u.output}
			if !existed {
				add(item, "declaration-added", nil, current)
				continue
			}
			if !exists {
				add(item, "declaration-removed", prev, nil)
				continue
			}
			add(item, "declaration-kind", prev.Kind, current.Kind)
			add(item, "declaration-name", prev.Name, current.Name)
			add(item, "shape", prev.Shape, current.Shape)
			add(item, "inputs", prev.Inputs, current.Inputs)
			add(item, "output", prev.Output, current.Output)
			apiFieldChanges(item, prev.Fields, current.Fields, add)
		}
	}
	return changes
}

func apiFieldChanges(base APIChange, before, after []APIField, add func(APIChange, string, any, any)) {
	a, b := map[string]APIField{}, map[string]APIField{}
	oldOrder, newOrder := make([]string, 0, len(before)), make([]string, 0, len(after))
	for _, field := range before {
		a[field.ID] = field
		oldOrder = append(oldOrder, field.ID)
	}
	for _, field := range after {
		b[field.ID] = field
		newOrder = append(newOrder, field.ID)
	}
	keys := unionKeys(a, b)
	// Compare surviving identities so an addition/removal cannot hide a reorder.
	oldCommon, newCommon := []string{}, []string{}
	for _, id := range oldOrder {
		if _, ok := b[id]; ok {
			oldCommon = append(oldCommon, id)
		}
	}
	for _, id := range newOrder {
		if _, ok := a[id]; ok {
			newCommon = append(newCommon, id)
		}
	}
	if !reflect.DeepEqual(oldCommon, newCommon) {
		add(base, "field-order", oldOrder, newOrder)
	}
	for _, id := range keys {
		old, had := a[id]
		next, has := b[id]
		item := base
		item.FieldID, item.BeforeRequired, item.AfterRequired = id, old.Presence == "required", next.Presence == "required"
		if !had {
			add(item, "field-added", nil, next)
			continue
		}
		if !has {
			add(item, "field-removed", old, nil)
			continue
		}
		add(item, "field-name", old.Name, next.Name)
		add(item, "field-type", old.TypeID, next.TypeID)
		add(item, "field-presence", old.Presence, next.Presence)
		add(item, "field-cardinality", old.Cardinality, next.Cardinality)
		add(item, "field-aliases", old.Aliases, next.Aliases)
	}
}

func unionKeys[T any](a, b map[string]T) []string {
	keys := make([]string, 0, len(a)+len(b))
	for key := range a {
		keys = append(keys, key)
	}
	for key := range b {
		if _, ok := a[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
