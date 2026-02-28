package diary

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/peacock0803sz/notizen/internal/tmpl"
)

// Entry holds template data for a diary entry.
type Entry struct {
	FormattedDate string
}

// Create writes today's diary entry under sourceDir.
// Returns the created file path. If the entry already exists, returns the path and ErrAlreadyExists.
func Create(sourceDir string, mkdir bool, now time.Time) (string, error) {
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	weekday := now.Format("Mon")

	entryDir := filepath.Join(sourceDir, "Diaries", year, month)
	entryPath := filepath.Join(entryDir, day+".md")

	if _, err := os.Stat(entryPath); err == nil {
		return entryPath, fmt.Errorf("%w: %s", ErrAlreadyExists, entryPath)
	}

	if err := ensureDirs(sourceDir, year, month, mkdir); err != nil {
		return "", err
	}

	t, err := tmpl.Load("diary/entry.md.tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, Entry{
		FormattedDate: fmt.Sprintf("%s/%s/%s (%s)", year, month, day, weekday),
	}); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile(entryPath, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("failed to write entry: %w", err)
	}
	return entryPath, nil
}

var ErrAlreadyExists = fmt.Errorf("diary entry already exists")

// ensureDirs creates the Diaries/Year/Month directory hierarchy and index files for new dirs.
func ensureDirs(sourceDir, year, month string, mkdir bool) error {
	yearDir := filepath.Join(sourceDir, "Diaries", year)
	monthDir := filepath.Join(yearDir, month)

	yearNew := false
	monthNew := false

	if _, err := os.Stat(yearDir); os.IsNotExist(err) {
		if !mkdir {
			return fmt.Errorf("directory does not exist (--no-mkdir): %s", yearDir)
		}
		yearNew = true
	}
	if _, err := os.Stat(monthDir); os.IsNotExist(err) {
		if !mkdir {
			return fmt.Errorf("directory does not exist (--no-mkdir): %s", monthDir)
		}
		monthNew = true
	}

	if mkdir {
		if err := os.MkdirAll(monthDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if yearNew {
		if err := writeIndexFile(yearDir); err != nil {
			return err
		}
	}
	if monthNew {
		if err := writeIndexFile(monthDir); err != nil {
			return err
		}
	}
	return nil
}

func writeIndexFile(dir string) error {
	t, err := tmpl.Load("diary/index.md.tmpl")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, struct{ Entries []string }{nil}); err != nil {
		return fmt.Errorf("failed to execute index template: %w", err)
	}
	indexPath := filepath.Join(dir, "index.md")
	return os.WriteFile(indexPath, buf.Bytes(), 0o644)
}
