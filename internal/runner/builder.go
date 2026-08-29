package runner

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectType represents the detected project ecosystem.
type ProjectType string

const (
	ProjectGo      ProjectType = "Go"
	ProjectC       ProjectType = "C/C++"
	ProjectRust    ProjectType = "Rust"
	ProjectMake    ProjectType = "Make"
	ProjectUnknown ProjectType = "Unknown"
)

// BuildTarget contains the detected project type and default build command.
type BuildTarget struct {
	Type        ProjectType
	Command     string
	Description string
}

// RunTarget contains the detected project run command and description.
type RunTarget struct {
	Command     string
	Description string
}

// DetectBuildTarget analyzes the given directory and active file to determine the best build command.
func DetectBuildTarget(dir string, activeFile string) BuildTarget {
	// 1. Check for Cargo.toml (Rust project)
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return BuildTarget{
			Type:        ProjectRust,
			Command:     "cargo build",
			Description: "Cargo build (Cargo.toml detected)",
		}
	}

	// 2. Check for Makefile
	if fileExists(filepath.Join(dir, "Makefile")) || fileExists(filepath.Join(dir, "makefile")) {
		return BuildTarget{
			Type:        ProjectMake,
			Command:     "make",
			Description: "Make build (Makefile detected)",
		}
	}

	// 3. Check for Go module or Go files
	if fileExists(filepath.Join(dir, "go.mod")) {
		return BuildTarget{
			Type:        ProjectGo,
			Command:     "go build .",
			Description: "Go build (go.mod detected)",
		}
	}

	// 4. If active file is a Go file or directory contains .go files
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

	// 5. Check for Rust files (.rs)
	if activeFile != "" && filepath.Ext(activeFile) == ".rs" {
		base := filepath.Base(activeFile)
		outName := strings.TrimSuffix(base, ".rs")
		if fileExists(filepath.Join(dir, "Cargo.toml")) {
			return BuildTarget{
				Type:        ProjectRust,
				Command:     "cargo build",
				Description: "Cargo build (Rust project)",
			}
		}
		return BuildTarget{
			Type:        ProjectRust,
			Command:     "rustc -g " + filepath.ToSlash(activeFile) + " -o " + outName,
			Description: "Rust compile (" + base + ")",
		}
	}
	if hasFilesWithExt(dir, ".rs") {
		if fileExists(filepath.Join(dir, "main.rs")) {
			return BuildTarget{
				Type:        ProjectRust,
				Command:     "rustc -g main.rs -o main",
				Description: "Rust build (main.rs -> main)",
			}
		}
		return BuildTarget{
			Type:        ProjectRust,
			Command:     "cargo build",
			Description: "Cargo build (.rs files found)",
		}
	}

	// 6. Check for C/C++ files
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

	// 7. Fallback
	return BuildTarget{
		Type:        ProjectUnknown,
		Command:     "go build .",
		Description: "Default build (go build .)",
	}
}

// DetectRunTarget determines the command to execute for the active file or workspace.
func DetectRunTarget(dir string, activeFile string) RunTarget {
	if activeFile != "" {
		relPath, err := filepath.Rel(dir, activeFile)
		if err != nil || relPath == "" {
			relPath = filepath.Base(activeFile)
		}
		slashRelPath := filepath.ToSlash(relPath)
		ext := strings.ToLower(filepath.Ext(activeFile))
		base := filepath.Base(activeFile)

		// 1. If active file is already an executable binary on disk (has execute permissions)
		if info, err := os.Stat(activeFile); err == nil && !info.IsDir() && info.Mode().Perm()&0111 != 0 {
			cmd := slashRelPath
			if !strings.HasPrefix(cmd, ".") && !filepath.IsAbs(cmd) {
				cmd = "./" + cmd
			}
			return RunTarget{
				Command:     cmd,
				Description: "Run executable (" + cmd + ")",
			}
		}

		// 2. If active file is explicitly a Makefile
		if strings.EqualFold(base, "Makefile") || strings.HasSuffix(base, ".mk") {
			return RunTarget{
				Command:     "make run",
				Description: "Make run (make run)",
			}
		}

		// 3. Script files: run source directly with interpreter
		switch ext {
		case ".py":
			return RunTarget{
				Command:     "python3 " + slashRelPath,
				Description: "Python (" + base + ")",
			}
		case ".rb":
			return RunTarget{
				Command:     "ruby " + slashRelPath,
				Description: "Ruby (" + base + ")",
			}
		case ".sh", ".bash":
			return RunTarget{
				Command:     "bash " + slashRelPath,
				Description: "Shell script (" + base + ")",
			}
		case ".js":
			return RunTarget{
				Command:     "node " + slashRelPath,
				Description: "Node.js (" + base + ")",
			}
		case ".ts":
			return RunTarget{
				Command:     "ts-node " + slashRelPath,
				Description: "TypeScript (" + base + ")",
			}
		}

		// 4. Rust file
		if ext == ".rs" {
			if fileExists(filepath.Join(dir, "Cargo.toml")) {
				return RunTarget{
					Command:     "cargo run",
					Description: "Cargo run (cargo run)",
				}
			}
			strippedName := base[:len(base)-len(ext)]
			relDir := filepath.Dir(slashRelPath)
			var binCmd string
			if relDir == "." || relDir == "" {
				binCmd = "./" + strippedName
			} else {
				binCmd = "./" + relDir + "/" + strippedName
			}
			return RunTarget{
				Command:     binCmd,
				Description: "Run binary (" + binCmd + ")",
			}
		}

		// 5. Compiled source files (C, C++, Go): strip extension to run resulting binary
		if ext == ".go" || ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
			strippedName := base[:len(base)-len(ext)]
			relDir := filepath.Dir(slashRelPath)

			var binCmd string
			if relDir == "." || relDir == "" {
				binCmd = "./" + strippedName
			} else {
				binCmd = "./" + relDir + "/" + strippedName
			}

			return RunTarget{
				Command:     binCmd,
				Description: "Run binary (" + binCmd + ")",
			}
		}
	}

	// 5. Workspace fallbacks
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return RunTarget{
			Command:     "cargo run",
			Description: "Cargo run (Cargo.toml detected)",
		}
	}

	if fileExists(filepath.Join(dir, "Makefile")) || fileExists(filepath.Join(dir, "makefile")) {
		return RunTarget{
			Command:     "make run",
			Description: "Make run (make run)",
		}
	}

	if fileExists(filepath.Join(dir, "go.mod")) {
		return RunTarget{
			Command:     "go run .",
			Description: "Go run (go run .)",
		}
	}

	// Look for executable candidate in directory
	dirBase := filepath.Base(dir)
	for _, candidate := range []string{dirBase, "main", "app", "program", "a.out"} {
		candPath := filepath.Join(dir, candidate)
		if info, err := os.Stat(candPath); err == nil && !info.IsDir() && info.Mode().Perm()&0111 != 0 {
			return RunTarget{
				Command:     "./" + candidate,
				Description: "Run binary (./" + candidate + ")",
			}
		}
	}

	return RunTarget{
		Command:     "./main",
		Description: "Run binary (./main)",
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
