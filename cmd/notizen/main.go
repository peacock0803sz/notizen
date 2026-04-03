package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peacock0803sz/notizen/internal/config"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:     "notizen",
	Short:   "Personal notes and diary manager",
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().String("root", "", "Root directory for notizen data")
}

func main() {
	rootCmd.AddCommand(diaryCmd, commitCmd, syncCmd, specsCmd, plansCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ensureRoot resolves the notizen root directory and ensures it (and root/source) exist.
func ensureRoot() (string, error) {
	flagValue, _ := rootCmd.PersistentFlags().GetString("root")
	root, err := config.ResolveRoot(flagValue)
	if err != nil {
		return "", fmt.Errorf("could not resolve root directory: %w", err)
	}
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", source, err)
	}
	return root, nil
}
