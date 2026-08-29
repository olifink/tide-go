package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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

func TestBufferReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-reload-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "reload.go")
	_ = os.WriteFile(filePath, []byte("package main\n"), 0644)

	ed := New("monokai", 80, 20)
	if err := ed.OpenFile(filePath); err != nil {
		t.Fatal(err)
	}

	// 1. External change on disk (e.g. gofmt or shell command)
	_ = os.WriteFile(filePath, []byte("package main\n\nfunc NewFunc() {}\n"), 0644)
	changed := ed.Reload()
	if !changed {
		t.Errorf("expected ed.Reload() to return true after external modification")
	}
	if !strings.Contains(ed.Buffer.CurrentText, "NewFunc") {
		t.Errorf("expected buffer to contain updated text from disk")
	}

	// 2. In-memory unsaved edits are protected
	ed.Buffer.SetText("package main\n// in-memory unsaved")
	_ = os.WriteFile(filePath, []byte("package main\n// external change"), 0644)
	changed = ed.Reload()
	if changed {
		t.Errorf("ed.Reload() should not overwrite unsaved in-memory edits")
	}
	if !strings.Contains(ed.Buffer.CurrentText, "in-memory unsaved") {
		t.Errorf("unsaved edits should be preserved")
	}
}

func TestBinaryFileRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-bin-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Binary file with null bytes
	binPath := filepath.Join(tmpDir, "binary.dat")
	binData := []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01, 0x02, 0x00, 0x00}
	if err := os.WriteFile(binPath, binData, 0644); err != nil {
		t.Fatal(err)
	}

	isText, checkErr := IsTextFile(binPath)
	if isText || checkErr == nil {
		t.Errorf("expected binary file to be rejected")
	}

	buf, err := LoadFile(binPath)
	if err == nil || buf.IsLoaded {
		t.Errorf("expected LoadFile on binary file to fail")
	}
	if !strings.Contains(buf.ErrorMessage, "binary") {
		t.Errorf("expected binary error message, got: %s", buf.ErrorMessage)
	}
}

func TestBinaryExtensionRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-ext-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	imgFile := filepath.Join(tmpDir, "photo.png")
	_ = os.WriteFile(imgFile, []byte("fake-png-data"), 0644)

	isText, checkErr := IsTextFile(imgFile)
	if isText || checkErr == nil {
		t.Errorf("expected .png extension to be rejected")
	}
}

func TestLargeFileRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-large-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// File larger than MaxAutoOpenFileSize (2MB + 1KB)
	largeFile := filepath.Join(tmpDir, "huge.txt")
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatal(err)
	}
	chunk := []byte(strings.Repeat("a", 1024))
	for i := 0; i < 2050; i++ { // ~2.05 MB
		_, _ = f.Write(chunk)
	}
	f.Close()

	isText, checkErr := IsTextFile(largeFile)
	if isText || checkErr == nil {
		t.Errorf("expected large file >2MB to be rejected")
	}
	if !strings.Contains(checkErr.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", checkErr)
	}

	buf, _ := LoadFile(largeFile)
	if buf.IsLoaded {
		t.Errorf("expected buffer not to be loaded for large file")
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

func TestHighlightCodeMakefileNoEmptyLines(t *testing.T) {
	makeContent := `CC = gcc
CFLAGS = -Wall -Wextra

all: program

program: main.o
	$(CC) main.o -o program

clean:
	rm -f *.o program
`
	inputLines := strings.Split(strings.TrimRight(makeContent, "\n"), "\n")
	output := HighlightCode("Makefile", makeContent, "monokai", nil, 0, len(inputLines), 80)
	outLines := strings.Split(output, "\n")

	if len(outLines) != len(inputLines) {
		t.Errorf("expected %d output lines, got %d", len(inputLines), len(outLines))
	}

	// Verify line 1 contains CC = gcc and line 2 contains CFLAGS
	if !strings.Contains(ansi.Strip(outLines[0]), "CC = gcc") {
		t.Errorf("line 1 mismatch: %s", outLines[0])
	}
	if !strings.Contains(ansi.Strip(outLines[1]), "CFLAGS = -Wall") {
		t.Errorf("line 2 mismatch: %s", outLines[1])
	}
}

func TestHighlightCodeCRLFHandling(t *testing.T) {
	crlfContent := "package main\r\n\r\nfunc main() {\r\n}\r\n"
	output := HighlightCode("main.go", crlfContent, "monokai", nil, 0, 4, 80)
	outLines := strings.Split(output, "\n")

	if len(outLines) != 4 {
		t.Errorf("expected 4 output lines, got %d", len(outLines))
	}

	for i, l := range outLines {
		if strings.Contains(l, "\r") {
			t.Errorf("line %d contains carriage return: %q", i+1, l)
		}
	}
}
