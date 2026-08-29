package runner

import (
	"os"
	"path/filepath"
)

// ProjectType represents the detected project ecosystem.
type ProjectType string

const (
	ProjectGo      ProjectType = "Go"
	ProjectC       ProjectType = "C/C++"
	ProjectMake    ProjectType = "Make"
	ProjectUnknown ProjectType = "Unknown"
)

// BuildTarget contains the detected project type and default build command.
type BuildTarget struct {
	Type        ProjectType
	Command     string
	Description string
}

// DetectBuildTarget analyzes the given directory and active file to determine the best build command.
func DetectBuildTarget(dir string, activeFile string) BuildTarget {
	// 1. Check for Makefile
	if fileExists(filepath.Join(dir, "Makefile")) || fileExists(filepath.Join(dir, "makefile")) {
		return BuildTarget{
			Type:        ProjectMake,
			Command:     "make",
			Description: "Make build (Makefile detected)",
		}
	}

	// 2. Check for Go module or Go files
	if fileExists(filepath.Join(dir, "go.mod")) {
		return BuildTarget{
			Type:        ProjectGo,
			Command:     "go build .",
			Description: "Go build (go.mod detected)",
		}
	}

	// 3. If active file is a Go file or directory contains .go files
	if activeFile != "" && filepath.Ext(activeFile) == ".go" {
		return BuildTarget{
			Type:        ProjectGo,
			Command:     "go build .",
			Description: "Go build (.go file active)",
		}
	}
	if hasFilesWithExt(dir, ".go") {
		return BuildTarget{
			Type:        ProjectGo,
			Command:     "go build .",
			Description: "Go build (.go files found)",
		}
	}

	// 4. Check for C/C++ files
	if activeFile != "" && (filepath.Ext(activeFile) == ".c" || filepath.Ext(activeFile) == ".cpp") {
		base := filepath.Base(activeFile)
		outName := base[:len(base)-len(filepath.Ext(base))]
		compiler := "gcc"
		if filepath.Ext(activeFile) == ".cpp" {
			compiler = "g++"
		}
		return BuildTarget{
			Type:        ProjectC,
			Command:     compiler + " -Wall -Wextra -g " + filepath.ToSlash(activeFile) + " -o " + outName,
			Description: "C/C++ compile (" + base + ")",
		}
	}

	if hasFilesWithExt(dir, ".c") {
		// Look for main.c
		if fileExists(filepath.Join(dir, "main.c")) {
			return BuildTarget{
				Type:        ProjectC,
				Command:     "gcc -Wall -Wextra -g main.c -o app",
				Description: "C build (main.c -> app)",
			}
		}
		return BuildTarget{
			Type:        ProjectC,
			Command:     "gcc -Wall -Wextra -g *.c -o app",
			Description: "C build (*.c -> app)",
		}
	}

	// 5. Fallback
	return BuildTarget{
		Type:        ProjectUnknown,
		Command:     "",
		Description: "No known build target detected",
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasFilesWithExt(dir string, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ext {
			return true
		}
	}
	return false
}
