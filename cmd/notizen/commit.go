package main

import (
	"time"

	"github.com/peacock0803sz/notizen/internal/gitops"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Fetch, commit, and push source changes",
	RunE:  runCommit,
}

func runCommit(cmd *cobra.Command, args []string) error {
	root, err := ensureRoot()
	if err != nil {
		return err
	}
	return gitops.Sync(root, cmd.OutOrStdout(), time.Now())
}
