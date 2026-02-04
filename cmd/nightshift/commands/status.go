package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show run history",
	Long: `Display nightshift run history and activity.

Shows the last N runs (default: 5) or today's activity summary.`,
	Run: func(cmd *cobra.Command, args []string) {
		last, _ := cmd.Flags().GetInt("last")
		today, _ := cmd.Flags().GetBool("today")

		if today {
			fmt.Println("status --today: not implemented yet")
		} else {
			fmt.Printf("status --last %d: not implemented yet\n", last)
		}
	},
}

func init() {
	statusCmd.Flags().IntP("last", "n", 5, "Show last N runs")
	statusCmd.Flags().Bool("today", false, "Show today's activity summary")
	rootCmd.AddCommand(statusCmd)
}
