package speckit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/peacock0803sz/notizen/internal/tmpl"
)

// Link creates a symlink from sourceDir/Agents/Specs/{repoName}/ to repoPath/specs/.
func Link(sourceDir, repoPath, nameOverride string) (string, error) {
	if _, err := os.Stat(repoPath); err != nil {
		return "", fmt.Errorf("repo path not found: %s", repoPath)
	}

	repoName, err := resolveRepoName(repoPath, nameOverride)
	if err != nil {
		return "", err
	}

	specsTarget := filepath.Join(repoPath, "specs")
	linkDir := filepath.Join(sourceDir, "Agents", "Specs", repoName)
	if err := os.MkdirAll(filepath.Dir(linkDir), 0o755); err != nil {
		return "", fmt.Errorf("failed to create parent directory: %w", err)
	}

	action, err := handleSymlink(linkDir, specsTarget)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(specsTarget); err == nil {
		branchName, _ := currentBranch(repoPath)
		if err := writeSpecIndexes(linkDir, branchName); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	switch action {
	case "created":
		return fmt.Sprintf("Linked %s specs to %s", repoName, linkDir), nil
	case "skipped":
		return fmt.Sprintf("Already linked: %s", repoName), nil
	default:
		return fmt.Sprintf("Updated link for %s", repoName), nil
	}
}

func resolveRepoName(repoPath, nameOverride string) (string, error) {
	if nameOverride != "" {
		return filepath.Base(nameOverride), nil
	}
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get git remote — use --name to specify repo name: %w", err)
	}
	url := strings.TrimSpace(string(out))
	// Extract repo name from URL, stripping .git suffix
	name := filepath.Base(url)
	return strings.TrimSuffix(name, ".git"), nil
}

// handleSymlink inspects the current state of linkDir and acts accordingly.
// Returns "created", "skipped", or "updated".
func handleSymlink(linkDir, target string) (string, error) {
	info, err := os.Lstat(linkDir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Symlink(target, linkDir); err != nil {
			return "", fmt.Errorf("failed to create symlink: %w", err)
		}
		return "created", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		current, err := os.Readlink(linkDir)
		if err != nil {
			return "", err
		}
		if current == target {
			return "skipped", nil
		}
		// Wrong target — update
		if err := os.Remove(linkDir); err != nil {
			return "", err
		}
		if err := os.Symlink(target, linkDir); err != nil {
			return "", err
		}
		return "updated", nil
	}

	// Real directory — migrate contents then replace with symlink
	if err := migrateDir(linkDir, target); err != nil {
		return "", err
	}
	return "created", nil
}

func migrateDir(src, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		oldPath := filepath.Join(src, e.Name())
		newPath := filepath.Join(dest, e.Name())
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}
	if err := os.Remove(src); err != nil {
		return err
	}
	return os.Symlink(dest, src)
}

func currentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func writeSpecIndexes(linkDir, branchName string) error {
	repoIdx, err := tmpl.Load("speckit/index.md.tmpl")
	if err != nil {
		return err
	}
	branches := []string{}
	if branchName != "" {
		branches = []string{branchName}
	}
	var buf bytes.Buffer
	if err := repoIdx.Execute(&buf, struct{ Branches []string }{branches}); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(linkDir, "index.md"), buf.Bytes(), 0o644); err != nil {
		return err
	}

	if branchName == "" {
		return nil
	}
	branchDir := filepath.Join(linkDir, branchName)
	if _, err := os.Stat(branchDir); err != nil {
		return nil // branch directory does not exist yet; skip
	}

	branchIdx, err := tmpl.Load("speckit/branch.md.tmpl")
	if err != nil {
		return err
	}
	entries, err := listMarkdownFiles(branchDir)
	if err != nil {
		return err
	}
	buf.Reset()
	if err := branchIdx.Execute(&buf, struct{ Files []string }{entries}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(branchDir, "index.md"), buf.Bytes(), 0o644)
}

func listMarkdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "index.md" {
			files = append(files, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return files, nil
}
