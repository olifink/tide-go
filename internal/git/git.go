package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// StatusType represents the git status of an individual file.
type StatusType int

const (
	StatusClean StatusType = iota
	StatusModified
	StatusAdded
	StatusUntracked
	StatusDeleted
	StatusRenamed
	StatusConflict
)

// FileStatus holds status details for a single file.
type FileStatus struct {
	Path       string     // Relative path from repo root
	AbsPath    string     // Absolute path on disk
	StatusType StatusType // Category of change
	Staged     bool       // Staged in index
	Worktree   bool       // Modified in worktree
}

// RepoStatus holds aggregated git status information for a repository.
type RepoStatus struct {
	IsRepo         bool
	RepoRoot       string
	Branch         string
	FileStatuses   map[string]FileStatus // Keyed by absolute path
	ModifiedCount  int
	AddedCount     int
	UntrackedCount int
	DeletedCount   int
	HasChanges     bool
}

// IsInstalled returns true if the git executable is in the system PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetRepoRoot returns the top-level directory of the git repo containing dir, or error if not in a repo.
func GetRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetBranch returns the active git branch name, or short commit hash if detached.
func GetBranch(dir string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		branch := strings.TrimSpace(stdout.String())
		if branch != "" {
			return branch
		}
	}

	// If detached HEAD or empty branch name, fallback to short SHA
	cmdSha := exec.Command("git", "rev-parse", "--short", "HEAD")
	if dir != "" {
		cmdSha.Dir = dir
	}
	stdout.Reset()
	cmdSha.Stdout = &stdout
	if err := cmdSha.Run(); err == nil {
		sha := strings.TrimSpace(stdout.String())
		if sha != "" {
			return sha
		}
	}

	return "HEAD"
}

// GetStatus retrieves the full RepoStatus for the given directory.
func GetStatus(dir string) RepoStatus {
	if !IsInstalled() {
		return RepoStatus{IsRepo: false}
	}

	repoRoot, err := GetRepoRoot(dir)
	if err != nil || repoRoot == "" {
		return RepoStatus{IsRepo: false}
	}

	branch := GetBranch(repoRoot)

	// Run git status --porcelain=v1 -uall
	cmd := exec.Command("git", "status", "--porcelain=v1", "-uall")
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return RepoStatus{
			IsRepo:   true,
			RepoRoot: repoRoot,
			Branch:   branch,
		}
	}

	fileStatuses := make(map[string]FileStatus)
	lines := strings.Split(stdout.String(), "\n")
	modCount := 0
	addCount := 0
	untrackCount := 0
	delCount := 0

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		x := line[0]
		y := line[1]
		relPath := strings.TrimSpace(line[3:])

		if strings.Contains(relPath, " -> ") {
			parts := strings.Split(relPath, " -> ")
			relPath = parts[len(parts)-1]
		}
		relPath = strings.Trim(relPath, "\"")
		absPath := filepath.Join(repoRoot, relPath)

		status := FileStatus{
			Path:     relPath,
			AbsPath:  absPath,
			Staged:   x != ' ' && x != '?',
			Worktree: y != ' ' && y != '?',
		}

		if x == '?' && y == '?' {
			status.StatusType = StatusUntracked
			untrackCount++
		} else if x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D') {
			status.StatusType = StatusConflict
			modCount++
		} else if x == 'A' || y == 'A' {
			status.StatusType = StatusAdded
			addCount++
		} else if x == 'D' || y == 'D' {
			status.StatusType = StatusDeleted
			delCount++
		} else if x == 'R' || y == 'R' {
			status.StatusType = StatusRenamed
			modCount++
		} else {
			status.StatusType = StatusModified
			modCount++
		}

		fileStatuses[absPath] = status
	}

	hasChanges := len(fileStatuses) > 0

	return RepoStatus{
		IsRepo:         true,
		RepoRoot:       repoRoot,
		Branch:         branch,
		FileStatuses:   fileStatuses,
		ModifiedCount:  modCount,
		AddedCount:     addCount,
		UntrackedCount: untrackCount,
		DeletedCount:   delCount,
		HasChanges:     hasChanges,
	}
}

// BuildCommitCmd constructs the shell command to add all changes and commit with message (optionally pushing to remote).
func BuildCommitCmd(message string, push bool) string {
	safeMsg := "'" + strings.ReplaceAll(message, "'", "'\\''") + "'"
	if push {
		return fmt.Sprintf("git add -A && git commit -m %s && git push", safeMsg)
	}
	return fmt.Sprintf("git add -A && git commit -m %s", safeMsg)
}

// BuildCommitAndPushCmd constructs the shell command to add all changes, commit with message, and push.
func BuildCommitAndPushCmd(message string) string {
	return BuildCommitCmd(message, true)
}
