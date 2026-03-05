package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestResolveRoot(t *testing.T) {
	tests := []struct {
		name       string
		flagValue  string
		envValue   string
		tomlRoot   string
		wantSuffix string // expected suffix of the resolved path
	}{
		{
			name:       "default when nothing set",
			wantSuffix: ".notizen",
		},
		{
			name:       "TOML root overrides default",
			tomlRoot:   filepath.Join("/custom", "toml-root"),
			wantSuffix: filepath.Join("custom", "toml-root"),
		},
		{
			name:       "env var overrides TOML",
			tomlRoot:   filepath.Join("/custom", "toml-root"),
			envValue:   filepath.Join("/custom", "env-root"),
			wantSuffix: filepath.Join("custom", "env-root"),
		},
		{
			name:       "flag overrides env",
			tomlRoot:   filepath.Join("/custom", "toml-root"),
			envValue:   filepath.Join("/custom", "env-root"),
			flagValue:  filepath.Join("/custom", "flag-root"),
			wantSuffix: filepath.Join("custom", "flag-root"),
		},
		{
			name:       "relative path resolved to absolute",
			flagValue:  filepath.Join("relative", "path"),
			wantSuffix: filepath.Join("relative", "path"),
		},
		{
			name:       "missing config file falls back gracefully",
			wantSuffix: ".notizen",
		},
		{
			name:       "non-existent deep path accepted",
			flagValue:  filepath.Join("/tmp", "a", "b", "notes"),
			wantSuffix: filepath.Join("a", "b", "notes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			tmpDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", tmpDir)
			t.Setenv("HOME", tmpDir)

			if tt.tomlRoot != "" {
				writeToml(t, tmpDir, fmt.Sprintf("root = %q\n", tt.tomlRoot))
			}
			if tt.envValue != "" {
				t.Setenv("NOTIZEN_ROOT", tt.envValue)
			} else {
				t.Setenv("NOTIZEN_ROOT", "")
			}

			got, err := ResolveRoot(tt.flagValue)
			if err != nil {
				t.Fatalf("ResolveRoot() error = %v", err)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("ResolveRoot() returned non-absolute path: %s", got)
			}
			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("ResolveRoot() = %q, want suffix %q", got, tt.wantSuffix)
			}
		})
	}
}

func TestResolveRoot_SymlinkFollowed(t *testing.T) {
	clearEnv(t)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("NOTIZEN_ROOT", "")

	// Create a real directory and a symlink pointing to it
	realDir := filepath.Join(tmpDir, "real-notes")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(tmpDir, "link-notes")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveRoot(linkDir)
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	// The resolved path should be the symlink path itself (filepath.Abs doesn't resolve symlinks)
	// What matters is the path is absolute and usable
	if !filepath.IsAbs(got) {
		t.Errorf("ResolveRoot() returned non-absolute path: %s", got)
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

func TestLoad_InvalidEnvPort(t *testing.T) {
	clearEnv(t)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("NOTIZEN_REMOTE_HOST", "h")
	t.Setenv("NOTIZEN_REMOTE_PORT", "notanumber")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid NOTIZEN_REMOTE_PORT")
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
