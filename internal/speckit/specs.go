package speckit

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/peacock0803sz/notizen/internal/repolink"
	"github.com/peacock0803sz/notizen/internal/tmpl"
)

// plannedIndex is an index file rendered during preflight and written only
// after every validation has passed.
type plannedIndex struct {
	path    string
	content []byte
}

// Link creates a symlink from sourceDir/Agents/Specs/{repoName}/ to
// repoPath/specs/ and refreshes the generated index files inside specs/.
// Everything is validated before the filesystem is modified; see the
// repolink package for the safety guarantees.
func Link(sourceDir, repoPath, nameOverride string, force bool) (string, error) {
	if repoPath == "" {
		return "", errors.New("repo path must not be empty")
	}
	repoRoot, err := repolink.NormalizeDir(repoPath)
	if err != nil {
		return "", fmt.Errorf("repo path not found: %s: %w", repoPath, err)
	}
	source, err := repolink.NormalizeDir(sourceDir)
	if err != nil {
		return "", fmt.Errorf("source directory not found: %s: %w", sourceDir, err)
	}
	repoName, err := repolink.ResolveRepoName(repoRoot, nameOverride)
	if err != nil {
		return "", err
	}

	linkParent, _, err := repolink.ResolveUnder(source, "Agents", "Specs")
	if err != nil {
		return "", err
	}
	linkDir := filepath.Join(linkParent, repoName)
	specsDir, specsExists, err := repolink.ResolveUnder(repoRoot, "specs")
	if err != nil {
		return "", err
	}
	overlap, err := repolink.PathsOverlap(linkDir, specsDir)
	if err != nil {
		return "", err
	}
	if overlap {
		return "", fmt.Errorf("link directory %s and target %s overlap — choose a different --root or --name", linkDir, specsDir)
	}

	var indexes []plannedIndex
	if specsExists {
		if indexes, err = planIndexes(repoRoot, specsDir, force); err != nil {
			return "", err
		}
	}

	action, err := repolink.EnsureSymlink(linkDir, specsDir)
	if err != nil {
		return "", err
	}
	for _, idx := range indexes {
		if err := repolink.WriteIndex(idx.path, idx.content, force); err != nil {
			return "", err
		}
	}

	switch action {
	case repolink.ActionCreated:
		return fmt.Sprintf("Linked %s specs to %s", repoName, linkDir), nil
	case repolink.ActionSkipped:
		return fmt.Sprintf("Already linked: %s", repoName), nil
	default:
		return fmt.Sprintf("Updated link for %s", repoName), nil
	}
}

// planIndexes renders the root index (listing the current branch even when
// its directory does not exist yet) and, when the branch directory exists,
// the branch index. All destination files are ownership-checked here so a
// conflict surfaces before any change, and the failures are joined so
// --force users see every affected path at once.
func planIndexes(repoRoot, specsDir string, force bool) ([]plannedIndex, error) {
	// git may be absent or the repo may have no commits; the link is still
	// useful then, so a missing branch only skips the branch listing.
	branchName, _ := currentBranch(repoRoot)

	var branches []string
	branchDir := ""
	branchDirExists := false
	if branchName != "" {
		parts := strings.Split(branchName, "/")
		if strings.EqualFold(parts[0], "index.md") {
			return nil, fmt.Errorf("branch name %q collides with the generated index.md", branchName)
		}
		dir, exists, err := repolink.ResolveUnder(specsDir, parts...)
		if err != nil {
			return nil, err
		}
		branches = []string{branchName}
		branchDir, branchDirExists = dir, exists
	}

	rootTmpl, err := tmpl.Load("speckit/index.md.tmpl")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := rootTmpl.Execute(&buf, struct{ Branches []string }{branches}); err != nil {
		return nil, err
	}
	indexes := []plannedIndex{{
		path:    filepath.Join(specsDir, "index.md"),
		content: append([]byte(nil), buf.Bytes()...),
	}}

	if branchDirExists {
		branchTmpl, err := tmpl.Load("speckit/branch.md.tmpl")
		if err != nil {
			return nil, err
		}
		files, err := repolink.ListMarkdownFiles(branchDir)
		if err != nil {
			return nil, err
		}
		buf.Reset()
		if err := branchTmpl.Execute(&buf, struct{ Files []string }{files}); err != nil {
			return nil, err
		}
		indexes = append(indexes, plannedIndex{
			path:    filepath.Join(branchDir, "index.md"),
			content: append([]byte(nil), buf.Bytes()...),
		})
	}

	var errs []error
	for _, idx := range indexes {
		if err := repolink.CheckIndexWritable(idx.path, force); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return indexes, nil
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
