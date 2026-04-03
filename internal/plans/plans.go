package plans

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Sentinel errors for plan file processing.
var (
	ErrAlreadyTimestamped = errors.New("file already has timestamp")
	ErrMissingHeading     = errors.New("no H1 heading found")
	ErrDestExists         = errors.New("destination file already exists")
)

// Result describes the outcome of archiving a single plan file.
type Result struct {
	OriginalPath string
	ArchivedPath string
}

var (
	timestampPrefix = regexp.MustCompile(`(?m)^# \d{2}:\d{2}:\d{2} `)
	h1Pattern       = regexp.MustCompile(`(?m)^(# )(.+)$`)
)

// ProcessFile reads a plan markdown file, prepends HH:MM:SS to its first H1
// heading, writes the result to destBase/YYYY/MM/DD/HHMMSS_<filename>, and
// removes the original.
//
// The caller is responsible for validating that filePath is a legitimate source
// (e.g. under ~/.claude/plans/). ProcessFile trusts the path it receives.
func ProcessFile(filePath, destBase string) (*Result, error) {
	if !strings.HasSuffix(filePath, ".md") {
		return nil, fmt.Errorf("not a markdown file: %s", filePath)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}
	text := string(content)

	if timestampPrefix.MatchString(text) {
		return nil, ErrAlreadyTimestamped
	}

	loc := h1Pattern.FindStringIndex(text)
	if loc == nil {
		return nil, ErrMissingHeading
	}

	// Replace only the first H1 match.
	mtime := info.ModTime()
	timeStr := mtime.Format("15:04:05")
	match := h1Pattern.FindStringSubmatch(text)
	replacement := match[1] + timeStr + " " + match[2]
	newText := text[:loc[0]] + replacement + text[loc[1]:]

	// Build destination path: destBase/YYYY/MM/DD/HHMMSS_<filename>
	destDir := filepath.Join(destBase, mtime.Format("2006"), mtime.Format("01"), mtime.Format("02"))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	newName := mtime.Format("150405") + "_" + filepath.Base(filePath)
	destPath := filepath.Join(destDir, newName)

	if _, err := os.Stat(destPath); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrDestExists, destPath)
	}

	if err := os.WriteFile(destPath, []byte(newText), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write archived file: %w", err)
	}

	if err := os.Remove(filePath); err != nil {
		return nil, fmt.Errorf("failed to remove original file: %w", err)
	}

	return &Result{
		OriginalPath: filePath,
		ArchivedPath: destPath,
	}, nil
}

// ArchiveAll processes every .md file in plansDir.
// Skippable errors (already timestamped, missing heading) are silently ignored.
// Hard errors (dest collision, IO failure) stop processing immediately.
func ArchiveAll(plansDir, destBase string) ([]Result, error) {
	pattern := filepath.Join(plansDir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob %s: %w", pattern, err)
	}

	var results []Result
	for _, path := range matches {
		result, err := ProcessFile(path, destBase)
		if err != nil {
			if errors.Is(err, ErrAlreadyTimestamped) || errors.Is(err, ErrMissingHeading) {
				continue
			}
			return results, err
		}
		results = append(results, *result)
	}
	return results, nil
}
