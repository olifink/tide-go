package filetree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tide/internal/git"
)

func TestFileTreeOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-tree-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test structure
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Readme"), 0644)
	subDir := filepath.Join(tmpDir, "pkg")
	_ = os.Mkdir(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "util.go"), []byte("package pkg"), 0644)

	ft := New(tmpDir, 30, 20)
	if len(ft.VisibleItems) < 2 {
		t.Fatalf("expected at least 2 visible items, got %d", len(ft.VisibleItems))
	}

	// Move cursor down and up
	ft.MoveDown()
	if ft.Cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", ft.Cursor)
	}
	ft.MoveUp()
	if ft.Cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", ft.Cursor)
	}

	// Select file
	mainGoPath := filepath.Join(tmpDir, "main.go")
	ft.SelectFile(mainGoPath)
	if ft.ActiveFile != mainGoPath {
		t.Errorf("expected active file %s, got %s", mainGoPath, ft.ActiveFile)
	}
}

func TestFileTreeExecutableDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-exec-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Regular text file (0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "normal.txt"), []byte("hello"), 0644)

	// Executable binary / script (0755)
	execPath := filepath.Join(tmpDir, "build.sh")
	_ = os.WriteFile(execPath, []byte("#!/bin/sh\necho hi"), 0755)

	ft := New(tmpDir, 40, 20)
	ft.SelectFile(execPath)

	item := ft.SelectedItem()
	if item == nil {
		t.Fatalf("expected to select executable item")
	}
	if !item.IsExecutable {
		t.Errorf("expected item.IsExecutable to be true for 0755 file")
	}

	path, isFile, isExec := ft.ToggleCurrent()
	if !isFile || !isExec || path != execPath {
		t.Errorf("expected isFile=true, isExec=true from ToggleCurrent, got isFile=%v, isExec=%v", isFile, isExec)
	}

	// Verify view renders * prefix
	view := ft.View()
	if !strings.Contains(view, "* build.sh") {
		t.Errorf("expected view to contain '* build.sh', got:\n%s", view)
	}
}

func TestFileTreeToggleHidden(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-hidden-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("node_modules"), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, ".github"), 0755)

	ft := New(tmpDir, 40, 20)

	// Hidden files should NOT be visible initially
	for _, item := range ft.VisibleItems {
		if strings.HasPrefix(item.Name, ".") {
			t.Errorf("did not expect hidden file %s to be visible initially", item.Name)
		}
	}

	// Toggle hidden files ON
	ft.ToggleHidden()
	if !ft.ShowHidden {
		t.Errorf("expected ShowHidden to be true after toggle")
	}

	foundGitignore := false
	foundGithubDir := false
	for _, item := range ft.VisibleItems {
		if item.Name == ".gitignore" {
			foundGitignore = true
		}
		if item.Name == ".github" {
			foundGithubDir = true
		}
	}

	if !foundGitignore || !foundGithubDir {
		t.Errorf("expected .gitignore and .github to be visible after toggle ON")
	}

	// Toggle hidden files OFF
	ft.ToggleHidden()
	if ft.ShowHidden {
		t.Errorf("expected ShowHidden to be false after toggle OFF")
	}
	for _, item := range ft.VisibleItems {
		if strings.HasPrefix(item.Name, ".") {
			t.Errorf("did not expect hidden file %s after toggle OFF", item.Name)
		}
	}
}

func TestFileTreeGitStatusHighlighting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-tree-git-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modPath := filepath.Join(tmpDir, "modified.go")
	_ = os.WriteFile(modPath, []byte("package main"), 0644)
	newPath := filepath.Join(tmpDir, "untracked.go")
	_ = os.WriteFile(newPath, []byte("package main"), 0644)

	ft := New(tmpDir, 40, 20)
	statuses := map[string]git.FileStatus{
		modPath: {
			Path:       "modified.go",
			AbsPath:    modPath,
			StatusType: git.StatusModified,
		},
		newPath: {
			Path:       "untracked.go",
			AbsPath:    newPath,
			StatusType: git.StatusUntracked,
		},
	}
	ft.SetGitStatuses(statuses)

	view := ft.View()
	if !strings.Contains(view, "modified.go") || !strings.Contains(view, "untracked.go") {
		t.Errorf("expected view to contain files, got:\n%s", view)
	}
}

