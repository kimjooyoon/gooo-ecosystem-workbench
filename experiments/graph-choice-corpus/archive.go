package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Sorted relative names, fixed metadata and a zero-time gzip header make a
// compact publication of exactly the observations, independent of host paths.
func archiveEvidence(root string) (err error) {
	file, err := os.Create(filepath.Join(root, "raw-evidence.tar.gz"))
	if err != nil {
		return err
	}
	defer func() {
		if e := file.Close(); err == nil {
			err = e
		}
	}()
	gz := gzip.NewWriter(file)
	defer func() {
		if e := gz.Close(); err == nil {
			err = e
		}
	}()
	tw := tar.NewWriter(gz)
	defer func() {
		if e := tw.Close(); err == nil {
			err = e
		}
	}()
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == "raw-evidence.tar.gz" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.Open(path)
		if err != nil {
			return err
		}
		defer raw.Close()
		info, err := raw.Stat()
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(relative), Size: info.Size(), Mode: 0644, ModTime: time.Unix(0, 0)}); err != nil {
			return err
		}
		_, err = io.Copy(tw, raw)
		return err
	})
}
