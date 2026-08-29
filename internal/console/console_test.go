package console

import (
	"testing"
	"time"
	"tide/internal/runner"
)

func TestConsoleAddEntries(t *testing.T) {
	c := New(80, 10)
	if len(c.Entries) != 0 {
		t.Errorf("expected 0 entries initially, got %d", len(c.Entries))
	}

	// Add command result
	c.AddCommandResult(runner.ProcessFinishedMsg{
		Command:  "go test ./...",
		Output:   "PASS\nok  example 0.01s",
		ExitCode: 0,
		Duration: 10 * time.Millisecond,
	}, true)

	if len(c.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(c.Entries))
	}
	if c.Entries[0].Type != TypeBuild {
		t.Errorf("expected TypeBuild, got %v", c.Entries[0].Type)
	}

	// Add AI chunk
	c.AddAIChunk("Hello from Gemini", true)
	if len(c.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(c.Entries))
	}
	if c.Entries[1].Type != TypeAI {
		t.Errorf("expected TypeAI, got %v", c.Entries[1].Type)
	}

	c.AddAIChunk("! More text.", false)
	if c.Entries[1].Content != "Hello from Gemini! More text." {
		t.Errorf("unexpected appended AI content: %s", c.Entries[1].Content)
	}

	// Clear console
	c.Clear()
	if len(c.Entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(c.Entries))
	}
}
