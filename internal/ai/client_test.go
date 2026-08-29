package ai

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	prompt := BuildPrompt(
		"Why did this fail?",
		"main.go",
		"package main\nfunc main() { panic(\"err\") }",
		"go run .",
		"panic: err\nmain.go:2",
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
	if !strings.Contains(prompt, "panic: err") {
		t.Errorf("prompt missing output trace")
	}
}
