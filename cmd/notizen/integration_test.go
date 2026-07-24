package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peacock0803sz/notizen/internal/repolink"
)

// buildBinary compiles the CLI binary into a temp directory for end-to-end tests.
func buildBinary(t *testing.T) string {
	t.Helper()
	// Neutralize NOTIZEN_ROOT from the developer's environment so the binary
	// under test resolves its root from the per-test HOME instead.
	t.Setenv("NOTIZEN_ROOT", "")
	bin := filepath.Join(t.TempDir(), "notizen.out")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(".") // cmd/notizen
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s", out)
	}
	return bin
}

func TestIntegration_DiaryCreatesFile(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	// stdout must contain the created file path
	path := strings.TrimSpace(string(out))
	if path == "" {
		t.Fatal("expected file path on stdout")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("created file not found: %v", err)
	}
}

func TestIntegration_DiaryExitCode0(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	cmd := exec.Command(bin, "diary")
	if err := cmd.Run(); err != nil {
		t.Fatalf("diary should exit 0, got: %v", err)
	}
}

func TestIntegration_DiaryDuplicate(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	// First call creates the entry
	if err := exec.Command(bin, "diary").Run(); err != nil {
		t.Fatalf("first diary call failed: %v", err)
	}
	// Second call must exit 0 and print "Already exists"
	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("second diary call should exit 0: %v", err)
	}
	if !strings.Contains(string(out), "Already exists") {
		t.Errorf("expected 'Already exists' on stdout, got: %s", out)
	}
}

func TestIntegration_DiaryConfigRoot(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	customRoot := filepath.Join(tmpHome, "custom-notes")
	t.Setenv("HOME", tmpHome)

	// Write config.toml with custom root
	cfgDir := filepath.Join(tmpHome, ".config", "notizen")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := fmt.Sprintf("root = %q\n", customRoot)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", "")

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, customRoot) {
		t.Errorf("diary path = %q, want prefix %q", path, customRoot)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("created file not found: %v", err)
	}
}

func TestIntegration_DiaryDefaultBackwardCompat(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", "")

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	defaultRoot := filepath.Join(tmpHome, ".notizen")
	if !strings.HasPrefix(path, defaultRoot) {
		t.Errorf("diary path = %q, want prefix %q (default)", path, defaultRoot)
	}
}

func TestIntegration_DiaryConfigRootAutoCreatesDeepPath(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	deepRoot := filepath.Join(tmpHome, "a", "b", "notes")
	t.Setenv("HOME", tmpHome)

	cfgDir := filepath.Join(tmpHome, ".config", "notizen")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := fmt.Sprintf("root = %q\n", deepRoot)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", "")

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, deepRoot) {
		t.Errorf("diary path = %q, want prefix %q", path, deepRoot)
	}
	// Verify both root and root/source directories were created
	sourceDir := filepath.Join(deepRoot, "source")
	if _, err := os.Stat(sourceDir); err != nil {
		t.Errorf("source directory not created: %v", err)
	}
}

// --- US2: Environment variable override tests ---

func TestIntegration_DiaryEnvRoot(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	envRoot := filepath.Join(tmpHome, "env-notes")
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", envRoot)

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, envRoot) {
		t.Errorf("diary path = %q, want prefix %q", path, envRoot)
	}
}

func TestIntegration_DiaryEnvOverridesConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	configRoot := filepath.Join(tmpHome, "config-notes")
	envRoot := filepath.Join(tmpHome, "env-notes")
	t.Setenv("HOME", tmpHome)

	cfgDir := filepath.Join(tmpHome, ".config", "notizen")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tomlContent := fmt.Sprintf("root = %q\n", configRoot)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", envRoot)

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, envRoot) {
		t.Errorf("diary path = %q, want prefix %q (env should win over config)", path, envRoot)
	}
}

func TestIntegration_DiaryEnvRootAutoCreatesDeepPath(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	deepRoot := filepath.Join(tmpHome, "x", "y", "env-notes")
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", deepRoot)

	cmd := exec.Command(bin, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, deepRoot) {
		t.Errorf("diary path = %q, want prefix %q", path, deepRoot)
	}
	sourceDir := filepath.Join(deepRoot, "source")
	if _, err := os.Stat(sourceDir); err != nil {
		t.Errorf("source directory not created: %v", err)
	}
}

func TestIntegration_DiaryEnvRootPermissionDenied(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	readOnlyDir := filepath.Join(tmpHome, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(readOnlyDir, 0o755); err != nil {
			t.Log(err)
		}
	})

	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", filepath.Join(readOnlyDir, "notes"))

	cmd := exec.Command(bin, "diary")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for permission-denied root")
	}
	output := string(out)
	if !strings.Contains(output, "readonly") {
		t.Errorf("stderr should mention the path, got: %s", output)
	}
}

// --- US3: CLI flag override tests ---

func TestIntegration_DiaryFlagRoot(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	flagRoot := filepath.Join(tmpHome, "flag-notes")
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", "")

	cmd := exec.Command(bin, "--root", flagRoot, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, flagRoot) {
		t.Errorf("diary path = %q, want prefix %q", path, flagRoot)
	}
}

func TestIntegration_DiaryFlagOverridesAll(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	configRoot := filepath.Join(tmpHome, "config-notes")
	envRoot := filepath.Join(tmpHome, "env-notes")
	flagRoot := filepath.Join(tmpHome, "flag-notes")
	t.Setenv("HOME", tmpHome)

	cfgDir := filepath.Join(tmpHome, ".config", "notizen")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tomlContent := fmt.Sprintf("root = %q\n", configRoot)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", envRoot)

	cmd := exec.Command(bin, "--root", flagRoot, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !strings.HasPrefix(path, flagRoot) {
		t.Errorf("diary path = %q, want prefix %q (flag should win over env and config)", path, flagRoot)
	}
}

func TestIntegration_DiaryFlagRootSymlink(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	realDir := filepath.Join(tmpHome, "real-notes")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(tmpHome, "link-notes")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("NOTIZEN_ROOT", "")

	cmd := exec.Command(bin, "--root", linkDir, "diary")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("diary command failed: %v, output: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	// File should exist regardless of whether path goes through symlink or real dir
	if _, err := os.Stat(path); err != nil {
		t.Errorf("created file not found: %v", err)
	}
	// source/ should exist under the real directory
	sourceDir := filepath.Join(realDir, "source")
	if _, err := os.Stat(sourceDir); err != nil {
		t.Errorf("source directory not found under real dir: %v", err)
	}
}

func TestIntegration_VersionFlag(t *testing.T) {
	bin := buildBinary(t)
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	if !strings.Contains(string(out), "notizen") {
		t.Errorf("expected 'notizen' in version output, got: %s", out)
	}
}

func TestIntegration_UnknownCommand_Exit1(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "nonexistent-command")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected exit code 1 for unknown command")
	}
}

func TestIntegration_SpecsNoArgs_Exit1(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cmd := exec.Command(bin, "specs")
	if err := cmd.Run(); err == nil {
		t.Fatal("specs without args should exit 1")
	}
}

func TestIntegration_SuperpowersNoArgs_Exit1(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cmd := exec.Command(bin, "superpowers")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("superpowers without args should fail with an exit error, got %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}
	if stderr.Len() == 0 {
		t.Error("expected a usage error on stderr")
	}
}

func TestIntegration_SuperpowersLinksRepo(t *testing.T) {
	bin := buildBinary(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	root := filepath.Join(tmpHome, "notes")

	// --name makes git unnecessary; a plain directory tree is enough.
	repo := t.TempDir()
	specsDir := filepath.Join(repo, "docs", "superpowers", "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "demo-design.md"), []byte("stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--root", root, "superpowers", repo, "-n", "demo")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("superpowers command failed: %v, output: %s", err, out)
	}
	if !strings.Contains(string(out), "Linked") {
		t.Errorf("expected 'Linked' on stdout, got: %s", out)
	}
	linkDir := filepath.Join(root, "source", "Agents", "Superpowers", "demo")
	info, err := os.Lstat(linkDir)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink")
	}
	data, err := os.ReadFile(filepath.Join(repo, "docs", "superpowers", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), repolink.GeneratedMarker+"\n") {
		t.Errorf("generated index must carry the marker, got %q", data)
	}
}
