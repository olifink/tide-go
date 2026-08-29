package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
