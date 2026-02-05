package commands

import (
	"fmt"

	"github.com/marcus/nightshift/internal/ui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the Nightshift TUI",
	Long:  "Launch the interactive terminal UI for monitoring Nightshift runs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ui.New().Run(); err != nil {
			return fmt.Errorf("run tui: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
