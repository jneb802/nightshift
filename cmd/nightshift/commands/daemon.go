package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage background daemon",
	Long:  `Start, stop, or check status of the nightshift background daemon.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start background daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("daemon start: not implemented yet")
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop background daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("daemon stop: not implemented yet")
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("daemon status: not implemented yet")
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}
