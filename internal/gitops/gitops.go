package gitops

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// Sync runs git fetch/pull/add/commit/push in workDir.
// If there are no changes under source/, it exits early without committing.
func Sync(workDir string, out io.Writer, now time.Time) error {
	run := func(args ...string) error {
		return runCmd(workDir, out, args...)
	}

	if _, err := fmt.Fprintln(out, "Fetching origin..."); err != nil {
		return err
	}
	if err := run("git", "fetch", "origin"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "Pulling origin main..."); err != nil {
		return err
	}
	if err := run("git", "pull", "origin", "main"); err != nil {
		return err
	}

	changed, err := hasChanges(workDir)
	if err != nil {
		return err
	}
	if !changed {
		_, err := fmt.Fprintln(out, "Nothing to commit.")
		return err
	}

	if _, err := fmt.Fprintln(out, "Staging source/..."); err != nil {
		return err
	}
	if err := run("git", "add", "source/"); err != nil {
		return err
	}

	msg := fmt.Sprintf("Sync source/ at %s", now.Format("2006-01-02 15:04"))
	if _, err := fmt.Fprintf(out, "Committing: %s\n", msg); err != nil {
		return err
	}
	if err := run("git", "commit", "-m", msg); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "Pushing to origin main..."); err != nil {
		return err
	}
	return run("git", "push", "origin", "main")
}

func hasChanges(workDir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain", "source/")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

func runCmd(workDir string, out io.Writer, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %v failed: %w", args, err)
	}
	return nil
}
