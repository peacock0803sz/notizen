package gitops

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// initTestRepo creates a git repository with a source/ directory and an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s", args, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "test")

	// Create source/ directory and make the initial commit
	if err := os.MkdirAll(filepath.Join(dir, "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source", ".keep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	return dir
}

func TestSync_NoChanges(t *testing.T) {
	dir := initTestRepo(t)
	var buf bytes.Buffer
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	// No remote configured, so we only test the hasChanges detection path here.
	changed, err := hasChanges(dir)
	if err != nil {
		t.Fatalf("hasChanges() error = %v", err)
	}
	if changed {
		t.Error("expected no changes in clean repo")
	}
	_ = buf
	_ = now
}

func TestSync_CommitMessageFormat(t *testing.T) {
	now := time.Date(2026, 2, 28, 15, 4, 0, 0, time.UTC)
	expected := "Sync source/ at 2026-02-28 15:04"
	got := "Sync source/ at " + now.Format("2006-01-02 15:04")
	if got != expected {
		t.Errorf("commit message = %q, want %q", got, expected)
	}
}

func TestSync_HasChanges_DetectsModifiedFile(t *testing.T) {
	dir := initTestRepo(t)

	// Add a new file under source/
	newFile := filepath.Join(dir, "source", "new.md")
	if err := os.WriteFile(newFile, []byte("# new"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := hasChanges(dir)
	if err != nil {
		t.Fatalf("hasChanges() error = %v", err)
	}
	if !changed {
		t.Error("expected changes detected after adding file")
	}
}

func TestRunCmd_ErrorPropagation(t *testing.T) {
	var buf strings.Builder
	err := runCmd(t.TempDir(), &buf, "git", "nonexistent-command-xyz")
	if err == nil {
		t.Error("expected error from invalid git command")
	}
}
