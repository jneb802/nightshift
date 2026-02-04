package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Show budget status",
	Long: `Display current budget status and usage.

Shows spending across all providers or a specific provider.`,
	Run: func(cmd *cobra.Command, args []string) {
		provider, _ := cmd.Flags().GetString("provider")
		if provider != "" {
			fmt.Printf("budget --provider %s: not implemented yet\n", provider)
		} else {
			fmt.Println("budget: not implemented yet")
		}
	},
}

func init() {
	budgetCmd.Flags().StringP("provider", "p", "", "Show specific provider status (e.g., claude, openai)")
	rootCmd.AddCommand(budgetCmd)
}
