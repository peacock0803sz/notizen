package plans

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupPlanFile creates a .claude/plans/ directory tree inside a temp dir and
// writes a plan file with the given content. Returns the file path.
func setupPlanFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".claude", "plans")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set a known mtime so tests are deterministic.
	mtime := time.Date(2026, 4, 3, 14, 30, 45, 0, time.Local)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProcessFile_Basic(t *testing.T) {
	src := setupPlanFile(t, "my-plan.md", "# My Great Plan\n\nSome details.\n")
	destBase := t.TempDir()

	result, err := ProcessFile(src, destBase)
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}

	// Verify destination path structure.
	wantSuffix := filepath.Join("2026", "04", "03", "143045_my-plan.md")
	if !strings.HasSuffix(result.ArchivedPath, wantSuffix) {
		t.Errorf("ArchivedPath = %q, want suffix %q", result.ArchivedPath, wantSuffix)
	}

	// Verify timestamp was prepended to H1.
	data, err := os.ReadFile(result.ArchivedPath)
	if err != nil {
		t.Fatalf("cannot read archived file: %v", err)
	}
	if !strings.HasPrefix(string(data), "# 14:30:45 My Great Plan\n") {
		t.Errorf("archived content = %q, want H1 with timestamp", string(data))
	}

	// Original must be deleted.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("original file should have been deleted")
	}
}

func TestProcessFile_AlreadyTimestamped(t *testing.T) {
	src := setupPlanFile(t, "done.md", "# 12:34:56 Already Done\n\nBody.\n")
	destBase := t.TempDir()

	_, err := ProcessFile(src, destBase)
	if !errors.Is(err, ErrAlreadyTimestamped) {
		t.Errorf("expected ErrAlreadyTimestamped, got %v", err)
	}

	// Original must still exist.
	if _, err := os.Stat(src); err != nil {
		t.Error("original file should be preserved")
	}
}

func TestProcessFile_MissingHeading(t *testing.T) {
	src := setupPlanFile(t, "no-heading.md", "No heading here.\n\nJust text.\n")
	destBase := t.TempDir()

	_, err := ProcessFile(src, destBase)
	if !errors.Is(err, ErrMissingHeading) {
		t.Errorf("expected ErrMissingHeading, got %v", err)
	}
}

func TestProcessFile_NotMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ProcessFile(path, t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-markdown file")
	}
}

func TestProcessFile_NonExistent(t *testing.T) {
	_, err := ProcessFile("/nonexistent/plan.md", t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestProcessFile_DestCollision(t *testing.T) {
	src := setupPlanFile(t, "collision.md", "# Plan Title\n\nBody.\n")
	destBase := t.TempDir()

	// Pre-create the destination file to trigger collision.
	destDir := filepath.Join(destBase, "2026", "04", "03")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	destPath := filepath.Join(destDir, "143045_collision.md")
	if err := os.WriteFile(destPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ProcessFile(src, destBase)
	if !errors.Is(err, ErrDestExists) {
		t.Errorf("expected ErrDestExists, got %v", err)
	}

	// Original must still exist.
	if _, err := os.Stat(src); err != nil {
		t.Error("original file should be preserved on dest collision")
	}
}

func TestProcessFile_OriginalPreservedOnWriteFailure(t *testing.T) {
	src := setupPlanFile(t, "keep-me.md", "# Keep Me\n\nBody.\n")

	// Make a read-only parent so MkdirAll fails inside ProcessFile.
	parent := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parent, 0o755); err != nil {
			t.Logf("cleanup: chmod: %v", err)
		}
	})

	_, err := ProcessFile(src, filepath.Join(parent, "sub"))
	if err == nil {
		t.Fatal("expected error for write failure")
	}

	// Original must still exist.
	if _, err := os.Stat(src); err != nil {
		t.Error("original file should be preserved when write fails")
	}
}

func TestArchiveAll_Empty(t *testing.T) {
	plansDir := t.TempDir()
	destBase := t.TempDir()

	results, err := ArchiveAll(plansDir, destBase)
	if err != nil {
		t.Fatalf("ArchiveAll() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestArchiveAll_Multiple(t *testing.T) {
	plansDir := filepath.Join(t.TempDir(), ".claude", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatal(err)
	}
	destBase := t.TempDir()

	// Create a valid plan and an already-timestamped plan.
	mtime := time.Date(2026, 4, 3, 14, 30, 45, 0, time.Local)
	for name, content := range map[string]string{
		"valid.md":       "# Valid Plan\n\nBody.\n",
		"timestamped.md": "# 09:00:00 Already Done\n\nBody.\n",
	} {
		path := filepath.Join(plansDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}

	results, err := ArchiveAll(plansDir, destBase)
	if err != nil {
		t.Fatalf("ArchiveAll() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (timestamped skipped), got %d", len(results))
	}
}

func TestArchiveAll_HardError(t *testing.T) {
	plansDir := filepath.Join(t.TempDir(), ".claude", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatal(err)
	}
	destBase := t.TempDir()

	mtime := time.Date(2026, 4, 3, 14, 30, 45, 0, time.Local)
	path := filepath.Join(plansDir, "will-collide.md")
	if err := os.WriteFile(path, []byte("# Plan\n\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	// Pre-create destination to trigger collision.
	destDir := filepath.Join(destBase, "2026", "04", "03")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "143045_will-collide.md"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ArchiveAll(plansDir, destBase)
	if !errors.Is(err, ErrDestExists) {
		t.Errorf("expected ErrDestExists from ArchiveAll, got %v", err)
	}
}
