package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
		t.Errorf("expected FileName 'example.go', got %s", buf.FileName())
	}
	if buf.Language != "Go" {
		t.Errorf("expected Language 'Go', got %s", buf.Language)
	}
	if buf.IsModified {
		t.Errorf("expected IsModified false initially")
	}

	// Modify text
	buf.SetText("package main\n\nfunc main() {\n    println(\"hello\")\n}\n")
	if !buf.IsModified {
		t.Errorf("expected IsModified true after change")
	}

	// Save back
	if err := buf.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	if buf.IsModified {
		t.Errorf("expected IsModified false after save")
	}

	// Verify disk content
	savedData, _ := os.ReadFile(filePath)
	if string(savedData) != buf.CurrentText {
		t.Errorf("disk content does not match buffer content")
	}
}

func TestBufferReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-reload-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.go")
	_ = os.WriteFile(filePath, []byte("version 1"), 0644)

	buf, err := LoadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	// No external change
	reloaded, err := buf.Reload()
	if err != nil || reloaded {
		t.Errorf("expected no reload when unchanged")
	}

	// External change on disk (e.g. gofmt or git checkout)
	_ = os.WriteFile(filePath, []byte("version 2 (reloaded)"), 0644)

	reloaded, err = buf.Reload()
	if err != nil || !reloaded {
		t.Errorf("expected reload to succeed")
	}
	if buf.CurrentText != "version 2 (reloaded)" {
		t.Errorf("expected reloaded text, got: %s", buf.CurrentText)
	}

	// In-memory edit should NOT be overwritten by external reload
	buf.SetText("in-memory uncommitted edit")
	buf.IsModified = true

	_ = os.WriteFile(filePath, []byte("version 3 on disk"), 0644)
	reloaded, _ = buf.Reload()
	if reloaded {
		t.Errorf("expected reload to be skipped when buffer has unsaved edits")
	}
	if buf.CurrentText != "in-memory uncommitted edit" {
		t.Errorf("expected unsaved in-memory edit to be preserved")
	}
}

func TestBinaryFileRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-bin-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	binFile := filepath.Join(tmpDir, "sample.bin")
	// Write null bytes
	binaryData := []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x00, 0x01, 0x01}
	_ = os.WriteFile(binFile, binaryData, 0644)

	isText, checkErr := IsTextFile(binFile)
	if isText || checkErr == nil {
		t.Errorf("expected binary file to be rejected")
	}

	buf, _ := LoadFile(binFile)
	if buf.IsLoaded {
		t.Errorf("expected buffer not to be loaded for binary file")
	}
	if buf.ErrorMessage == "" {
		t.Errorf("expected non-empty ErrorMessage for binary file")
	}
}

func TestBinaryExtensionRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-ext-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	zipFile := filepath.Join(tmpDir, "archive.zip")
	_ = os.WriteFile(zipFile, []byte("dummy zip content"), 0644)

	isText, checkErr := IsTextFile(zipFile)
	if isText || checkErr == nil {
		t.Errorf("expected .zip to be rejected")
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

	output := HighlightCode("main.go", code, "monokai", diags, 0, 10, 80, false, 0)
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
	output := HighlightCode("Makefile", makeContent, "monokai", nil, 0, len(inputLines), 80, false, 0)
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
	output := HighlightCode("main.go", crlfContent, "monokai", nil, 0, 4, 80, false, 0)
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

func TestHighlightCodeWordWrap(t *testing.T) {
	longLine := "func VeryLongFunctionWithManyParameters(firstParameter string, secondParameter int, thirdParameter bool) (string, error) {"
	output := HighlightCode("main.go", longLine, "monokai", nil, 0, 10, 40, true, 0)
	outLines := strings.Split(output, "\n")

	// Should wrap into multiple visual lines with continuation indicator ↳
	hasContinuation := false
	for _, l := range outLines {
		if strings.Contains(l, "↳") {
			hasContinuation = true
			break
		}
	}
	if !hasContinuation {
		t.Errorf("expected wrapped output to contain continuation arrow ↳, got:\n%s", output)
	}
}

func TestHighlightCodeHorizontalScroll(t *testing.T) {
	longLine := "1234567890abcdefghijklmnopqrstuvwxyz"
	// Output with scrollCol=0 vs scrollCol=10
	out0 := ansi.Strip(HighlightCode("test.txt", longLine, "monokai", nil, 0, 1, 80, false, 0))
	out10 := ansi.Strip(HighlightCode("test.txt", longLine, "monokai", nil, 0, 1, 80, false, 10))

	if !strings.Contains(out0, "1234567890") {
		t.Errorf("expected out0 to contain prefix, got: %s", out0)
	}
	if strings.Contains(out10, "1234567890") {
		t.Errorf("expected out10 to have scrolled past prefix, got: %s", out10)
	}
	if !strings.Contains(out10, "abcdefgh") {
		t.Errorf("expected out10 to contain scrolled content, got: %s", out10)
	}
}

func TestNormalizeMakefileTabs(t *testing.T) {
	spaceMakefile := `CC = gcc
CFLAGS = -Wall

all: program

program: main.o
    $(CC) main.o -o program

clean:
    rm -f *.o program
`
	normalized := NormalizeMakefileTabs(spaceMakefile)
	lines := strings.Split(normalized, "\n")

	// Check line 7 (program recipe) and line 10 (clean recipe) start with \t
	if !strings.HasPrefix(lines[6], "\t$(CC)") {
		t.Errorf("expected line 6 to start with tab, got: %q", lines[6])
	}
	if !strings.HasPrefix(lines[9], "\trm") {
		t.Errorf("expected line 9 to start with tab, got: %q", lines[9])
	}
}

func TestMakefileTabPreservationInEditor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-make-edit-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	makePath := filepath.Join(tmpDir, "Makefile")
	initialContent := "all: program\n\nprogram:\n\tgcc main.c -o program\n"
	_ = os.WriteFile(makePath, []byte(initialContent), 0644)

	ed := New("monokai", 80, 24)
	if err := ed.OpenFile(makePath); err != nil {
		t.Fatalf("failed to open Makefile: %v", err)
	}

	// Switch to edit mode
	ed.ToggleMode()
	if ed.Mode != ModeEdit {
		t.Fatalf("expected ModeEdit")
	}

	// In edit mode, add a new recipe line by typing
	ed.Textarea.SetValue("all: program\n\nprogram:\n    gcc main.c -o program\n\nclean:\n    rm -f program\n")

	// Switch back to view mode (triggers SetText)
	ed.ToggleMode()
	if ed.Mode != ModeView {
		t.Fatalf("expected ModeView")
	}

	// Save file
	if err := ed.SaveFile(); err != nil {
		t.Fatalf("failed to save Makefile: %v", err)
	}

	// Read from disk
	savedBytes, err := os.ReadFile(makePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	savedStr := string(savedBytes)

	if !strings.Contains(savedStr, "\tgcc main.c") {
		t.Errorf("expected saved file to contain '\\tgcc', got:\n%s", savedStr)
	}
	if !strings.Contains(savedStr, "\trm -f") {
		t.Errorf("expected saved file to contain '\\trm', got:\n%s", savedStr)
	}
}

func TestEditorTabKeyInsertsIndentation(t *testing.T) {
	ed := New("monokai", 80, 24)
	ed.Buffer.FilePath = "Makefile"
	ed.Buffer.IsLoaded = true
	ed.Buffer.Language = "Makefile"
	ed.Buffer.UsesTabs = true
	ed.Mode = ModeEdit

	ed.Textarea.SetValue("all:\n")
	ed.Textarea.CursorDown()

	// Press Tab on recipe line
	newEd, _ := ed.Update(tea.KeyMsg{Type: tea.KeyTab})
	ed = newEd
	ed.Textarea.InsertString("gcc main.c -o all")

	// Toggle mode back to view mode to commit
	ed.ToggleMode()

	if !strings.Contains(ed.Buffer.CurrentText, "\tgcc") {
		t.Errorf("expected buffer text to contain '\\tgcc' after Tab key press in Makefile, got: %q", ed.Buffer.CurrentText)
	}
}
