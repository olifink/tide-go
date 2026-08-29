package ai

import (
	"strings"
	"testing"
)

func TestBuildPromptQA(t *testing.T) {
	prompt := BuildPrompt(
		ModeConsoleQA,
		"Why did this fail?",
		"main.go",
		"package main\nfunc main() { panic(\"err\") }",
		"go run .",
		"panic: err\nmain.go:2",
		nil,
	)

	if !strings.Contains(prompt, "main.go") {
		t.Errorf("prompt missing active file name")
	}
	if !strings.Contains(prompt, "panic(\"err\")") {
		t.Errorf("prompt missing code content")
	}
	if !strings.Contains(prompt, "Why did this fail?") {
		t.Errorf("prompt missing user query")
	}
}

func TestBuildPromptUpdate(t *testing.T) {
	prompt := BuildPrompt(
		ModeUpdateFile,
		"Add error handling to open function",
		"io.go",
		"func open() {}",
		"",
		"",
		nil,
	)

	if !strings.Contains(prompt, "io.go") {
		t.Errorf("update prompt missing file name")
	}
	if !strings.Contains(prompt, "COMPLETE, UPDATED file content") {
		t.Errorf("update prompt missing full file instruction")
	}
}

func TestBuildPromptGenerateFile(t *testing.T) {
	prompt := BuildPrompt(
		ModeGenerateFile,
		"create math utility functions",
		"",
		"",
		"",
		"",
		[]string{"main.go", "go.mod"},
	)

	if !strings.Contains(prompt, "FILENAME:") {
		t.Errorf("generate prompt missing FILENAME instruction")
	}
	if !strings.Contains(prompt, "main.go, go.mod") {
		t.Errorf("generate prompt missing file list")
	}
}

func TestExtractCodeAndFile(t *testing.T) {
	response := `FILENAME: utils.go

Here is the updated utility file:
` + "```go\npackage main\n\nfunc Add(a, b int) int {\n    return a + b\n}\n```\n" + `I implemented the Add function.`

	fn, code, exp := ExtractCodeAndFile(response, "default.go")

	if fn != "utils.go" {
		t.Errorf("expected filename 'utils.go', got '%s'", fn)
	}
	if !strings.Contains(code, "func Add(a, b int) int") {
		t.Errorf("extracted code missing expected function: %s", code)
	}
	if !strings.Contains(exp, "implemented the Add function") {
		t.Errorf("extracted explanation missing details: %s", exp)
	}
}
