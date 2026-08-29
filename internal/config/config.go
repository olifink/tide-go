package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents TIDE user configuration.
type Config struct {
	GeminiAPIKey        string `json:"gemini_api_key,omitempty"`
	GeminiModel         string `json:"gemini_model,omitempty"`
	DefaultBuildCommand string `json:"default_build_command,omitempty"`
	ChromaTheme         string `json:"chroma_theme,omitempty"`
	TabWidth            int    `json:"tab_width,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		GeminiModel: "gemini-2.5-flash",
		ChromaTheme: "monokai",
		TabWidth:    4,
	}
}

// ConfigDir returns the path to ~/.config/tide
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tide"), nil
}

// ConfigPath returns the full path to config.json
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load loads the configuration from disk, falling back to defaults.
func Load() Config {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	_ = json.Unmarshal(data, &cfg)
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = "gemini-2.5-flash"
	}
	if cfg.ChromaTheme == "" {
		cfg.ChromaTheme = "monokai"
	}
	if cfg.TabWidth <= 0 {
		cfg.TabWidth = 4
	}
	return cfg
}

// Save writes the configuration to disk.
func Save(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetGeminiAPIKey returns the Gemini API key from environment variables or config.
func GetGeminiAPIKey(cfg Config) string {
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		return key
	}
	return cfg.GeminiAPIKey
}

// SaveGeminiAPIKey updates and persists the Gemini API key.
func SaveGeminiAPIKey(key string) error {
	cfg := Load()
	cfg.GeminiAPIKey = key
	return Save(cfg)
}
