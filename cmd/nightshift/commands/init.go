package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create configuration file",
	Long: `Initialize a new nightshift configuration file.

By default, creates nightshift.yaml in the current directory.
Use --global to create a global config at ~/.config/nightshift/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		global, _ := cmd.Flags().GetBool("global")
		if global {
			fmt.Println("init --global: not implemented yet")
		} else {
			fmt.Println("init: not implemented yet")
		}
	},
}

func init() {
	initCmd.Flags().Bool("global", false, "Create global config instead of project config")
	rootCmd.AddCommand(initCmd)
}
