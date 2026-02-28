package tmpl

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_EmbeddedFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty dir with no templates

	tmpl, err := Load("diary/entry.md.tmpl")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ FormattedDate string }{"2026/02/28 (Sat)"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := buf.String(); got == "" {
		t.Error("expected non-empty output from embedded template")
	}
}

func TestLoad_FilesystemOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Place a custom template at the override path
	overrideDir := filepath.Join(tmpDir, "notizen", "templates", "diary")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customContent := "CUSTOM: {{ .FormattedDate }}"
	overrideFile := filepath.Join(overrideDir, "entry.md.tmpl")
	if err := os.WriteFile(overrideFile, []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpl, err := Load("diary/entry.md.tmpl")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ FormattedDate string }{"2026/02/28 (Sat)"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := buf.String(); got != "CUSTOM: 2026/02/28 (Sat)" {
		t.Errorf("expected custom template output, got %q", got)
	}
}

func TestLoad_MissingTemplate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, err := Load("diary/nonexistent.md.tmpl")
	if err == nil {
		t.Fatal("expected error for missing template, got nil")
	}
}
