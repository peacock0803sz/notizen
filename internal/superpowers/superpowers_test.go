package superpowers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peacock0803sz/notizen/internal/repolink"
)

// initTestRepo creates a git repo with an optional remote and no docs yet.
func initTestRepo(t *testing.T, remote string) string {
	t.Helper()
	// Isolate git from the developer's global/system config (e.g. signing).
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
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
	return dir
}

// addSection creates docs/superpowers/<section> with the given stub files.
func addSection(t *testing.T, repoPath, section string, files ...string) {
	t.Helper()
	dir := filepath.Join(repoPath, "docs", "superpowers", section)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("stub\n"), 0o644); err != nil {
			t.Fatal(err)
		}
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

func caseInsensitiveFS(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "casecheck"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := os.Stat(filepath.Join(dir, "CASECHECK"))
	return err == nil
}

func TestLink_NewLink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "https://github.com/user/myrepo.git")
	addSection(t, repoPath, "specs", "2026-07-01-foo-design.md")
	addSection(t, repoPath, "plans", "2026-07-02-foo.md")
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "myrepo", false)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if !strings.Contains(msg, "Linked") {
		t.Errorf("expected 'Linked' in message, got: %s", msg)
	}

	linkDir := filepath.Join(sourceDir, "Agents", "Superpowers", "myrepo")
	info, err := os.Lstat(linkDir)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular dir")
	}
	got, err := os.Readlink(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonical(t, repoPath), "docs", "superpowers")
	if got != want {
		t.Errorf("link target: got %q, want %q", got, want)
	}
}

func TestLink_RelativeRepoPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	sourceDir := t.TempDir()

	t.Chdir(filepath.Dir(repoPath))
	if _, err := Link(sourceDir, filepath.Base(repoPath), "repo", false); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(sourceDir, "Agents", "Superpowers", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonical(t, repoPath), "docs", "superpowers")
	if got != want {
		t.Errorf("relative repo path must produce an absolute target: got %q, want %q", got, want)
	}
}

func TestLink_RelativeSourceDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	parent := t.TempDir()
	sourceDir := filepath.Join(parent, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(parent)
	if _, err := Link("source", repoPath, "repo", false); err != nil {
		t.Fatalf("Link() with relative source error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents", "Superpowers", "repo")); err != nil {
		t.Errorf("link not created under the absolutized source: %v", err)
	}
}

func TestLink_EmptyRepoPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := Link(t.TempDir(), "", "repo", false); err == nil {
		t.Fatal("expected error for empty repo path")
	}
}

func TestLink_AlreadyCorrect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
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
	linkDir := filepath.Join(sourceDir, "Agents", "Superpowers", "repo")
	if err := os.MkdirAll(filepath.Dir(linkDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/wrong/target", linkDir); err != nil {
		t.Fatal(err)
	}

	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	msg, err := Link(sourceDir, repoPath, "repo", false)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if !strings.Contains(msg, "Updated") {
		t.Errorf("expected 'Updated', got: %s", msg)
	}
	got, err := os.Readlink(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonical(t, repoPath), "docs", "superpowers")
	if got != want {
		t.Errorf("link target: got %q, want %q", got, want)
	}
}

func TestLink_RepoPathNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := Link(t.TempDir(), "/nonexistent/path", "repo", false); err == nil {
		t.Fatal("expected error for nonexistent repo path")
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

func TestLink_NoGitRemoteWithoutName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")

	if _, err := Link(t.TempDir(), repoPath, "", false); err == nil {
		t.Fatal("expected error when no git remote and no --name")
	}
}

func TestLink_NameOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "custom-name", false)
	if err != nil {
		t.Fatalf("Link() with name override error = %v", err)
	}
	if !strings.Contains(msg, "custom-name") {
		t.Errorf("expected custom-name in message, got: %s", msg)
	}
}

func TestLink_RootIndexListsBothSections(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs", "a-design.md")
	addSection(t, repoPath, "plans", "a.md")

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, "docs", "superpowers", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	specsAt := strings.Index(content, "specs/index")
	plansAt := strings.Index(content, "plans/index")
	if specsAt < 0 || plansAt < 0 {
		t.Fatalf("root index must list both sections, got %q", content)
	}
	if specsAt > plansAt {
		t.Errorf("specs must come before plans, got %q", content)
	}
}

func TestLink_RootIndexOnlyPlans(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "plans", "a.md")

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	docsDir := filepath.Join(repoPath, "docs", "superpowers")
	data, err := os.ReadFile(filepath.Join(docsDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "specs/index") {
		t.Errorf("missing section must not be listed, got %q", data)
	}
	if !strings.Contains(string(data), "plans/index") {
		t.Errorf("existing section must be listed, got %q", data)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "plans", "index.md")); err != nil {
		t.Errorf("plans index must be written: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(docsDir, "specs")); !os.IsNotExist(err) {
		t.Errorf("missing section directory must not be created, stat err: %v", err)
	}
}

func TestLink_NoSectionsWritesEmptyRootIndex(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	docsDir := filepath.Join(repoPath, "docs", "superpowers")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(docsDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), ".. toctree::") {
		t.Errorf("root index must contain an empty toctree, got %q", data)
	}
	if strings.Contains(string(data), "/index") {
		t.Errorf("empty toctree must have no entries, got %q", data)
	}
}

func TestLink_SectionIndexListsFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs", "2026-07-01-foo-design.md")

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, "docs", "superpowers", "specs", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "2026-07-01-foo-design") {
		t.Errorf("section index must list the document, got %q", data)
	}
	if strings.Contains(string(data), "   index\n") {
		t.Errorf("section index must not list itself, got %q", data)
	}
}

func TestLink_EmptySectionWritesEmptyToctree(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, "docs", "superpowers", "specs", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), ".. toctree::") {
		t.Errorf("empty section index must contain a toctree, got %q", data)
	}
}

func TestLink_DanglingWhenDocsMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()

	msg, err := Link(sourceDir, repoPath, "repo", false)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if !strings.Contains(msg, "Linked") {
		t.Errorf("expected 'Linked', got: %s", msg)
	}
	linkDir := filepath.Join(sourceDir, "Agents", "Superpowers", "repo")
	info, err := os.Lstat(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink")
	}
	if _, err := os.Lstat(filepath.Join(repoPath, "docs")); !os.IsNotExist(err) {
		t.Errorf("docs must not be created, stat err: %v", err)
	}
}

func TestLink_DocsSuperpowersIsFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoPath, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "docs", "superpowers"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for regular-file docs/superpowers")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
}

func TestLink_DocsSuperpowersIsSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	sourceDir := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoPath, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repoPath, "docs", "superpowers")); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for symlinked docs/superpowers")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
}

func TestLink_DocsItselfIsSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "superpowers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repoPath, "docs")); err != nil {
		t.Fatal(err)
	}

	if _, err := Link(t.TempDir(), repoPath, "repo", false); err == nil {
		t.Fatal("expected error for symlinked docs")
	}
	if _, err := os.Lstat(filepath.Join(outside, "superpowers", "index.md")); !os.IsNotExist(err) {
		t.Errorf("nothing may be written outside the repo, stat err: %v", err)
	}
}

func TestLink_AgentsSymlinkRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
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
	addSection(t, repoPath, "specs")
	sourceDir := filepath.Join(repoPath, "docs", "superpowers", "source")
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

func TestLink_SourceInsideTargetRejectedCaseVariant(t *testing.T) {
	if !caseInsensitiveFS(t) {
		t.Skip("requires a case-insensitive filesystem")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	if err := os.MkdirAll(filepath.Join(repoPath, "docs", "superpowers", "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The same directory spelled with a different case must still be
	// detected as living inside the target.
	sourceDir := filepath.Join(repoPath, "docs", "SUPERPOWERS", "source")

	_, err := Link(sourceDir, repoPath, "demo", false)
	if err == nil {
		t.Fatal("expected overlap error for case-variant spelling")
	}
	if !strings.Contains(err.Error(), "overlap") {
		t.Errorf("expected overlap error, got %v", err)
	}
}

func TestLink_SectionIsFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "plans")
	docsDir := filepath.Join(repoPath, "docs", "superpowers")
	if err := os.WriteFile(filepath.Join(docsDir, "specs"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for regular-file section")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(docsDir, "index.md")); !os.IsNotExist(err) {
		t.Errorf("no index may be written on error, stat err: %v", err)
	}
}

func TestLink_SectionIsSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "plans")
	docsDir := filepath.Join(repoPath, "docs", "superpowers")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(docsDir, "specs")); err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for symlinked section")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(outside, "index.md")); !os.IsNotExist(err) {
		t.Errorf("nothing may be written outside the repo, stat err: %v", err)
	}
}

func TestLink_JoinedOwnershipErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	docsDir := filepath.Join(repoPath, "docs", "superpowers")
	if err := os.WriteFile(filepath.Join(docsDir, "index.md"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "specs", "index.md"), []byte("also mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()

	_, err := Link(sourceDir, repoPath, "repo", false)
	if err == nil {
		t.Fatal("expected ownership error")
	}
	docsCanon := filepath.Join(canonical(t, repoPath), "docs", "superpowers")
	for _, path := range []string{
		filepath.Join(docsCanon, "index.md"),
		filepath.Join(docsCanon, "specs", "index.md"),
	} {
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error must list %s, got %v", path, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(docsDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mine\n" {
		t.Errorf("hand-written index must be preserved, got %q", data)
	}

	if _, err := Link(sourceDir, repoPath, "repo", true); err != nil {
		t.Fatalf("Link() with force error = %v", err)
	}
	data, err = os.ReadFile(filepath.Join(docsDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), repolink.GeneratedMarker+"\n") {
		t.Errorf("adopted index must carry the marker, got %q", data)
	}
}

func TestLink_BrokenCustomTemplate(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	override := filepath.Join(cfgDir, "notizen", "templates", "superpowers")
	if err := os.MkdirAll(override, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(override, "index.md.tmpl"), []byte("{{ range .Sections }}"), 0o644); err != nil {
		t.Fatal(err)
	}
	repoPath := initTestRepo(t, "")
	addSection(t, repoPath, "specs")
	sourceDir := t.TempDir()

	if _, err := Link(sourceDir, repoPath, "repo", false); err == nil {
		t.Fatal("expected error for broken custom template")
	}
	if _, err := os.Lstat(filepath.Join(sourceDir, "Agents")); !os.IsNotExist(err) {
		t.Errorf("link must not be created on error, stat err: %v", err)
	}
}
