package main

import (
	"fmt"
	"path/filepath"

	"github.com/peacock0803sz/notizen/internal/speckit"
	"github.com/spf13/cobra"
)

var specsCmd = &cobra.Command{
	Use:   "specs <repo-path>",
	Short: "Link repo spec artifacts into notizen documentation",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecs,
}

func init() {
	specsCmd.Flags().StringP("name", "n", "", "Override auto-detected repo name")
}

func runSpecs(cmd *cobra.Command, args []string) error {
	root, err := ensureRoot()
	if err != nil {
		return err
	}

	repoPath := args[0]
	name, _ := cmd.Flags().GetString("name")
	sourceDir := filepath.Join(root, "source")

	msg, err := speckit.Link(sourceDir, repoPath, name)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), msg)
	return err
}
