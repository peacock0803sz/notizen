package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/peacock0803sz/notizen/internal/diary"
	"github.com/spf13/cobra"
)

var diaryCmd = &cobra.Command{
	Use:   "diary",
	Short: "Create today's diary entry",
	RunE:  runDiary,
}

func init() {
	diaryCmd.Flags().Bool("mkdir", true, "Create missing directories automatically")
	diaryCmd.Flags().Bool("no-mkdir", false, "Do not create missing directories")
}

func runDiary(cmd *cobra.Command, args []string) error {
	root, err := ensureRoot()
	if err != nil {
		return err
	}

	mkdir, _ := cmd.Flags().GetBool("mkdir")
	noMkdir, _ := cmd.Flags().GetBool("no-mkdir")
	if noMkdir {
		mkdir = false
	}

	sourceDir := filepath.Join(root, "source")
	path, err := diary.Create(sourceDir, mkdir, time.Now())
	if err != nil {
		if errors.Is(err, diary.ErrAlreadyExists) {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Already exists: %s\n", path); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), path)
	return nil
}
