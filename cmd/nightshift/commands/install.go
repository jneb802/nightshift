package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [launchd|systemd|cron]",
	Short: "Install system service",
	Long: `Generate and install a system service for nightshift.

Supported init systems:
  launchd  - macOS (creates ~/Library/LaunchAgents plist)
  systemd  - Linux (creates user systemd unit)
  cron     - Universal (creates crontab entry)

If no init system is specified, auto-detects based on OS.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("install: not implemented yet (auto-detect)")
		} else {
			fmt.Printf("install %s: not implemented yet\n", args[0])
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove system service",
	Long:  `Remove the nightshift system service.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("uninstall: not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
