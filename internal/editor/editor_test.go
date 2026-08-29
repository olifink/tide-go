package editor

import (
	"os"
	"path/filepath"
	"testing"
	"tide/internal/runner"
)

func TestBufferLoadAndModify(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-editor-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "example.go")
	initialContent := "package main\n\nfunc main() {\n}\n"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	buf, err := LoadFile(filePath)
	if err != nil {
		t.Fatalf("failed to load file: %v", err)
	}

	if buf.FileName() != "example.go" {
		t.Errorf("unexpected filename: %s", buf.FileName())
	}
	if buf.Language != "Go" {
		t.Errorf("unexpected language: %s", buf.Language)
	}
	if buf.IsModified {
		t.Errorf("buffer should not be modified initially")
	}

	// Modify buffer
	buf.SetText("package main\n\nfunc main() {\n    // modified\n}\n")
	if !buf.IsModified {
		t.Errorf("buffer should be marked modified")
	}

	// Save buffer
	if err := buf.Save(); err != nil {
		t.Fatalf("failed to save buffer: %v", err)
	}
	if buf.IsModified {
		t.Errorf("buffer should not be modified after save")
	}

	// Verify disk content
	savedData, _ := os.ReadFile(filePath)
	if string(savedData) != buf.CurrentText {
		t.Errorf("disk content mismatch")
	}
}

func TestHighlightCodeAndDiagnostics(t *testing.T) {
	code := `package main

import "fmt"

func main() {
	undefinedFunc()
}
`
	diags := map[int][]runner.Diagnostic{
		6: {
			{
				File:     "main.go",
				Line:     6,
				Severity: "error",
				Message:  "undefined: undefinedFunc",
			},
		},
	}

	output := HighlightCode("main.go", code, "monokai", diags, 0, 10, 80)
	if output == "" {
		t.Errorf("expected non-empty highlighted output")
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"main.go", "Go"},
		{"app.c", "C"},
		{"header.h", "C"},
		{"server.cpp", "C++"},
		{"main.rs", "Rust"},
		{"script.py", "Python"},
		{"Makefile", "Makefile"},
		{"README.md", "Markdown"},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.file)
		if got != tt.want {
			t.Errorf("DetectLanguage(%s) = %s, want %s", tt.file, got, tt.want)
		}
	}
}
