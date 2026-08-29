package editor

import (
	"os"
	"path/filepath"
	"strings"
)

// Buffer holds the loaded file path, content, and modification state.
type Buffer struct {
	FilePath     string
	InitialText  string
	CurrentText  string
	IsModified   bool
	LineCount    int
	Language     string
	IsLoaded     bool
	ErrorMessage string
}

// NewBuffer creates an empty buffer.
func NewBuffer() Buffer {
	return Buffer{}
}

// LoadFile reads a file from disk into the buffer.
func LoadFile(filePath string) (Buffer, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return Buffer{
			FilePath:     absPath,
			Language:     DetectLanguage(absPath),
			IsLoaded:     false,
			ErrorMessage: err.Error(),
		}, err
	}

	text := string(data)
	lines := strings.Split(text, "\n")

	return Buffer{
		FilePath:    absPath,
		InitialText: text,
		CurrentText: text,
		IsModified:  false,
		LineCount:   len(lines),
		Language:    DetectLanguage(absPath),
		IsLoaded:    true,
	}, nil
}

// SetText updates current text and recalculates modification status.
func (b *Buffer) SetText(newText string) {
	b.CurrentText = newText
	b.IsModified = (b.CurrentText != b.InitialText)
	lines := strings.Split(newText, "\n")
	b.LineCount = len(lines)
}

// Save writes current buffer text back to disk.
func (b *Buffer) Save() error {
	if b.FilePath == "" {
		return nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(b.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	err := os.WriteFile(b.FilePath, []byte(b.CurrentText), 0644)
	if err != nil {
		return err
	}

	b.InitialText = b.CurrentText
	b.IsModified = false
	return nil
}

// FileName returns the base filename.
func (b *Buffer) FileName() string {
	if b.FilePath == "" {
		return "[No Name]"
	}
	return filepath.Base(b.FilePath)
}
