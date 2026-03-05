package main

import (
	"path/filepath"

	"github.com/peacock0803sz/notizen/internal/config"
	"github.com/peacock0803sz/notizen/internal/remote"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync source directory to remote server via rsync",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().StringP("src", "s", "", "Local source directory (default: <root>/source)")
	syncCmd.Flags().StringP("dest", "d", "", "Remote destination path (from config if unset)")
	syncCmd.Flags().BoolP("dry-run", "n", false, "Preview without making changes")
	syncCmd.Flags().Bool("no-delete", false, "Do not delete remote files not in source")
	syncCmd.Flags().Bool("no-recursive", false, "Non-recursive sync")
}

func runSync(cmd *cobra.Command, args []string) error {
	root, err := ensureRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	src, _ := cmd.Flags().GetString("src")
	if src == "" {
		src = filepath.Join(root, "source")
	}

	if dest, _ := cmd.Flags().GetString("dest"); dest != "" {
		cfg.Path = dest
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noDelete, _ := cmd.Flags().GetBool("no-delete")
	noRecursive, _ := cmd.Flags().GetBool("no-recursive")

	return remote.Sync(cfg, src, dryRun, !noDelete, !noRecursive, cmd.OutOrStdout())
}
