package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"tide/internal/ai"
	"tide/internal/editor"
	"tide/internal/git"
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
	if m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen false when directory launched")
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
	if !m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen true when file launched")
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
	if !m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen true when existing file launched")
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

func TestAppBuildAndRunFocusConsole(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	m.ActivePane = PaneEditor
	m.updateFocus()

	// Trigger Build with Ctrl+B
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = newM.(Model)

	if m.ActivePane != PaneConsole {
		t.Errorf("expected Ctrl+B to switch active pane to PaneConsole, got %d", m.ActivePane)
	}
	if !m.Console.Focused {
		t.Errorf("expected console to be focused")
	}

	// Reset focus to Editor
	m.ActivePane = PaneEditor
	m.updateFocus()

	// Trigger Run with Ctrl+R
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
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

func TestAppFileTreeEnterOpensInEditor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-open-exec-*")
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

	// Switch to Files pane and select the file
	m.ActivePane = PaneFiles
	m.FileTree.SelectFile(execPath)
	m.updateFocus()

	// Press Enter on the item
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	// Should switch active pane to Editor and load file
	if m.ActivePane != PaneEditor {
		t.Errorf("expected active pane to switch to PaneEditor, got %d", m.ActivePane)
	}
	if m.Editor.Buffer.FileName() != "myscript.sh" {
		t.Errorf("expected editor to open myscript.sh, got: %s", m.Editor.Buffer.FileName())
	}
}

func TestAppConsoleToggleMaximize(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 40
	m.recalculateLayout()

	// Switch to Console pane
	m.ActivePane = PaneConsole
	m.updateFocus()

	defaultConsoleH := m.Console.Height

	// Press 'm' to maximize
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = newM.(Model)

	if !m.ConsoleMaximized {
		t.Errorf("expected ConsoleMaximized to be true after pressing 'm'")
	}
	if m.Console.Height <= defaultConsoleH {
		t.Errorf("expected maximized height (%d) > default height (%d)", m.Console.Height, defaultConsoleH)
	}
	if !strings.Contains(m.StatusMessage, "Console maximized") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}

	// Press 'm' again to restore
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = newM.(Model)

	if m.ConsoleMaximized {
		t.Errorf("expected ConsoleMaximized to be false after pressing 'm' second time")
	}
	if m.Console.Height != defaultConsoleH {
		t.Errorf("expected restored height (%d) == default height (%d)", m.Console.Height, defaultConsoleH)
	}
}

func TestAppFileTreeToggleDotKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-dot-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("SECRET=1"), 0644)

	m := InitialModel(tmpDir)
	m.ActivePane = PaneFiles
	m.updateFocus()

	if m.FileTree.ShowHidden {
		t.Errorf("expected ShowHidden false initially")
	}

	// Press '.' in Files pane
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	m = newM.(Model)

	if !m.FileTree.ShowHidden {
		t.Errorf("expected ShowHidden true after pressing '.'")
	}
	if !strings.Contains(m.StatusMessage, "Showing hidden files") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}

	// Press '.' again
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	m = newM.(Model)

	if m.FileTree.ShowHidden {
		t.Errorf("expected ShowHidden false after second '.'")
	}
	if !strings.Contains(m.StatusMessage, "Hiding hidden files") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}
}

func TestAppEditorFullscreenToggle(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Initial state
	if m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen false initially")
	}

	// Press Ctrl+Z to enter Fullscreen
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = newM.(Model)

	if !m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen true after pressing Ctrl+Z")
	}
	if m.ActivePane != PaneEditor {
		t.Errorf("expected ActivePane to be PaneEditor in fullscreen, got %d", m.ActivePane)
	}
	if m.Editor.Width < 90 {
		t.Errorf("expected Editor.Width to expand to full width (~98), got %d", m.Editor.Width)
	}
	if !strings.Contains(m.StatusMessage, "Editor fullscreen enabled") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}

	// Press Ctrl+Z again to restore
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = newM.(Model)

	if m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen false after pressing Ctrl+Z second time")
	}
	if !strings.Contains(m.StatusMessage, "Restored") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}
}

func TestAppEditorFullscreenKeyZInViewMode(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.ActivePane = PaneEditor
	m.Editor.Mode = editor.ModeView
	m.updateFocus()

	// Press 'z' in editor view mode
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = newM.(Model)

	if !m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen true after pressing 'z' in View mode")
	}

	// Press 'z' again in editor view mode to restore
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = newM.(Model)

	if m.EditorFullscreen {
		t.Errorf("expected EditorFullscreen false after pressing 'z' second time")
	}
}

func TestAppEditorPressEEntersEditMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-e-key-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.go")
	_ = os.WriteFile(filePath, []byte("package main"), 0644)

	m := InitialModel(filePath)
	m.Width = 100
	m.Height = 30
	m.Editor.Mode = editor.ModeView
	m.ActivePane = PaneEditor
	m.updateFocus()

	if m.Editor.Mode != editor.ModeView {
		t.Fatalf("expected ModeView initially")
	}

	// Press 'e' in editor view mode
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = newM.(Model)

	if m.Editor.Mode != editor.ModeEdit {
		t.Errorf("expected ModeEdit after pressing 'e', got %d", m.Editor.Mode)
	}
}

func TestAppFileListAltBackspaceOpensShellRm(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-alt-backspace-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fileToDelete := filepath.Join(tmpDir, "obsolete.txt")
	_ = os.WriteFile(fileToDelete, []byte("temp content"), 0644)

	m := InitialModel(tmpDir)
	m.Width = 100
	m.Height = 30
	m.ActivePane = PaneFiles
	m.FileTree.SelectFile(fileToDelete)
	m.updateFocus()

	// Press Alt+Backspace in Files pane
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	m = newM.(Model)

	if !m.Modal.Active {
		t.Fatalf("expected Modal to be active after Alt+Backspace")
	}
	if m.Modal.Type != modal.ShellCommand {
		t.Errorf("expected ModalType ShellCommand, got %v", m.Modal.Type)
	}
	if !strings.Contains(m.Modal.Value(), "rm -f obsolete.txt") {
		t.Errorf("expected modal input to contain 'rm -f obsolete.txt', got %q", m.Modal.Value())
	}
}

func TestAppAltBConfiguresBuildCommand(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// 1. Press Alt+B to open build config dialog
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
	m = newM.(Model)

	if !m.Modal.Active {
		t.Fatalf("expected modal active after Alt+B")
	}
	if m.Modal.Type != modal.BuildCommand {
		t.Errorf("expected ModalType BuildCommand, got %v", m.Modal.Type)
	}

	// 2. Edit command in modal and submit with Enter
	m.Modal.Input.SetValue("make release-all")
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.CustomBuildCmd != "make release-all" {
		t.Errorf("expected CustomBuildCmd 'make release-all', got %q", m.CustomBuildCmd)
	}
	if m.Modal.Active {
		t.Errorf("expected modal closed after submitting")
	}

	// 3. Trigger Ctrl+B to ensure CustomBuildCmd is used
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = newM.(Model)

	if m.Console.RunningTitle != "make release-all" {
		t.Errorf("expected Ctrl+B to run 'make release-all', got %q", m.Console.RunningTitle)
	}
}

func TestAppAltRConfiguresRunCommand(t *testing.T) {
	m := InitialModel(".")
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// 1. Press Alt+R to open run config dialog
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}, Alt: true})
	m = newM.(Model)

	if !m.Modal.Active {
		t.Fatalf("expected modal active after Alt+R")
	}
	if m.Modal.Type != modal.RunCommand {
		t.Errorf("expected ModalType RunCommand, got %v", m.Modal.Type)
	}

	// 2. Edit command in modal and submit with Enter
	m.Modal.Input.SetValue("./build/server --port=8080")
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.CustomRunCmd != "./build/server --port=8080" {
		t.Errorf("expected CustomRunCmd './build/server --port=8080', got %q", m.CustomRunCmd)
	}
	if m.Modal.Active {
		t.Errorf("expected modal closed after submitting")
	}

	// 3. Trigger Ctrl+R to ensure CustomRunCmd is used
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = newM.(Model)

	if m.Console.RunningTitle != "./build/server --port=8080" {
		t.Errorf("expected Ctrl+R to run './build/server --port=8080', got %q", m.Console.RunningTitle)
	}
}

func initTestGitRepoForApp(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "tide-app-git-*")
	if err != nil {
		t.Fatal(err)
	}

	cmdInit := exec.Command("git", "init", "-b", "main")
	cmdInit.Dir = tmpDir
	if err := cmdInit.Run(); err != nil {
		cmdInitOld := exec.Command("git", "init")
		cmdInitOld.Dir = tmpDir
		if err := cmdInitOld.Run(); err != nil {
			t.Fatal(err)
		}
	}

	cmdCfg1 := exec.Command("git", "config", "user.name", "TideTester")
	cmdCfg1.Dir = tmpDir
	_ = cmdCfg1.Run()
	cmdCfg2 := exec.Command("git", "config", "user.email", "tester@tide.local")
	cmdCfg2.Dir = tmpDir
	_ = cmdCfg2.Run()

	return tmpDir
}

func TestAppGitModeIntegration(t *testing.T) {
	if !git.IsInstalled() {
		t.Skip("git not installed")
	}

	repoDir := initTestGitRepoForApp(t)
	defer os.RemoveAll(repoDir)

	// Create initial file and commit
	initialFile := filepath.Join(repoDir, "main.go")
	_ = os.WriteFile(initialFile, []byte("package main\n\nfunc main() {}\n"), 0644)
	cmdAdd := exec.Command("git", "add", "main.go")
	cmdAdd.Dir = repoDir
	_ = cmdAdd.Run()
	cmdCommit := exec.Command("git", "commit", "-m", "initial")
	cmdCommit.Dir = repoDir
	_ = cmdCommit.Run()

	m := InitialModel(repoDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// 1. Verify Git mode detected
	if !m.GitStatus.IsRepo {
		t.Fatalf("expected git repo detected")
	}
	if m.GitStatus.HasChanges {
		t.Errorf("expected clean repo initially")
	}

	// 2. Title bar contains git branch
	viewClean := m.View()
	if !strings.Contains(viewClean, "git:main") {
		t.Errorf("expected title bar to contain 'git:main', got:\n%s", viewClean)
	}

	// 3. Create a change and verify title bar shows git:main*
	_ = os.WriteFile(filepath.Join(repoDir, "newfile.txt"), []byte("new content"), 0644)
	m.refreshWorkspace()

	if !m.GitStatus.HasChanges {
		t.Errorf("expected git status to have changes after adding new file")
	}
	viewDirty := m.View()
	if !strings.Contains(viewDirty, "git:main*") {
		t.Errorf("expected title bar to contain 'git:main*', got:\n%s", viewDirty)
	}
}

func TestAppGitSyncDialogAndAction(t *testing.T) {
	if !git.IsInstalled() {
		t.Skip("git not installed")
	}

	repoDir := initTestGitRepoForApp(t)
	defer os.RemoveAll(repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("data"), 0644)

	m := InitialModel(repoDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Trigger Git Sync with Ctrl+Shift+G / Alt+G
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}, Alt: true})
	m = newM.(Model)

	if !m.Modal.Active {
		t.Fatalf("expected GitSync modal to be active after Alt+G")
	}
	if m.Modal.Type != modal.GitSync {
		t.Errorf("expected ModalType GitSync, got %v", m.Modal.Type)
	}

	// Set commit message and confirm
	m.Modal.Input.SetValue("Add initial file")
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.Modal.Active {
		t.Errorf("expected modal to close after submitting")
	}
	if m.ActivePane != PaneConsole {
		t.Errorf("expected active pane to switch to PaneConsole, got %d", m.ActivePane)
	}
	if !m.Console.IsRunning {
		t.Errorf("expected Console.IsRunning to be true")
	}
	if !strings.Contains(m.StatusMessage, "Committing & pushing") {
		t.Errorf("unexpected status message: %s", m.StatusMessage)
	}
	if cmd == nil {
		t.Errorf("expected tea.Cmd returned to run git command")
	}

	// Test ProcessFinishedMsg with exit code 0 for git sync
	procMsg := runner.ProcessFinishedMsg{
		Command:  "git add -A && git commit -m 'Add initial file' && git push",
		Output:   "[main 1234567] Add initial file\n 1 file changed\n",
		ExitCode: 0,
	}
	newM, _ = m.Update(procMsg)
	m = newM.(Model)

	if !strings.Contains(m.StatusMessage, "pushed successfully") {
		t.Errorf("expected success status message, got %s", m.StatusMessage)
	}
}

func TestAppGitSyncOutsideRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-app-nogit-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := InitialModel(tmpDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Trigger Git Sync with Alt+G in non-repo
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}, Alt: true})
	m = newM.(Model)

	if m.Modal.Active {
		t.Errorf("GitSync modal should not open outside a git repository")
	}
	if !strings.Contains(m.StatusMessage, "Not a git repository") {
		t.Errorf("expected 'Not a git repository' status message, got: %s", m.StatusMessage)
	}
}

func TestAppGitSyncTogglePushAndLocalCommit(t *testing.T) {
	if !git.IsInstalled() {
		t.Skip("git not installed")
	}

	repoDir := initTestGitRepoForApp(t)
	defer os.RemoveAll(repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("data"), 0644)

	m := InitialModel(repoDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Open GitSync modal
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}, Alt: true})
	m = newM.(Model)

	if !m.Modal.PushToRemote {
		t.Errorf("expected PushToRemote true initially")
	}

	// Press Tab to toggle push OFF
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)

	if m.Modal.PushToRemote {
		t.Errorf("expected PushToRemote false after Tab")
	}

	// Submit with Enter
	m.Modal.Input.SetValue("Local commit only")
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.Console.RunningTitle != "git add && commit" {
		t.Errorf("expected RunningTitle 'git add && commit', got %q", m.Console.RunningTitle)
	}

	// Simulate commit finish
	procMsg := runner.ProcessFinishedMsg{
		Command:  "git add -A && git commit -m 'Local commit only'",
		Output:   "[main abc1234] Local commit only\n",
		ExitCode: 0,
	}
	newM, _ = m.Update(procMsg)
	m = newM.(Model)

	if !strings.Contains(m.StatusMessage, "committed locally") {
		t.Errorf("expected 'committed locally' in status message, got %s", m.StatusMessage)
	}
}

func TestAppGitSyncCtrlEnterForcesPush(t *testing.T) {
	if !git.IsInstalled() {
		t.Skip("git not installed")
	}

	repoDir := initTestGitRepoForApp(t)
	defer os.RemoveAll(repoDir)

	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("data"), 0644)

	m := InitialModel(repoDir)
	m.Width = 100
	m.Height = 30
	m.recalculateLayout()

	// Open GitSync modal
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}, Alt: true})
	m = newM.(Model)

	// Press Tab to toggle push OFF
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)

	// Set value and press Ctrl+Enter / Alt+Enter
	m.Modal.Input.SetValue("Push with Ctrl+Enter")
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m = newM.(Model)

	if m.Console.RunningTitle != "git add && commit && push" {
		t.Errorf("expected RunningTitle 'git add && commit && push', got %q", m.Console.RunningTitle)
	}
}

