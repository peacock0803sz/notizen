package diary

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testTime = time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

func TestCreate_NewEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sourceDir := t.TempDir()

	path, err := Create(sourceDir, true, testTime)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	expected := filepath.Join(sourceDir, "Diaries", "2026", "02", "28.md")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("created file not found: %v", err)
	}
}

func TestCreate_DuplicateEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sourceDir := t.TempDir()

	if _, err := Create(sourceDir, true, testTime); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	path, err := Create(sourceDir, true, testTime)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
	// The existing file must be preserved
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("existing file should be preserved: %v", statErr)
	}
}

func TestCreate_NoMkdir_MissingDirs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sourceDir := t.TempDir()

	_, err := Create(sourceDir, false, testTime)
	if err == nil {
		t.Fatal("expected error when dirs missing and --no-mkdir set")
	}
}

func TestCreate_YearMonthIndexFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sourceDir := t.TempDir()

	if _, err := Create(sourceDir, true, testTime); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	yearIndex := filepath.Join(sourceDir, "Diaries", "2026", "index.md")
	monthIndex := filepath.Join(sourceDir, "Diaries", "2026", "02", "index.md")
	for _, p := range []string{yearIndex, monthIndex} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("index file not created: %s", p)
		}
	}
}
