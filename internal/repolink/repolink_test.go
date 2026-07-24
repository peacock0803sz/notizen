package repolink

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tempDir returns a symlink-free temp directory. On macOS t.TempDir lives
// under /var, a symlink, which the physical path comparisons must not see;
// production callers hand PathsOverlap normalized paths for the same reason.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func readLink(t *testing.T, link string) string {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func caseInsensitiveFS(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "casecheck"), "")
	_, err := os.Stat(filepath.Join(dir, "CASECHECK"))
	return err == nil
}

func initGitRepo(t *testing.T, remote string) string {
	t.Helper()
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
	if remote != "" {
		run("git", "remote", "add", "origin", remote)
	}
	return dir
}

func TestResolveRepoName(t *testing.T) {
	repoWithRemote := initGitRepo(t, "https://github.com/user/myrepo.git")
	repoDegenerate := initGitRepo(t, ".git")
	repoNoRemote := initGitRepo(t, "")

	cases := []struct {
		name     string
		repoPath string
		override string
		want     string
		wantErr  bool
	}{
		{name: "override plain", repoPath: repoNoRemote, override: "custom", want: "custom"},
		{name: "override with path", repoPath: repoNoRemote, override: "sub/custom", want: "custom"},
		{name: "override dot", repoPath: repoNoRemote, override: ".", wantErr: true},
		{name: "override dotdot", repoPath: repoNoRemote, override: "..", wantErr: true},
		{name: "override root", repoPath: repoNoRemote, override: "/", wantErr: true},
		{name: "remote url", repoPath: repoWithRemote, want: "myrepo"},
		{name: "remote degenerates to empty", repoPath: repoDegenerate, wantErr: true},
		{name: "no remote no override", repoPath: repoNoRemote, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveRepoName(tc.repoPath, tc.override)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveUnder(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(t *testing.T, base string)
		components []string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "all real directories",
			setup:      func(t *testing.T, base string) { mkdirAll(t, filepath.Join(base, "docs", "superpowers")) },
			components: []string{"docs", "superpowers"},
			wantExists: true,
		},
		{
			name:       "first element missing",
			setup:      func(t *testing.T, base string) {},
			components: []string{"docs", "superpowers"},
			wantExists: false,
		},
		{
			name:       "last element missing",
			setup:      func(t *testing.T, base string) { mkdirAll(t, filepath.Join(base, "docs")) },
			components: []string{"docs", "superpowers"},
			wantExists: false,
		},
		{
			name: "intermediate symlink",
			setup: func(t *testing.T, base string) {
				mkdirAll(t, filepath.Join(base, "real"))
				symlink(t, filepath.Join(base, "real"), filepath.Join(base, "docs"))
			},
			components: []string{"docs", "superpowers"},
			wantErr:    true,
		},
		{
			name:       "intermediate regular file",
			setup:      func(t *testing.T, base string) { writeFile(t, filepath.Join(base, "docs"), "not a dir") },
			components: []string{"docs", "superpowers"},
			wantErr:    true,
		},
		{name: "empty component", components: []string{"docs", ""}, wantErr: true},
		{name: "dot component", components: []string{"."}, wantErr: true},
		{name: "dotdot component", components: []string{".."}, wantErr: true},
		{name: "absolute component", components: []string{"/etc"}, wantErr: true},
		{name: "component with separator", components: []string{"a/b"}, wantErr: true},
		{name: "invalid after missing", components: []string{"missing", "..", ".."}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := tempDir(t)
			if tc.setup != nil {
				tc.setup(t, base)
			}
			path, exists, err := ResolveUnder(base, tc.components...)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got path %q exists %v", path, exists)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(append([]string{base}, tc.components...)...)
			if path != want {
				t.Errorf("path: got %q, want full candidate %q", path, want)
			}
			if exists != tc.wantExists {
				t.Errorf("exists: got %v, want %v", exists, tc.wantExists)
			}
		})
	}
}

func TestResolveUnder_CandidateUsableAsDanglingTarget(t *testing.T) {
	base := tempDir(t)
	candidate, exists, err := ResolveUnder(base, "docs", "superpowers")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected candidate to be absent")
	}
	link := filepath.Join(tempDir(t), "link")
	symlink(t, candidate, link)
	if got := readLink(t, link); got != filepath.Join(base, "docs", "superpowers") {
		t.Errorf("dangling link target: got %q", got)
	}
}

func TestEnsureSymlink_CreatesFromCleanRoot(t *testing.T) {
	base := tempDir(t)
	target := filepath.Join(base, "repo", "docs")
	mkdirAll(t, target)
	linkDir := filepath.Join(base, "source", "Agents", "Superpowers", "demo")

	action, err := EnsureSymlink(linkDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionCreated {
		t.Errorf("action: got %q, want %q", action, ActionCreated)
	}
	info, err := os.Lstat(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected a symlink")
	}
	if got := readLink(t, linkDir); got != target {
		t.Errorf("target: got %q, want %q", got, target)
	}
}

func TestEnsureSymlink_SkipsCorrect(t *testing.T) {
	base := tempDir(t)
	target := filepath.Join(base, "target")
	linkDir := filepath.Join(base, "link")
	symlink(t, target, linkDir)

	action, err := EnsureSymlink(linkDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionSkipped {
		t.Errorf("action: got %q, want %q", action, ActionSkipped)
	}
}

func TestEnsureSymlink_UpdatesWrongTarget(t *testing.T) {
	base := tempDir(t)
	target := filepath.Join(base, "target")
	parent := filepath.Join(base, "parent")
	mkdirAll(t, parent)
	linkDir := filepath.Join(parent, "link")
	symlink(t, "/wrong/target", linkDir)

	action, err := EnsureSymlink(linkDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionUpdated {
		t.Errorf("action: got %q, want %q", action, ActionUpdated)
	}
	if got := readLink(t, linkDir); got != target {
		t.Errorf("target: got %q, want %q", got, target)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected only the link in %s, got %d entries", parent, len(entries))
	}
}

func TestEnsureSymlink_KeepsOldLinkWhenTempCreationFails(t *testing.T) {
	base := tempDir(t)
	parent := filepath.Join(base, "parent")
	mkdirAll(t, parent)
	linkDir := filepath.Join(parent, "link")
	symlink(t, "/wrong/target", linkDir)
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	if _, err := EnsureSymlink(linkDir, filepath.Join(base, "target")); err == nil {
		t.Fatal("expected error")
	}
	if got := readLink(t, linkDir); got != "/wrong/target" {
		t.Errorf("old link should survive, got %q", got)
	}
}

func TestEnsureSymlink_KeepsOldLinkWhenSymlinkFails(t *testing.T) {
	base := tempDir(t)
	parent := filepath.Join(base, "parent")
	mkdirAll(t, parent)
	linkDir := filepath.Join(parent, "link")
	symlink(t, "/wrong/target", linkDir)

	// A NUL byte makes os.Symlink itself fail, after the temp name was
	// already reserved and released.
	if _, err := EnsureSymlink(linkDir, "bad\x00target"); err == nil {
		t.Fatal("expected error")
	}
	if got := readLink(t, linkDir); got != "/wrong/target" {
		t.Errorf("old link should survive, got %q", got)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected only the link in %s, got %d entries", parent, len(entries))
	}
}

func TestEnsureSymlink_RealDirectoryError(t *testing.T) {
	base := tempDir(t)
	target := filepath.Join(base, "target")
	linkDir := filepath.Join(base, "link")
	mkdirAll(t, linkDir)
	writeFile(t, filepath.Join(linkDir, "note.md"), "keep me")

	_, err := EnsureSymlink(linkDir, target)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "manually") {
		t.Errorf("error should advise manual migration, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(linkDir, "note.md")); err != nil {
		t.Errorf("directory contents must be untouched: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("target must not be created, stat err: %v", err)
	}
}

func TestEnsureSymlink_RegularFileError(t *testing.T) {
	base := tempDir(t)
	linkDir := filepath.Join(base, "link")
	writeFile(t, linkDir, "a file")

	if _, err := EnsureSymlink(linkDir, filepath.Join(base, "target")); err == nil {
		t.Fatal("expected error")
	}
}

func TestPathsOverlap(t *testing.T) {
	insensitive := caseInsensitiveFS(t)
	cases := []struct {
		name string
		skip func() bool
		ab   func(t *testing.T, base string) (string, string)
		want bool
	}{
		{
			name: "same existing path",
			ab: func(t *testing.T, base string) (string, string) {
				mkdirAll(t, filepath.Join(base, "d"))
				return filepath.Join(base, "d"), filepath.Join(base, "d")
			},
			want: true,
		},
		{
			name: "linkDir under target",
			ab: func(t *testing.T, base string) (string, string) {
				target := filepath.Join(base, "specs")
				mkdirAll(t, target)
				return filepath.Join(target, "source", "Agents", "Specs", "demo"), target
			},
			want: true,
		},
		{
			name: "target under linkDir",
			ab: func(t *testing.T, base string) (string, string) {
				linkDir := filepath.Join(base, "link")
				mkdirAll(t, linkDir)
				return linkDir, filepath.Join(linkDir, "sub", "target")
			},
			want: true,
		},
		{
			name: "existing siblings",
			ab: func(t *testing.T, base string) (string, string) {
				mkdirAll(t, filepath.Join(base, "x"))
				mkdirAll(t, filepath.Join(base, "y"))
				return filepath.Join(base, "x"), filepath.Join(base, "y")
			},
			want: false,
		},
		{
			name: "missing siblings",
			ab: func(t *testing.T, base string) (string, string) {
				return filepath.Join(base, "x", "deep"), filepath.Join(base, "y", "deep")
			},
			want: false,
		},
		{
			name: "containment across existing intermediate directories",
			ab: func(t *testing.T, base string) (string, string) {
				target := filepath.Join(base, "specs")
				mkdirAll(t, filepath.Join(target, "source", "Agents"))
				return filepath.Join(target, "source", "Agents", "Specs", "demo"), target
			},
			want: true,
		},
		{
			name: "correct symlink is not overlap",
			ab: func(t *testing.T, base string) (string, string) {
				target := filepath.Join(base, "target")
				mkdirAll(t, target)
				link := filepath.Join(base, "link")
				symlink(t, target, link)
				return link, target
			},
			want: false,
		},
		{
			name: "wrong-target symlink is not overlap",
			ab: func(t *testing.T, base string) (string, string) {
				target := filepath.Join(base, "target")
				mkdirAll(t, target)
				link := filepath.Join(base, "link")
				symlink(t, "/somewhere/else", link)
				return link, target
			},
			want: false,
		},
		{
			name: "symlink placed inside target",
			ab: func(t *testing.T, base string) (string, string) {
				target := filepath.Join(base, "target")
				mkdirAll(t, target)
				link := filepath.Join(target, "link")
				symlink(t, "/somewhere/else", link)
				return link, target
			},
			want: true,
		},
		{
			name: "case-variant spelling of the same entity",
			skip: func() bool { return !insensitive },
			ab: func(t *testing.T, base string) (string, string) {
				mkdirAll(t, filepath.Join(base, "Repo"))
				return filepath.Join(base, "repo", "docs"), filepath.Join(base, "Repo")
			},
			want: true,
		},
		{
			name: "unresolved elements match case-insensitively",
			ab: func(t *testing.T, base string) (string, string) {
				mkdirAll(t, filepath.Join(base, "zone"))
				return filepath.Join(base, "zone", "Specs", "x"), filepath.Join(base, "zone", "specs")
			},
			want: true,
		},
		{
			name: "existing distinct case-variant trees stay separate",
			skip: func() bool { return insensitive },
			ab: func(t *testing.T, base string) (string, string) {
				mkdirAll(t, filepath.Join(base, "Tree", "Agents"))
				mkdirAll(t, filepath.Join(base, "tree", "Agents"))
				return filepath.Join(base, "Tree", "Agents", "demo"), filepath.Join(base, "tree", "Agents", "demo", "repo")
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip != nil && tc.skip() {
				t.Skip("not applicable on this filesystem")
			}
			base := tempDir(t)
			a, b := tc.ab(t, base)
			got, err := PathsOverlap(a, b)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("PathsOverlap(%q, %q): got %v, want %v", a, b, got, tc.want)
			}
		})
	}
}

func TestPathsOverlap_PermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	base := tempDir(t)
	locked := filepath.Join(base, "locked")
	mkdirAll(t, filepath.Join(locked, "inner"))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	if _, err := PathsOverlap(filepath.Join(locked, "inner", "x"), filepath.Join(base, "other")); err == nil {
		t.Fatal("expected permission error to propagate")
	}
}

func TestListMarkdownFiles(t *testing.T) {
	dir := tempDir(t)
	writeFile(t, filepath.Join(dir, "a.md"), "")
	writeFile(t, filepath.Join(dir, "b.md"), "")
	writeFile(t, filepath.Join(dir, "index.md"), "")
	writeFile(t, filepath.Join(dir, "note.txt"), "")
	mkdirAll(t, filepath.Join(dir, "subdir"))

	files, err := ListMarkdownFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0] != "a" || files[1] != "b" {
		t.Errorf("got %v, want [a b]", files)
	}
}

func TestListMarkdownFiles_IndexSpelling(t *testing.T) {
	dir := tempDir(t)
	if caseInsensitiveFS(t) {
		// Index.md is physically the file WriteIndex writes; it must not
		// list itself.
		writeFile(t, filepath.Join(dir, "Index.md"), "")
		files, err := ListMarkdownFiles(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Errorf("got %v, want no entries", files)
		}
		return
	}
	// On a case-sensitive filesystem Index.md is a distinct file and stays
	// listed.
	writeFile(t, filepath.Join(dir, "index.md"), "")
	writeFile(t, filepath.Join(dir, "Index.md"), "")
	files, err := ListMarkdownFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "Index" {
		t.Errorf("got %v, want [Index]", files)
	}
}

func TestWriteIndex_NewFile(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "index.md")

	if err := WriteIndex(path, []byte(".. toctree::\n"), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := GeneratedMarker + "\n\n.. toctree::\n"
	if string(data) != want {
		t.Errorf("content: got %q, want %q", data, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode: got %v, want 0644", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %d entries", len(entries))
	}
}

func TestWriteIndex_UpdatesGeneratedKeepingMode(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte(GeneratedMarker+"\n\nold\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := WriteIndex(path, []byte("new\n"), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "new\n") {
		t.Errorf("content not updated: %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode: got %v, want original 0600", info.Mode().Perm())
	}
}

func TestWriteIndex_RefusesHandWritten(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "index.md")
	writeFile(t, path, "my precious notes\n")

	err := WriteIndex(path, []byte("generated\n"), false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "my precious notes\n" {
		t.Errorf("hand-written content must be preserved, got %q", data)
	}
}

func TestWriteIndex_ForceAdoptsHandWritten(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "index.md")
	writeFile(t, path, "my precious notes\n")

	if err := WriteIndex(path, []byte("generated\n"), true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), GeneratedMarker+"\n") {
		t.Errorf("adopted file must carry the marker, got %q", data)
	}
}

func TestWriteIndex_RefusesSymlink(t *testing.T) {
	dir := tempDir(t)
	real := filepath.Join(dir, "real.md")
	writeFile(t, real, GeneratedMarker+"\n")
	path := filepath.Join(dir, "index.md")
	symlink(t, real, path)

	if err := WriteIndex(path, []byte("x\n"), false); err == nil {
		t.Fatal("expected error for symlink destination")
	}
	if err := WriteIndex(path, []byte("x\n"), true); err == nil {
		t.Fatal("force must not bypass the non-regular-file check")
	}
}

func TestWriteIndex_NoTempResidueOnFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	dir := tempDir(t)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := WriteIndex(filepath.Join(dir, "index.md"), []byte("x\n"), false); err == nil {
		t.Fatal("expected error")
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("temp files left behind: %v", entries)
	}
}
