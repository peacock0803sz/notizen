package config

import (
	"os"
	"path/filepath"
	"testing"
)

// clearEnv resets all NOTIZEN_REMOTE_* env vars before a test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"NOTIZEN_REMOTE_HOST", "NOTIZEN_REMOTE_USER",
		"NOTIZEN_REMOTE_PORT", "NOTIZEN_REMOTE_KEY", "NOTIZEN_REMOTE_PATH",
	} {
		t.Setenv(key, "")
	}
}

func writeToml(t *testing.T, dir, content string) {
	t.Helper()
	cfgDir := filepath.Join(dir, "notizen")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_EnvVarPrecedenceOverTOML(t *testing.T) {
	clearEnv(t)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	writeToml(t, tmpDir, "[remote]\nhost = \"toml-host\"\n")
	t.Setenv("NOTIZEN_REMOTE_HOST", "env-host")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Host != "env-host" {
		t.Errorf("Host = %q, want env-host", cfg.Host)
	}
}

func TestLoad_MissingHostError(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing Host")
	}
}

func TestLoad_InvalidPortError(t *testing.T) {
	clearEnv(t)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	writeToml(t, tmpDir, "[remote]\nhost = \"h\"\nport = 99999\n")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoad_DefaultPort22(t *testing.T) {
	clearEnv(t)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	writeToml(t, tmpDir, "[remote]\nhost = \"h\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != 22 {
		t.Errorf("Port = %d, want 22", cfg.Port)
	}
}

func TestLoad_KeyPathValidation(t *testing.T) {
	clearEnv(t)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Nonexistent key path must return an error
	writeToml(t, tmpDir, "[remote]\nhost = \"h\"\nkey = \"/nonexistent/key\"\n")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for nonexistent key path")
	}

	// Valid key path must succeed
	keyFile := filepath.Join(tmpDir, "id_ed25519")
	if err := os.WriteFile(keyFile, []byte("fake-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeToml(t, tmpDir, "[remote]\nhost = \"h\"\nkey = \""+keyFile+"\"\n")
	if _, err := Load(); err != nil {
		t.Errorf("Load() with valid key path error = %v", err)
	}
}
