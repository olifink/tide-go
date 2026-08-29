package filetree

import (
	"os"
	"path/filepath"
	"testing"
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
