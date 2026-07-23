// Package superpowers links a repository's superpowers plugin documents
// (docs/superpowers/) into the notizen source tree.
package superpowers

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/peacock0803sz/notizen/internal/repolink"
	"github.com/peacock0803sz/notizen/internal/tmpl"
)

// sections are the superpowers plugin's document sections in toctree order.
// The fixed list keeps the order deterministic and ignores unrelated
// directories that may appear next to them.
var sections = []string{"specs", "plans"}

// plannedIndex is an index file rendered during preflight and written only
// after every validation has passed.
type plannedIndex struct {
	path    string
	content []byte
}

// Link creates a symlink from sourceDir/Agents/Superpowers/{repoName}/ to
// repoPath/docs/superpowers/ and refreshes the generated index files inside
// it. Everything is validated before the filesystem is modified; see the
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

	linkParent, _, err := repolink.ResolveUnder(source, "Agents", "Superpowers")
	if err != nil {
		return "", err
	}
	linkDir := filepath.Join(linkParent, repoName)
	docsDir, docsExists, err := repolink.ResolveUnder(repoRoot, "docs", "superpowers")
	if err != nil {
		return "", err
	}
	overlap, err := repolink.PathsOverlap(linkDir, docsDir)
	if err != nil {
		return "", err
	}
	if overlap {
		return "", fmt.Errorf("link directory %s and target %s overlap — choose a different --root or --name", linkDir, docsDir)
	}

	var indexes []plannedIndex
	if docsExists {
		if indexes, err = planIndexes(docsDir, force); err != nil {
			return "", err
		}
	}

	action, err := repolink.EnsureSymlink(linkDir, docsDir)
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
		return fmt.Sprintf("Linked %s superpowers to %s", repoName, linkDir), nil
	case repolink.ActionSkipped:
		return fmt.Sprintf("Already linked: %s", repoName), nil
	default:
		return fmt.Sprintf("Updated link for %s", repoName), nil
	}
}

// planIndexes renders the root index over the sections that exist as real
// directories, plus one index per such section. All destination files are
// ownership-checked here so a conflict surfaces before any change, and the
// failures are joined so --force users see every affected path at once.
func planIndexes(docsDir string, force bool) ([]plannedIndex, error) {
	var present []string
	var sectionDirs []string
	for _, section := range sections {
		dir, exists, err := repolink.ResolveUnder(docsDir, section)
		if err != nil {
			return nil, err
		}
		if exists {
			present = append(present, section)
			sectionDirs = append(sectionDirs, dir)
		}
	}

	rootTmpl, err := tmpl.Load("superpowers/index.md.tmpl")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := rootTmpl.Execute(&buf, struct{ Sections []string }{present}); err != nil {
		return nil, err
	}
	indexes := []plannedIndex{{
		path:    filepath.Join(docsDir, "index.md"),
		content: append([]byte(nil), buf.Bytes()...),
	}}

	if len(sectionDirs) > 0 {
		sectionTmpl, err := tmpl.Load("superpowers/section.md.tmpl")
		if err != nil {
			return nil, err
		}
		for _, dir := range sectionDirs {
			files, err := repolink.ListMarkdownFiles(dir)
			if err != nil {
				return nil, err
			}
			buf.Reset()
			if err := sectionTmpl.Execute(&buf, struct{ Files []string }{files}); err != nil {
				return nil, err
			}
			indexes = append(indexes, plannedIndex{
				path:    filepath.Join(dir, "index.md"),
				content: append([]byte(nil), buf.Bytes()...),
			})
		}
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
