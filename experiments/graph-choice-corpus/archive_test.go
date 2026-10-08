package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEvidenceArchiveIgnoresHostTimesAndItsPreviousArchive(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "source.gooo")
	if err := os.WriteFile(name, []byte("source observation"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := archiveEvidence(root); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(root, "raw-evidence.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(name, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	if err := archiveEvidence(root); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(root, "raw-evidence.tar.gz"))
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("archive metadata is not deterministic", err)
	}
}
