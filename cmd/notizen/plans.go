package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/peacock0803sz/notizen/internal/plans"
	"github.com/spf13/cobra"
)

var plansCmd = &cobra.Command{
	Use:   "plans",
	Short: "Archive Claude plan files into notizen",
	RunE:  runPlans,
}

func init() {
	plansCmd.Flags().Bool("archive", false, "Batch-archive all plans from source directory")
	plansCmd.Flags().Bool("hook", false, "Read file path from stdin JSON (Claude Code hook mode)")
	plansCmd.Flags().String("plans-dir", "", "Override source directory for plans (default: ~/.claude/plans)")
}

func runPlans(cmd *cobra.Command, args []string) error {
	root, err := ensureRoot()
	if err != nil {
		return err
	}

	destBase := filepath.Join(root, "source", "Agents", "Plans")

	plansDir, _ := cmd.Flags().GetString("plans-dir")
	if plansDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		plansDir = filepath.Join(home, ".claude", "plans")
	}

	archive, _ := cmd.Flags().GetBool("archive")
	hook, _ := cmd.Flags().GetBool("hook")

	switch {
	case archive:
		results, err := plans.ArchiveAll(plansDir, destBase)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "No plan files to archive")
			return err
		}
		for _, r := range results {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Archived: %s -> %s\n", filepath.Base(r.OriginalPath), r.ArchivedPath); err != nil {
				return err
			}
		}
		return nil

	case hook:
		filePath, err := parseHookInput(os.Stdin)
		if err != nil {
			return err
		}
		result, err := plans.ProcessFile(filePath, destBase)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Moved to: %s\n", result.ArchivedPath)
		return err

	default:
		return cmd.Help()
	}
}

type hookInput struct {
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

func parseHookInput(r io.Reader) (string, error) {
	var input hookInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return "", fmt.Errorf("failed to parse hook input: %w", err)
	}
	if input.ToolInput.FilePath == "" {
		return "", fmt.Errorf("no file_path in hook input")
	}
	return input.ToolInput.FilePath, nil
}
