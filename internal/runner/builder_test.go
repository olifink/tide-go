package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBuildTargetGo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-go-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	target := DetectBuildTarget(tmpDir, filepath.Join(tmpDir, "main.go"))
	if target.Type != ProjectGo {
		t.Errorf("expected ProjectGo, got %s", target.Type)
	}
	if target.Command != "go build ." {
		t.Errorf("expected 'go build .', got %s", target.Command)
	}
}

func TestDetectBuildTargetMakefile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-make-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("all:\n\techo hi"), 0644)

	target := DetectBuildTarget(tmpDir, "")
	if target.Type != ProjectMake {
		t.Errorf("expected ProjectMake, got %s", target.Type)
	}
	if target.Command != "make" {
		t.Errorf("expected 'make', got %s", target.Command)
	}
}

func TestDetectBuildTargetC(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-c-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainC := filepath.Join(tmpDir, "main.c")
	_ = os.WriteFile(mainC, []byte("int main() { return 0; }"), 0644)

	target := DetectBuildTarget(tmpDir, mainC)
	if target.Type != ProjectC {
		t.Errorf("expected ProjectC, got %s", target.Type)
	}
	if target.Command == "" {
		t.Errorf("expected non-empty command for C")
	}
}
