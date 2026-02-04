package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcusvorwaller/nightshift/internal/budget"
	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/marcusvorwaller/nightshift/internal/providers"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Show budget status",
	Long: `Display current budget status and usage.

Shows spending across all providers or a specific provider.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		return runBudget(provider)
	},
}

func init() {
	budgetCmd.Flags().StringP("provider", "p", "", "Show specific provider status (claude, codex)")
	rootCmd.AddCommand(budgetCmd)
}

func runBudget(filterProvider string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Initialize providers
	var claude *providers.Claude
	var codex *providers.Codex

	if cfg.Providers.Claude.Enabled {
		dataPath := cfg.ExpandedProviderPath("claude")
		if dataPath != "" {
			claude = providers.NewClaudeWithPath(dataPath)
		} else {
			claude = providers.NewClaude()
		}
	}

	if cfg.Providers.Codex.Enabled {
		dataPath := cfg.ExpandedProviderPath("codex")
		if dataPath != "" {
			codex = providers.NewCodexWithPath(dataPath)
		} else {
			codex = providers.NewCodex()
		}
	}

	// Create budget manager
	mgr := budget.NewManagerFromProviders(cfg, claude, codex)

	// Determine which providers to show
	providerList := []string{}
	if filterProvider != "" {
		// Validate provider filter
		if filterProvider != "claude" && filterProvider != "codex" {
			return fmt.Errorf("unknown provider: %s (valid: claude, codex)", filterProvider)
		}
		providerList = append(providerList, filterProvider)
	} else {
		// Show all enabled providers
		if cfg.Providers.Claude.Enabled {
			providerList = append(providerList, "claude")
		}
		if cfg.Providers.Codex.Enabled {
			providerList = append(providerList, "codex")
		}
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	// Print header
	mode := cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}
	fmt.Printf("Budget Status (mode: %s)\n", mode)
	fmt.Println("================================")
	fmt.Println()

	// Print status for each provider
	for _, provName := range providerList {
		if err := printProviderBudget(mgr, cfg, provName, codex); err != nil {
			fmt.Printf("%s: error: %v\n\n", provName, err)
			continue
		}
		fmt.Println()
	}

	return nil
}

func printProviderBudget(mgr *budget.Manager, cfg *config.Config, provName string, codex *providers.Codex) error {
	result, err := mgr.CalculateAllowance(provName)
	if err != nil {
		return err
	}

	weeklyBudget := int64(cfg.GetProviderBudget(provName))

	// Provider name header
	fmt.Printf("[%s]\n", provName)

	// Mode-specific display
	if result.Mode == "daily" {
		dailyBudget := weeklyBudget / 7
		usedTokens := int64(float64(dailyBudget) * result.UsedPercent / 100)
		remaining := dailyBudget - usedTokens

		fmt.Printf("  Mode:         %s\n", result.Mode)
		fmt.Printf("  Weekly:       %s tokens\n", formatTokens64(weeklyBudget))
		fmt.Printf("  Daily:        %s tokens\n", formatTokens64(dailyBudget))
		fmt.Printf("  Used today:   %s (%.1f%%)\n", formatTokens64(usedTokens), result.UsedPercent)
		fmt.Printf("  Remaining:    %s tokens\n", formatTokens64(remaining))
		fmt.Printf("  Reserve:      %s tokens\n", formatTokens64(result.ReserveAmount))
		fmt.Printf("  Nightshift:   %s tokens available\n", formatTokens64(result.Allowance))
	} else {
		// Weekly mode
		usedTokens := int64(float64(weeklyBudget) * result.UsedPercent / 100)
		remaining := weeklyBudget - usedTokens

		fmt.Printf("  Mode:         %s\n", result.Mode)
		fmt.Printf("  Weekly:       %s tokens\n", formatTokens64(weeklyBudget))
		fmt.Printf("  Used:         %s (%.1f%%)\n", formatTokens64(usedTokens), result.UsedPercent)
		fmt.Printf("  Remaining:    %s tokens\n", formatTokens64(remaining))
		fmt.Printf("  Days left:    %d\n", result.RemainingDays)

		if result.Multiplier > 1.0 {
			fmt.Printf("  Multiplier:   %.1fx (end-of-week)\n", result.Multiplier)
		}

		fmt.Printf("  Reserve:      %s tokens\n", formatTokens64(result.ReserveAmount))
		fmt.Printf("  Nightshift:   %s tokens available\n", formatTokens64(result.Allowance))
	}

	// Show reset time for Codex
	if provName == "codex" && codex != nil {
		resetTime, err := codex.GetResetTime(result.Mode)
		if err == nil && !resetTime.IsZero() {
			fmt.Printf("  Resets at:    %s\n", formatResetTime(resetTime))
		}
	}

	// Progress bar
	fmt.Printf("  Progress:     %s\n", progressBar(result.UsedPercent, 30))

	return nil
}

func formatTokens64(tokens int64) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

func formatResetTime(t time.Time) string {
	now := time.Now()
	duration := t.Sub(now)

	if duration <= 0 {
		return "now"
	}

	// Show relative time
	if duration < time.Hour {
		return fmt.Sprintf("in %d min (%s)", int(duration.Minutes()), t.Format("15:04"))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("in %dh %dm (%s)", int(duration.Hours()), int(duration.Minutes())%60, t.Format("15:04"))
	}

	days := int(duration.Hours() / 24)
	return fmt.Sprintf("in %d days (%s)", days, t.Format("Jan 2 15:04"))
}

func progressBar(percent float64, width int) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	filled := int(percent * float64(width) / 100)
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "#"
	}
	for i := 0; i < empty; i++ {
		bar += "-"
	}

	return fmt.Sprintf("[%s] %.1f%%", bar, percent)
}
