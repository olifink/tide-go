package editor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// MaxAutoOpenFileSize is the maximum file size (2 MB) allowed for automatic editor viewing.
const MaxAutoOpenFileSize = 2 * 1024 * 1024

// Common binary file extensions that should not be opened as text.
var binaryExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".o": true, ".a": true, ".obj": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".ico": true, ".bmp": true, ".tiff": true,
	".pdf": true, ".zip": true, ".tar": true, ".gz": true,
	".7z": true, ".bz2": true, ".xz": true, ".iso": true,
	".db": true, ".sqlite": true, ".sqlite3": true,
	".wasm": true, ".class": true, ".pyc": true, ".pyo": true,
	".mp3": true, ".mp4": true, ".mkv": true, ".wav": true,
	".avi": true, ".mov": true, ".ttf": true, ".woff": true,
	".woff2": true, ".eot": true,
}

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
	FileSize     int64
}

// NewBuffer creates an empty buffer.
func NewBuffer() Buffer {
	return Buffer{}
}

// IsTextFile checks if a file exists, is within the size limit, and contains valid text data.
func IsTextFile(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("path is a directory")
	}

	// 1. Check size limit
	if info.Size() > MaxAutoOpenFileSize {
		return false, fmt.Errorf("file too large (%s > %s limit)", formatFileSize(info.Size()), formatFileSize(MaxAutoOpenFileSize))
	}

	// 2. Check known binary extensions
	ext := strings.ToLower(filepath.Ext(filePath))
	if binaryExts[ext] {
		return false, fmt.Errorf("binary file extension (%s)", ext)
	}

	// 3. Inspect initial chunk for null bytes or invalid UTF-8
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	if n > 0 && isBinaryData(buf[:n]) {
		return false, fmt.Errorf("binary content detected")
	}

	return true, nil
}

func isBinaryData(data []byte) bool {
	for _, b := range data {
		if b == 0 { // null byte indicates binary
			return true
		}
	}
	return !utf8.Valid(data)
}

func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
}

// LoadFile performs sanity checks and reads a file from disk into the buffer.
func LoadFile(filePath string) (Buffer, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Run text sanity check
	if isText, checkErr := IsTextFile(absPath); !isText {
		errMsg := "unknown error"
		if checkErr != nil {
			errMsg = checkErr.Error()
		}
		info, _ := os.Stat(absPath)
		var size int64
		if info != nil {
			size = info.Size()
		}
		return Buffer{
			FilePath:     absPath,
			Language:     DetectLanguage(absPath),
			IsLoaded:     false,
			ErrorMessage: errMsg,
			FileSize:     size,
		}, checkErr
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
		FileSize:    int64(len(data)),
	}, nil
}

// Reload re-reads the file from disk if it hasn't been modified in memory.
// Returns true if content on disk changed.
func (b *Buffer) Reload() (bool, error) {
	if b.FilePath == "" {
		return false, nil
	}

	// If modified in memory, preserve user's unsaved changes
	if b.IsModified {
		return false, nil
	}

	stat, err := os.Stat(b.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			b.IsLoaded = false
			b.ErrorMessage = "File was deleted on disk"
			return true, err
		}
		return false, err
	}

	if isText, checkErr := IsTextFile(b.FilePath); !isText {
		b.IsLoaded = false
		b.ErrorMessage = checkErr.Error()
		b.FileSize = stat.Size()
		return true, checkErr
	}

	data, err := os.ReadFile(b.FilePath)
	if err != nil {
		return false, err
	}

	newText := string(data)
	if newText == b.InitialText {
		return false, nil // No change
	}

	lines := strings.Split(newText, "\n")
	b.InitialText = newText
	b.CurrentText = newText
	b.IsModified = false
	b.LineCount = len(lines)
	b.Language = DetectLanguage(b.FilePath)
	b.IsLoaded = true
	b.FileSize = int64(len(data))
	b.ErrorMessage = ""
	return true, nil
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
	b.FileSize = int64(len(b.CurrentText))
	return nil
}

// FileName returns the base filename.
func (b *Buffer) FileName() string {
	if b.FilePath == "" {
		return "[No Name]"
	}
	return filepath.Base(b.FilePath)
}
