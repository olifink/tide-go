package runner

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDetectBuildTargetRust(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-rust-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test Cargo.toml
	_ = os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
	target := DetectBuildTarget(tmpDir, "")
	if target.Type != ProjectRust {
		t.Errorf("expected ProjectRust, got %s", target.Type)
	}
	if target.Command != "cargo build" {
		t.Errorf("expected 'cargo build', got %s", target.Command)
	}
}

func TestDetectRunTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-run-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Python script
	pyFile := filepath.Join(tmpDir, "script.py")
	_ = os.WriteFile(pyFile, []byte("print('hi')"), 0644)
	rtPy := DetectRunTarget(tmpDir, pyFile)
	if !strings.HasPrefix(rtPy.Command, "python3 ") {
		t.Errorf("expected python3 runner for .py, got: %s", rtPy.Command)
	}

	// 2. Ruby script
	rbFile := filepath.Join(tmpDir, "app.rb")
	_ = os.WriteFile(rbFile, []byte("puts 'hi'"), 0644)
	rtRb := DetectRunTarget(tmpDir, rbFile)
	if !strings.HasPrefix(rtRb.Command, "ruby ") {
		t.Errorf("expected ruby runner for .rb, got: %s", rtRb.Command)
	}

	// 3. C source file -> stripped binary
	cFile := filepath.Join(tmpDir, "server.c")
	_ = os.WriteFile(cFile, []byte("int main(){}"), 0644)
	rtC := DetectRunTarget(tmpDir, cFile)
	if rtC.Command != "./server" {
		t.Errorf("expected './server' for server.c, got: %s", rtC.Command)
	}

	// 4. Go source file -> stripped binary
	goFile := filepath.Join(tmpDir, "main.go")
	_ = os.WriteFile(goFile, []byte("package main"), 0644)
	rtGo := DetectRunTarget(tmpDir, goFile)
	if rtGo.Command != "./main" {
		t.Errorf("expected './main' for main.go, got: %s", rtGo.Command)
	}

	// 5. Rust with Cargo.toml -> cargo run
	_ = os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]"), 0644)
	rsFile := filepath.Join(tmpDir, "main.rs")
	_ = os.WriteFile(rsFile, []byte("fn main(){}"), 0644)
	rtRs := DetectRunTarget(tmpDir, rsFile)
	if rtRs.Command != "cargo run" {
		t.Errorf("expected 'cargo run' for Cargo project, got: %s", rtRs.Command)
	}

	// 6. Executable binary
	binFile := filepath.Join(tmpDir, "mytool")
	_ = os.WriteFile(binFile, []byte("ELF binary"), 0755)
	rtBin := DetectRunTarget(tmpDir, binFile)
	if rtBin.Command != "./mytool" {
		t.Errorf("expected './mytool' for binary, got: %s", rtBin.Command)
	}
}
