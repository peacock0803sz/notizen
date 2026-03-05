package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles the CLI binary into a temp directory for end-to-end tests.
func buildBinary(t *testing.T) string {
	t.Helper()
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
	toml := "root = \"" + customRoot + "\"\n"
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
	toml := "root = \"" + deepRoot + "\"\n"
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
