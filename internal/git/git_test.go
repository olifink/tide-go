package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "tide-git-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	cmdInit := exec.Command("git", "init", "-b", "main")
	cmdInit.Dir = tmpDir
	if err := cmdInit.Run(); err != nil {
		// Fallback for older git without -b
		cmdInitOld := exec.Command("git", "init")
		cmdInitOld.Dir = tmpDir
		if err := cmdInitOld.Run(); err != nil {
			t.Fatal(err)
		}
	}

	// Configure test user
	cmdCfg1 := exec.Command("git", "config", "user.name", "TideTester")
	cmdCfg1.Dir = tmpDir
	_ = cmdCfg1.Run()
	cmdCfg2 := exec.Command("git", "config", "user.email", "tester@tide.local")
	cmdCfg2.Dir = tmpDir
	_ = cmdCfg2.Run()

	return tmpDir
}

func TestGitIsInstalled(t *testing.T) {
	if !IsInstalled() {
		t.Skip("git not installed in test environment")
	}
}

func TestGitStatusInNonRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tide-nogit-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	st := GetStatus(tmpDir)
	if st.IsRepo {
		t.Errorf("expected IsRepo false for non-git dir, got true")
	}
}

func TestGitStatusInRepo(t *testing.T) {
	if !IsInstalled() {
		t.Skip("git not installed")
	}

	repoDir := initTestGitRepo(t)
	defer os.RemoveAll(repoDir)

	// Create initial committed file
	committedFile := filepath.Join(repoDir, "initial.txt")
	_ = os.WriteFile(committedFile, []byte("initial"), 0644)
	cmdAdd := exec.Command("git", "add", "initial.txt")
	cmdAdd.Dir = repoDir
	_ = cmdAdd.Run()
	cmdCommit := exec.Command("git", "commit", "-m", "Initial commit")
	cmdCommit.Dir = repoDir
	_ = cmdCommit.Run()

	// Initial clean status
	st := GetStatus(repoDir)
	if !st.IsRepo {
		t.Fatalf("expected IsRepo true")
	}
	if st.HasChanges {
		t.Errorf("expected clean repo, but HasChanges is true")
	}
	if st.Branch == "" {
		t.Errorf("expected non-empty branch name")
	}

	// Create an untracked file
	untrackedFile := filepath.Join(repoDir, "newfile.go")
	_ = os.WriteFile(untrackedFile, []byte("package main\n"), 0644)

	// Modify the existing file
	_ = os.WriteFile(committedFile, []byte("modified content\n"), 0644)

	// Check status
	stAfter := GetStatus(repoDir)
	if !stAfter.HasChanges {
		t.Errorf("expected HasChanges true after creating/modifying files")
	}
	if stAfter.UntrackedCount < 1 {
		t.Errorf("expected at least 1 untracked file, got %d", stAfter.UntrackedCount)
	}
	if stAfter.ModifiedCount < 1 {
		t.Errorf("expected at least 1 modified file, got %d", stAfter.ModifiedCount)
	}

	// Verify FileStatuses map contains the files
	absUntracked, _ := filepath.Abs(untrackedFile)
	absModified, _ := filepath.Abs(committedFile)

	f1, ok1 := stAfter.FileStatuses[absUntracked]
	if !ok1 || f1.StatusType != StatusUntracked {
		t.Errorf("expected %s to have StatusUntracked, got: %+v (ok=%v)", absUntracked, f1, ok1)
	}

	f2, ok2 := stAfter.FileStatuses[absModified]
	if !ok2 || f2.StatusType != StatusModified {
		t.Errorf("expected %s to have StatusModified, got: %+v (ok=%v)", absModified, f2, ok2)
	}
}

func TestBuildCommitAndPushCmd(t *testing.T) {
	cmdWithPush := BuildCommitCmd("Fix issue with 'quotes' and $vars", true)
	expectedPush := "git add -A && git commit -m 'Fix issue with '\\''quotes'\\'' and $vars' && git push"
	if cmdWithPush != expectedPush {
		t.Errorf("expected command %q, got %q", expectedPush, cmdWithPush)
	}

	cmdNoPush := BuildCommitCmd("Fix issue locally", false)
	expectedNoPush := "git add -A && git commit -m 'Fix issue locally'"
	if cmdNoPush != expectedNoPush {
		t.Errorf("expected command %q, got %q", expectedNoPush, cmdNoPush)
	}

	// Wrapper test
	if BuildCommitAndPushCmd("test") != BuildCommitCmd("test", true) {
		t.Errorf("expected wrapper to match BuildCommitCmd(msg, true)")
	}
}
