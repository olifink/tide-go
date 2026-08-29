package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.GeminiModel != "gemini-2.5-flash" {
		t.Errorf("unexpected default model: %s", cfg.GeminiModel)
	}
	if cfg.ChromaTheme != "monokai" {
		t.Errorf("unexpected default theme: %s", cfg.ChromaTheme)
	}
	if cfg.TabWidth != 4 {
		t.Errorf("unexpected default tab width: %d", cfg.TabWidth)
	}
}

func TestGetGeminiAPIKeyEnv(t *testing.T) {
	os.Setenv("GEMINI_API_KEY", "test-key-123")
	defer os.Unsetenv("GEMINI_API_KEY")

	cfg := Config{GeminiAPIKey: "fallback-key"}
	key := GetGeminiAPIKey(cfg)
	if key != "test-key-123" {
		t.Errorf("expected test-key-123, got %s", key)
	}
}

func TestGetGeminiAPIKeyConfig(t *testing.T) {
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("GOOGLE_API_KEY")

	cfg := Config{GeminiAPIKey: "config-key-456"}
	key := GetGeminiAPIKey(cfg)
	if key != "config-key-456" {
		t.Errorf("expected config-key-456, got %s", key)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-test-config-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := Config{
		GeminiAPIKey:        "saved-key",
		GeminiModel:         "gemini-2.0-pro",
		DefaultBuildCommand: "go test ./...",
		ChromaTheme:         "dracula",
		TabWidth:            2,
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded := Load()
	if loaded.GeminiAPIKey != "saved-key" {
		t.Errorf("unexpected loaded key: %s", loaded.GeminiAPIKey)
	}
	if loaded.GeminiModel != "gemini-2.0-pro" {
		t.Errorf("unexpected loaded model: %s", loaded.GeminiModel)
	}
	if loaded.ChromaTheme != "dracula" {
		t.Errorf("unexpected loaded theme: %s", loaded.ChromaTheme)
	}

	// Verify file was created at ~/.config/tide/config.json
	cfgFile := filepath.Join(tmpDir, ".config", "tide", "config.json")
	if _, err := os.Stat(cfgFile); err != nil {
		t.Errorf("expected config file at %s, got err: %v", cfgFile, err)
	}
}
