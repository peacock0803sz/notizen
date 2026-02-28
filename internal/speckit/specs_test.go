package speckit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a git repo with an optional remote and a specs/ directory.
func initTestRepo(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s", args, out)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "t@t.com")
	run("git", "config", "user.name", "t")
	if remote != "" {
		run("git", "remote", "add", "origin", remote)
	}
	// Create specs/ directory
	if err := os.MkdirAll(filepath.Join(dir, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLink_NewLink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "https://github.com/user/myrepo.git")
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "myrepo")
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if !strings.Contains(msg, "Linked") {
		t.Errorf("expected 'Linked' in message, got: %s", msg)
	}

	linkDir := filepath.Join(sourceDir, "Agents", "Specs", "myrepo")
	info, err := os.Lstat(linkDir)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular dir")
	}
}

func TestLink_AlreadyCorrect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	// First call creates the link
	if _, err := Link(sourceDir, repoPath, "repo"); err != nil {
		t.Fatal(err)
	}
	// Second call must report already linked
	msg, err := Link(sourceDir, repoPath, "repo")
	if err != nil {
		t.Fatalf("second Link() error = %v", err)
	}
	if !strings.Contains(msg, "Already linked") {
		t.Errorf("expected 'Already linked', got: %s", msg)
	}
}

func TestLink_WrongTargetUpdated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sourceDir := t.TempDir()
	linkDir := filepath.Join(sourceDir, "Agents", "Specs", "repo")
	if err := os.MkdirAll(filepath.Dir(linkDir), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a symlink pointing to the wrong target
	if err := os.Symlink("/wrong/target", linkDir); err != nil {
		t.Fatal(err)
	}

	repoPath := initTestRepo(t, "")
	msg, err := Link(sourceDir, repoPath, "repo")
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if !strings.Contains(msg, "Updated") {
		t.Errorf("expected 'Updated', got: %s", msg)
	}
}

func TestLink_RepoPathNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := Link(t.TempDir(), "/nonexistent/path", "repo")
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
}

func TestLink_NoGitRemoteWithoutName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "") // no remote configured

	_, err := Link(t.TempDir(), repoPath, "") // --name not provided
	if err == nil {
		t.Fatal("expected error when no git remote and no --name")
	}
}

func TestLink_NameOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "") // no remote configured
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "custom-name")
	if err != nil {
		t.Fatalf("Link() with name override error = %v", err)
	}
	if !strings.Contains(msg, "custom-name") {
		t.Errorf("expected custom-name in message, got: %s", msg)
	}
}
