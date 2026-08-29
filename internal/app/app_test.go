package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"tide/internal/ai"
	"tide/internal/modal"
	"tide/internal/runner"
)

func TestAppInitialModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-app-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644)

	m := InitialModel(tmpDir)
	if m.WorkingDir != tmpDir {
		t.Errorf("expected working dir %s, got %s", tmpDir, m.WorkingDir)
	}
	if m.Editor.Buffer.FileName() != "main.go" {
		t.Errorf("expected main.go loaded, got %s", m.Editor.Buffer.FileName())
	}
}

func TestAppInitialModelWithNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-newfile-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	newFilePath := filepath.Join(tmpDir, "src", "sub", "app.c")

	// Pass non-existent file path
	m := InitialModel(newFilePath)

	// Directory should be parent directory (src/sub)
	expectedDir := filepath.Dir(newFilePath)
	if m.WorkingDir != expectedDir {
		t.Errorf("expected working dir %s, got %s", expectedDir, m.WorkingDir)
	}

	// File should be created on disk
	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Errorf("file %s should have been created on disk", newFilePath)
	}

	// Editor should have file opened and loaded
	if m.Editor.Buffer.FileName() != "app.c" {
		t.Errorf("expected editor to open app.c, got %s", m.Editor.Buffer.FileName())
	}
	if !m.Editor.Buffer.IsLoaded {
		t.Errorf("expected buffer to be loaded")
	}
	if m.ActivePane != PaneEditor {
		t.Errorf("expected PaneEditor active, got %d", m.ActivePane)
	}
	if !strings.Contains(m.StatusMessage, "Created & opened app.c") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}
}

func TestAppInitialModelWithExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-existfile-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	existingFile := filepath.Join(tmpDir, "server.go")
	_ = os.WriteFile(existingFile, []byte("package main\n"), 0644)

	m := InitialModel(existingFile)
	if m.WorkingDir != tmpDir {
		t.Errorf("expected working dir %s, got %s", tmpDir, m.WorkingDir)
	}
	if m.Editor.Buffer.FileName() != "server.go" {
		t.Errorf("expected server.go loaded, got %s", m.Editor.Buffer.FileName())
	}
	if m.ActivePane != PaneEditor {
		t.Errorf("expected PaneEditor active for existing file")
	}
}

func TestAppKeyNavigation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-app-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := InitialModel(tmpDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Initial pane is Files
	if m.ActivePane != PaneFiles {
		t.Errorf("expected initial pane PaneFiles")
	}

	// Tab moves to Editor
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.ActivePane != PaneEditor {
		t.Errorf("expected PaneEditor after tab, got %d", m.ActivePane)
	}

	// Tab moves to Console
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.ActivePane != PaneConsole {
		t.Errorf("expected PaneConsole after 2nd tab, got %d", m.ActivePane)
	}

	// Ctrl+N opens NewFile modal
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = newM.(Model)
	if !m.Modal.Active || m.Modal.Type != modal.NewFile {
		t.Errorf("expected NewFile modal active")
	}

	// Esc on empty modal closes modal
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(Model)
	if m.Modal.Active {
		t.Errorf("modal should be closed after Esc")
	}
}

func TestAppModalEscClearsThenCloses(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Open shell modal with default command
	m.Modal.OpenShellCommand("git status")
	if !m.Modal.Active || m.Modal.Value() != "git status" {
		t.Fatalf("modal not initialized properly")
	}

	// 1st Esc: clears input, modal remains open
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(Model)
	if !m.Modal.Active {
		t.Errorf("modal should still be active after 1st Esc")
	}
	if m.Modal.Value() != "" {
		t.Errorf("modal input should be empty after 1st Esc, got: %s", m.Modal.Value())
	}

	// 2nd Esc: closes modal
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(Model)
	if m.Modal.Active {
		t.Errorf("modal should be closed after 2nd Esc on empty input")
	}
}

func TestAppProcessFinishedMsg(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	procMsg := runner.ProcessFinishedMsg{
		Command:  "go build .",
		Output:   "main.go:4:5: undefined: fmt.Println\n",
		ExitCode: 1,
		Diagnostics: []runner.Diagnostic{
			{
				File:     "main.go",
				Line:     4,
				Col:      5,
				Severity: "error",
				Message:  "undefined: fmt.Println",
			},
		},
	}

	newM, _ := m.Update(procMsg)
	m = newM.(Model)

	if len(m.Diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(m.Diagnostics))
	}
	if len(m.Console.Entries) == 0 {
		t.Errorf("expected console entry added")
	}
}

func TestAppTabInEditModeInsertsTab(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-tab-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.go")
	_ = os.WriteFile(filePath, []byte("func main() {\n}\n"), 0644)

	m := InitialModel(filePath)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Switch to Edit Mode
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = newM.(Model)
	if m.Editor.Mode != 1 { // ModeEdit
		t.Fatalf("expected editor in edit mode")
	}

	// Press Tab
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)

	// Pane should remain PaneEditor (NOT switch to PaneConsole)
	if m.ActivePane != PaneEditor {
		t.Errorf("expected pane to remain PaneEditor, got %d", m.ActivePane)
	}

	// Textarea should contain tab or soft-tab indentation
	val := m.Editor.Textarea.Value()
	if !strings.Contains(val, "\t") && !strings.Contains(val, "    ") {
		t.Errorf("expected textarea to contain tab indentation, got: %q", val)
	}
}

func TestAppShiftEscFocusesConsole(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	m.ActivePane = PaneEditor
	m.updateFocus()

	// Test alt+esc / shift+esc
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape, Alt: true})
	m = newM.(Model)

	if m.ActivePane != PaneConsole {
		t.Errorf("expected Shift+Esc / Alt+Esc to focus console, got %d", m.ActivePane)
	}
	if !m.Console.Focused {
		t.Errorf("expected console to be focused")
	}
}

func TestAppBuildFocusesConsole(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	m.ActivePane = PaneEditor
	m.updateFocus()

	// Trigger Build with Ctrl+R
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = newM.(Model)

	if m.ActivePane != PaneConsole {
		t.Errorf("expected Ctrl+R to switch active pane to PaneConsole, got %d", m.ActivePane)
	}
	if !m.Console.Focused {
		t.Errorf("expected console to be focused")
	}
}

func TestAppGeminiUpdateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-ai-update-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "main.go")
	_ = os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0644)

	m := InitialModel(filePath)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Simulate streaming chunks for file update
	chunk1 := "Here is the refactored code:\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Gemini!\")\n}\n```\nAdded greeting."
	newM, _ := m.Update(ai.AIChunkMsg{Chunk: chunk1, Mode: ai.ModeUpdateFile})
	m = newM.(Model)

	// Simulate stream completion
	newM, _ = m.Update(ai.AIChunkMsg{Done: true, Mode: ai.ModeUpdateFile})
	m = newM.(Model)

	// Verify buffer text was updated
	if !strings.Contains(m.Editor.Buffer.CurrentText, "Hello, Gemini!") {
		t.Errorf("expected buffer to contain updated code, got: %s", m.Editor.Buffer.CurrentText)
	}
	if !m.Editor.Buffer.IsModified {
		t.Errorf("expected buffer to be marked modified")
	}
	if m.ActivePane != PaneEditor {
		t.Errorf("expected editor pane active after Gemini update")
	}
}

func TestAppGeminiGenerateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-ai-gen-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := InitialModel(tmpDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Simulate streaming chunks for new file generation
	chunk1 := "FILENAME: math.go\n\n```go\npackage main\n\nfunc Multiply(a, b int) int {\n    return a * b\n}\n```\nImplemented math functions."
	newM, _ := m.Update(ai.AIChunkMsg{Chunk: chunk1, Mode: ai.ModeGenerateFile})
	m = newM.(Model)

	// Simulate stream completion
	newM, _ = m.Update(ai.AIChunkMsg{Done: true, Mode: ai.ModeGenerateFile})
	m = newM.(Model)

	// Verify new file was written and opened
	genFilePath := filepath.Join(tmpDir, "math.go")
	data, err := os.ReadFile(genFilePath)
	if err != nil {
		t.Fatalf("expected generated file on disk: %v", err)
	}
	if !strings.Contains(string(data), "func Multiply") {
		t.Errorf("expected generated file to contain code, got: %s", string(data))
	}
	if m.Editor.Buffer.FileName() != "math.go" {
		t.Errorf("expected editor to open math.go, got: %s", m.Editor.Buffer.FileName())
	}
}

func TestAppFileTreeEnterOnExecutableRunsShell(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-run-exec-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "myscript.sh")
	_ = os.WriteFile(execPath, []byte("#!/bin/sh\necho test-output\n"), 0755)

	m := InitialModel(tmpDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Switch to Files pane and select the executable
	m.ActivePane = PaneFiles
	m.FileTree.SelectFile(execPath)
	m.updateFocus()

	// Press Enter on the executable item
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	// Should switch active pane to Console and start running
	if m.ActivePane != PaneConsole {
		t.Errorf("expected active pane to switch to PaneConsole, got %d", m.ActivePane)
	}
	if !m.Console.IsRunning {
		t.Errorf("expected console to be running")
	}
	if !strings.Contains(m.Console.RunningTitle, "myscript.sh") {
		t.Errorf("expected running title to contain myscript.sh, got: %s", m.Console.RunningTitle)
	}
	if cmd == nil {
		t.Errorf("expected command returned for execution")
	}
}
