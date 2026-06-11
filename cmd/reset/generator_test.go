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

func TestGenerateWithoutResetStructs(t *testing.T) {
	tempDir := t.TempDir()

	pkgDir := filepath.Join(tempDir, "sample")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", pkgDir, err)
	}

	source := []byte(`package sample

type PlainState struct {
	Name string
}
`)

	sourcePath := filepath.Join(pkgDir, "state.go")
	if err := os.WriteFile(sourcePath, source, 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", sourcePath, err)
	}

	generatedFiles, err := Generate(tempDir)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	got := len(generatedFiles)
	want := 0
	if got != want {
		t.Fatalf("len(generatedFiles) = %d, want %d", got, want)
	}

	generatedPath := filepath.Join(pkgDir, generatedFile)
	if _, err := os.Stat(generatedPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want file not exists", generatedPath, err)
	}
}

func TestGeneratorHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "skip git", got: shouldSkipDir(".git"), want: true},
		{name: "skip github", got: shouldSkipDir(".github"), want: true},
		{name: "skip testdata", got: shouldSkipDir("testdata"), want: true},
		{name: "skip migrations", got: shouldSkipDir("migrations"), want: true},
		{name: "keep internal", got: shouldSkipDir("internal"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}

	sourceTests := []struct {
		path string
		want bool
	}{
		{path: "state.go", want: true},
		{path: "state_test.go", want: false},
		{path: generatedFile, want: false},
		{path: "README.md", want: false},
	}

	for _, tt := range sourceTests {
		t.Run(tt.path, func(t *testing.T) {
			got := isGoSource(tt.path)
			if got != tt.want {
				t.Fatalf("isGoSource(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
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
