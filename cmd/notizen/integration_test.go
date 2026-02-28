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
	exec.Command(bin, "diary").Run()
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
