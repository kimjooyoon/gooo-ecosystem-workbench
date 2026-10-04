// Package workbench connects source-owned Gooo ecosystem recipes to the compiler.
package workbench

import "embed"

// Source rules and finite examples remain in Gooo and JSON, including in a built CLI.
//
//go:embed recipes/*.gooo recipes/*-cases.json models/shared-qat/*
var assets embed.FS
