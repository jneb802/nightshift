package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/marcusvorwaller/nightshift/internal/calibrator"
	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/marcusvorwaller/nightshift/internal/db"
	"github.com/marcusvorwaller/nightshift/internal/providers"
	"github.com/marcusvorwaller/nightshift/internal/snapshots"
)

var budgetSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture a usage snapshot",
	Long: `Capture a usage snapshot for budget calibration.

By default, captures snapshots for all enabled providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		localOnly, _ := cmd.Flags().GetBool("local-only")
		return runBudgetSnapshot(cmd, provider, localOnly)
	},
}

var budgetHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent budget snapshots",
	Long:  `Show recent usage snapshots for budget calibration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		n, _ := cmd.Flags().GetInt("n")
		return runBudgetHistory(provider, n)
	},
}

var budgetCalibrateCmd = &cobra.Command{
	Use:   "calibrate",
	Short: "Show calibration status",
	Long:  `Show inferred budget calibration status for providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		return runBudgetCalibrate(provider)
	},
}

func init() {
	budgetSnapshotCmd.Flags().StringP("provider", "p", "", "Provider to snapshot (claude, codex)")
	budgetSnapshotCmd.Flags().Bool("local-only", false, "Skip tmux scraping and store local-only snapshot")

	budgetHistoryCmd.Flags().StringP("provider", "p", "", "Provider to show history for (claude, codex)")
	budgetHistoryCmd.Flags().IntP("n", "n", 20, "Number of snapshots to show")

	budgetCalibrateCmd.Flags().StringP("provider", "p", "", "Provider to calibrate (claude, codex)")

	budgetCmd.AddCommand(budgetSnapshotCmd)
	budgetCmd.AddCommand(budgetHistoryCmd)
	budgetCmd.AddCommand(budgetCalibrateCmd)
}

func runBudgetSnapshot(cmd *cobra.Command, filterProvider string, localOnly bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer database.Close()

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	scraper := snapshots.UsageScraper(nil)
	if !localOnly && cfg.Budget.CalibrateEnabled && strings.ToLower(cfg.Budget.BillingMode) != "api" {
		scraper = tmuxScraper{}
	}

	collector := snapshots.NewCollector(
		database,
		providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude")),
		providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex")),
		scraper,
		weekStartDayFromConfig(cfg),
	)

	ctx := cmd.Context()
	for _, provider := range providerList {
		snapshot, err := collector.TakeSnapshot(ctx, provider)
		if err != nil {
			fmt.Printf("%s: error: %v\n", provider, err)
			continue
		}
		fmt.Printf("%s\n", formatSnapshotLine(snapshot))
	}

	return nil
}

func runBudgetHistory(filterProvider string, n int) error {
	if n <= 0 {
		return fmt.Errorf("n must be positive")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer database.Close()

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	collector := snapshots.NewCollector(database, nil, nil, nil, weekStartDayFromConfig(cfg))

	for _, provider := range providerList {
		history, err := collector.GetLatest(provider, n)
		if err != nil {
			fmt.Printf("%s: error: %v\n\n", provider, err)
			continue
		}
		if len(history) == 0 {
			fmt.Printf("[%s]\n  No snapshots yet.\n\n", provider)
			continue
		}

		fmt.Printf("[%s]\n", provider)
		printSnapshotTable(history)
		fmt.Println()
	}

	return nil
}

func runBudgetCalibrate(filterProvider string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer database.Close()

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	cal := calibrator.New(database, cfg)
	for _, provider := range providerList {
		result, err := cal.Calibrate(provider)
		if err != nil {
			fmt.Printf("%s: error: %v\n\n", provider, err)
			continue
		}

		fmt.Printf("[%s]\n", provider)
		fmt.Printf("  Source:      %s\n", result.Source)
		fmt.Printf("  Budget:      %s tokens\n", formatTokens64(result.InferredBudget))
		fmt.Printf("  Confidence:  %s\n", result.Confidence)
		fmt.Printf("  Samples:     %d\n", result.SampleCount)
		if result.Variance > 0 {
			fmt.Printf("  Variance:    %.0f\n", result.Variance)
		}
		fmt.Println()
	}

	return nil
}

func formatSnapshotLine(snapshot snapshots.Snapshot) string {
	parts := []string{
		fmt.Sprintf("%s: local %s tokens", snapshot.Provider, formatTokens64(snapshot.LocalTokens)),
		fmt.Sprintf("daily %s", formatTokens64(snapshot.LocalDaily)),
	}

	if snapshot.ScrapedPct != nil {
		parts = append(parts, fmt.Sprintf("scraped %.1f%%", *snapshot.ScrapedPct))
	}
	if snapshot.InferredBudget != nil {
		parts = append(parts, fmt.Sprintf("inferred %s tokens", formatTokens64(*snapshot.InferredBudget)))
	}
	if snapshot.ScrapedPct == nil {
		parts = append(parts, "local-only")
	}

	return strings.Join(parts, ", ")
}

func printSnapshotTable(history []snapshots.Snapshot) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "Time\tLocal\tDaily\tPct\tInferred")
	for _, snapshot := range history {
		pct := "-"
		if snapshot.ScrapedPct != nil {
			pct = fmt.Sprintf("%.1f%%", *snapshot.ScrapedPct)
		}
		inferred := "-"
		if snapshot.InferredBudget != nil {
			inferred = formatTokens64(*snapshot.InferredBudget)
		}
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\n",
			snapshot.Timestamp.Format("Jan 02 15:04"),
			formatTokens64(snapshot.LocalTokens),
			formatTokens64(snapshot.LocalDaily),
			pct,
			inferred,
		)
	}
	writer.Flush()
}
