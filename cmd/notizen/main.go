package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:     "notizen",
	Short:   "Personal notes and diary manager",
	Version: version,
}

func main() {
	rootCmd.AddCommand(diaryCmd, commitCmd, syncCmd, specsCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ensureRoot ensures ~/.notizen and ~/.notizen/source exist, creating them if needed.
func ensureRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	root := filepath.Join(home, ".notizen")
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", source, err)
	}
	return root, nil
}
