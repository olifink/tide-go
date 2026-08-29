package modal

import (
	"testing"
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

	m.OpenGeminiPrompt(true)
	if !m.Active || m.Type != GeminiPrompt {
		t.Errorf("expected GeminiPrompt modal active")
	}

	m.OpenAPIKey()
	if !m.Active || m.Type != APIKey {
		t.Errorf("expected APIKey modal active")
	}
}
