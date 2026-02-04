package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute tasks",
	Long: `Execute configured tasks immediately.

By default, runs all enabled tasks. Use --task to run a specific task.
Use --dry-run to simulate execution without making changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		project, _ := cmd.Flags().GetString("project")
		task, _ := cmd.Flags().GetString("task")

		fmt.Printf("run: not implemented yet (dry-run=%v, project=%q, task=%q)\n", dryRun, project, task)
	},
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "Simulate execution without making changes")
	runCmd.Flags().StringP("project", "p", "", "Path to project directory")
	runCmd.Flags().StringP("task", "t", "", "Run specific task by name")
	rootCmd.AddCommand(runCmd)
}
