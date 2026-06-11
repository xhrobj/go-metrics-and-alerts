package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateBasicResetMethod(t *testing.T) {
	tempDir := t.TempDir()

	copyDir(t, filepath.Join("testdata", "basic", "input"), tempDir)

	generatedFiles, err := Generate(tempDir)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	gotGeneratedCount := len(generatedFiles)
	wantGeneratedCount := 1
	if gotGeneratedCount != wantGeneratedCount {
		t.Fatalf("len(generatedFiles) = %d, want %d", gotGeneratedCount, wantGeneratedCount)
	}

	gotGeneratedPath := filepath.Clean(generatedFiles[0])
	wantGeneratedPath := filepath.Clean(filepath.Join(tempDir, "sample", generatedFile))
	if gotGeneratedPath != wantGeneratedPath {
		t.Fatalf("generated file path = %q, want %q", gotGeneratedPath, wantGeneratedPath)
	}

	gotPath := filepath.Join(tempDir, "sample", generatedFile)
	wantPath := filepath.Join("testdata", "basic", "want", "sample", generatedFile)

	got, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", gotPath, err)
	}

	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", wantPath, err)
	}

	if string(got) != string(want) {
		t.Fatalf("generated file mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0644)
	})

	if err != nil {
		t.Fatalf("copyDir(%q, %q) error = %v", src, dst, err)
	}
}
