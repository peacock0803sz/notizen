package speckit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peacock0803sz/notizen/internal/repolink"
)

// initTestRepo creates a git repo with an optional remote and a specs/ directory.
func initTestRepo(t *testing.T, remote string) string {
	t.Helper()
	// Isolate git from the developer's global/system config (e.g. signing).
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "t@t.com")
	runGit(t, dir, "config", "user.name", "t")
	if remote != "" {
		runGit(t, dir, "remote", "add", "origin", remote)
	}
	if err := os.MkdirAll(filepath.Join(dir, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
}

func canonical(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestLink_NewLink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "https://github.com/user/myrepo.git")
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "myrepo", false)
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
	if _, err := Link(sourceDir, repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	// Second call must report already linked
	msg, err := Link(sourceDir, repoPath, "repo", false)
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
	msg, err := Link(sourceDir, repoPath, "repo", false)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if !strings.Contains(msg, "Updated") {
		t.Errorf("expected 'Updated', got: %s", msg)
	}
	want := filepath.Join(canonical(t, repoPath), "specs")
	got, err := os.Readlink(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("link target: got %q, want %q", got, want)
	}
}

func TestLink_RepoPathNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := Link(t.TempDir(), "/nonexistent/path", "repo", false)
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
}

func TestLink_NoGitRemoteWithoutName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "") // no remote configured

	_, err := Link(t.TempDir(), repoPath, "", false) // --name not provided
	if err == nil {
		t.Fatal("expected error when no git remote and no --name")
	}
}

func TestLink_NameOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "") // no remote configured
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "custom-name", false)
	if err != nil {
		t.Fatalf("Link() with name override error = %v", err)
	}
	if !strings.Contains(msg, "custom-name") {
		t.Errorf("expected custom-name in message, got: %s", msg)
	}
}

func TestLink_TargetIsCanonicalAbsolute(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(sourceDir, "Agents", "Specs", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonical(t, repoPath), "specs")
	if got != want {
		t.Errorf("link target: got %q, want %q", got, want)
	}
}

func TestLink_RelativeRepoPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	t.Chdir(filepath.Dir(repoPath))
	if _, err := Link(sourceDir, filepath.Base(repoPath), "repo", false); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(sourceDir, "Agents", "Specs", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonical(t, repoPath), "specs")
	if got != want {
		t.Errorf("relative repo path must produce an absolute target: got %q, want %q", got, want)
	}
}

func TestLink_EmptyRepoPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := Link(t.TempDir(), "", "repo", false); err == nil {
		t.Fatal("expected error for empty repo path")
	}
}

func TestLink_DangerousNameRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "..", false); err == nil {
		t.Fatal("expected error for dangerous name override")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("nothing must be created on error, stat err: %v", err)
	}
}

func TestLink_RepoPathIsFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	file := filepath.Join(t.TempDir(), "repo")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(t.TempDir(), file, "repo", false); err == nil {
		t.Fatal("expected error for regular-file repo path")
	}
}

func TestLink_SpecsIsSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	outside := t.TempDir()
	if err := os.Remove(filepath.Join(repoPath, "specs")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repoPath, "specs")); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for symlinked specs directory")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
}

func TestLink_SpecsIsFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	if err := os.Remove(filepath.Join(repoPath, "specs")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "specs"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err == nil {
		t.Fatal("expected error for regular-file specs")
	}
}

func TestLink_BranchIntermediateSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	runGit(t, repoPath, "commit", "--allow-empty", "-m", "init")
	runGit(t, repoPath, "switch", "-c", "feature/foo")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repoPath, "specs", "feature")); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for symlinked branch path element")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
}

func TestLink_ReservedBranchNames(t *testing.T) {
	for _, branch := range []string{"index.md", "index.md/foo"} {
		t.Run(branch, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			repoPath := initTestRepo(t, "")
			runGit(t, repoPath, "commit", "--allow-empty", "-m", "init")
			runGit(t, repoPath, "switch", "-c", branch)

			_, err := Link(t.TempDir(), repoPath, "repo", false)
			if err == nil {
				t.Fatal("expected error for reserved branch name")
			}
			if !strings.Contains(err.Error(), "index.md") {
				t.Errorf("error should name the collision, got %v", err)
			}
		})
	}
}

func TestLink_RealDirectoryAtLinkDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	linkDir := filepath.Join(sourceDir, "Agents", "Specs", "repo")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkDir, "note.md"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Link(sourceDir, repoPath, "repo", false)
	if err == nil {
		t.Fatal("expected error for real directory at link path")
	}
	if !strings.Contains(err.Error(), "manually") {
		t.Errorf("error should advise manual migration, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(linkDir, "note.md")); err != nil {
		t.Errorf("existing notes must be untouched: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(repoPath, "specs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("nothing may be moved into the repo, got %d entries", len(entries))
	}
}

func TestLink_NoGitWithName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := t.TempDir() // not a git repo
	if err := os.MkdirAll(filepath.Join(repoPath, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "plain", false)
	if err != nil {
		t.Fatalf("Link() without git error = %v", err)
	}
	if !strings.Contains(msg, "Linked") {
		t.Errorf("expected 'Linked', got: %s", msg)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, "specs", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), repolink.GeneratedMarker+"\n") {
		t.Errorf("root index must carry the marker, got %q", data)
	}
}

func TestLink_BranchListedWithoutDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, "specs", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "main/index") {
		t.Errorf("root index must list the branch even without its directory, got %q", data)
	}
	if _, err := os.Lstat(filepath.Join(repoPath, "specs", "main")); !os.IsNotExist(err) {
		t.Errorf("branch directory must not be created, stat err: %v", err)
	}
}

func TestLink_AgentsSymlinkRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(sourceDir, "Agents")); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for symlinked Agents directory")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the linked-to directory must stay unchanged, got %d entries", len(entries))
	}
}

func TestLink_SourceInsideTargetRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := filepath.Join(repoPath, "specs", "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Link(sourceDir, repoPath, "demo", false)
	if err == nil {
		t.Fatal("expected overlap error")
	}
	if !strings.Contains(err.Error(), "overlap") {
		t.Errorf("expected overlap error, got %v", err)
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("nothing must be created on error, stat err: %v", err)
	}
}

func TestLink_JoinedOwnershipErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	specsDir := filepath.Join(repoPath, "specs")
	branchDir := filepath.Join(specsDir, "main")
	if err := os.MkdirAll(branchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "index.md"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(branchDir, "index.md"), []byte("also mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Link(sourceDir, repoPath, "repo", false)
	if err == nil {
		t.Fatal("expected ownership error")
	}
	specsCanon := filepath.Join(canonical(t, repoPath), "specs")
	for _, path := range []string{
		filepath.Join(specsCanon, "index.md"),
		filepath.Join(specsCanon, "main", "index.md"),
	} {
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error must list %s, got %v", path, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(specsDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mine\n" {
		t.Errorf("hand-written index must be preserved, got %q", data)
	}
}

func TestLink_ForceAdoptsLegacyIndex(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	legacy := filepath.Join(repoPath, "specs", "index.md")
	if err := os.WriteFile(legacy, []byte(".. toctree::\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for marker-less legacy index")
	}
	if _, err := Link(sourceDir, repoPath, "repo", true); err != nil {
		t.Fatalf("Link() with force error = %v", err)
	}
	data, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), repolink.GeneratedMarker+"\n") {
		t.Errorf("adopted index must carry the marker, got %q", data)
	}
}
