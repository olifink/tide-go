package modal

import (
	"strings"
	"testing"
	"tide/internal/ai"
)

func TestModalLifecycle(t *testing.T) {
	m := New()
	if m.Active {
		t.Errorf("modal should not be active initially")
	}

	m.OpenNewFile()
	if !m.Active || m.Type != NewFile {
		t.Errorf("expected NewFile modal active")
	}

	m.Close()
	if m.Active || m.Type != None {
		t.Errorf("modal should be closed")
	}

	m.OpenShellCommand("go test")
	if !m.Active || m.Type != ShellCommand || m.Value() != "go test" {
		t.Errorf("unexpected shell modal state: %+v", m)
	}

	// Test Clear
	m.Clear()
	if m.Value() != "" {
		t.Errorf("expected empty value after Clear(), got %s", m.Value())
	}
	if !m.Active {
		t.Errorf("modal should remain active after Clear()")
	}

	// 1. Test UpdateFile prompt
	m.OpenGeminiPrompt(ai.ModeUpdateFile, "main.go")
	if !m.Active || m.Type != GeminiPrompt || m.AIMode != ai.ModeUpdateFile {
		t.Errorf("expected ModeUpdateFile prompt active")
	}
	if !strings.Contains(m.Title, "main.go") {
		t.Errorf("expected title to contain main.go, got: %s", m.Title)
	}

	// 2. Test GenerateFile prompt
	m.OpenGeminiPrompt(ai.ModeGenerateFile, "")
	if !m.Active || m.Type != GeminiPrompt || m.AIMode != ai.ModeGenerateFile {
		t.Errorf("expected ModeGenerateFile prompt active")
	}

	// 3. Test ConsoleQA prompt
	m.OpenGeminiPrompt(ai.ModeConsoleQA, "")
	if !m.Active || m.Type != GeminiPrompt || m.AIMode != ai.ModeConsoleQA {
		t.Errorf("expected ModeConsoleQA prompt active")
	}

	m.OpenAPIKey()
	if !m.Active || m.Type != APIKey {
		t.Errorf("expected APIKey modal active")
	}

	m.OpenBuildCommand("make custom-build")
	if !m.Active || m.Type != BuildCommand || m.Value() != "make custom-build" {
		t.Errorf("expected BuildCommand modal with value, got: %+v", m)
	}

	m.OpenRunCommand("./bin/custom-run --arg")
	if !m.Active || m.Type != RunCommand || m.Value() != "./bin/custom-run --arg" {
		t.Errorf("expected RunCommand modal with value, got: %+v", m)
	}

	m.OpenGitSync("feature-branch", 3)
	if !m.Active || m.Type != GitSync {
		t.Errorf("expected GitSync modal active")
	}
	if !m.PushToRemote {
		t.Errorf("expected PushToRemote true initially in GitSync")
	}
	if !strings.Contains(m.Title, "feature-branch") {
		t.Errorf("expected title to contain feature-branch, got: %s", m.Title)
	}
	if !strings.Contains(m.Description, "3 changed file(s)") {
		t.Errorf("expected description to mention 3 changed file(s), got: %s", m.Description)
	}

	m.TogglePush()
	if m.PushToRemote {
		t.Errorf("expected PushToRemote false after TogglePush")
	}
	m.TogglePush()
	if !m.PushToRemote {
		t.Errorf("expected PushToRemote true after second TogglePush")
	}
}
