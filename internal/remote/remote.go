package remote

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/peacock0803sz/notizen/internal/config"
)

// Sync transfers src to the remote host using rsync over SSH.
func Sync(cfg *config.RemoteConfig, src string, dryRun, delete, recursive bool, out io.Writer) error {
	args := buildArgs(cfg, src, dryRun, delete, recursive)
	cmd := exec.Command("rsync", args...)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return mapError(err)
	}
	return nil
}

// BuildArgs constructs the rsync argument list (exported for testing).
func BuildArgs(cfg *config.RemoteConfig, src string, dryRun, delete, recursive bool) []string {
	return buildArgs(cfg, src, dryRun, delete, recursive)
}

func buildArgs(cfg *config.RemoteConfig, src string, dryRun, delete, recursive bool) []string {
	var flags []string

	if recursive {
		// --archive implies -rlptgoD (includes recursive)
		flags = append(flags, "--archive")
	} else {
		// Explicit flags without -r
		flags = append(flags, "-lptoD")
	}
	flags = append(flags, "--verbose", "--compress")

	if delete {
		flags = append(flags, "--delete")
	}
	if dryRun {
		flags = append(flags, "--dry-run")
	}

	// Build SSH command string
	sshCmd := fmt.Sprintf("ssh -p %d", cfg.Port)
	if cfg.Key != "" {
		sshCmd += fmt.Sprintf(" -i %q", cfg.Key)
	}
	flags = append(flags, "-e", sshCmd)

	// Destination: user@host:path
	dest := cfg.Host
	if cfg.User != "" {
		dest = cfg.User + "@" + dest
	}
	if cfg.Path != "" {
		dest += ":" + cfg.Path
	}

	return append(flags, "--", src, dest)
}

func mapError(err error) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		switch exitErr.ExitCode() {
		case 11, 12:
			return fmt.Errorf("rsync partial transfer error: %w", err)
		case 23:
			return fmt.Errorf("some files failed to transfer: %w", err)
		case 255:
			return fmt.Errorf("SSH connection failed — check host, port, and key: %w", err)
		}
	}
	return fmt.Errorf("rsync error: %w", err)
}
