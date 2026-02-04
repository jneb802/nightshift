// Package budget implements token budget calculation and allocation for nightshift.
// Supports daily and weekly modes with reserve and aggressive end-of-week options.
package budget

import (
	"fmt"
	"math"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/marcusvorwaller/nightshift/internal/providers"
)

// UsageProvider is the interface for getting usage data from a provider.
type UsageProvider interface {
	Name() string
}

// ClaudeUsageProvider extends UsageProvider for Claude-specific usage methods.
type ClaudeUsageProvider interface {
	UsageProvider
	GetUsedPercent(mode string, weeklyBudget int64) (float64, error)
}

// CodexUsageProvider extends UsageProvider for Codex-specific usage methods.
type CodexUsageProvider interface {
	UsageProvider
	GetUsedPercent(mode string) (float64, error)
	GetResetTime(mode string) (time.Time, error)
}

// Manager calculates and manages token budget allocation across providers.
type Manager struct {
	cfg     *config.Config
	claude  ClaudeUsageProvider
	codex   CodexUsageProvider
	nowFunc func() time.Time // for testing
}

// NewManager creates a budget manager with the given configuration and providers.
func NewManager(cfg *config.Config, claude ClaudeUsageProvider, codex CodexUsageProvider) *Manager {
	return &Manager{
		cfg:     cfg,
		claude:  claude,
		codex:   codex,
		nowFunc: time.Now,
	}
}

// AllowanceResult contains the calculated budget allowance and metadata.
type AllowanceResult struct {
	Allowance      int64   // Final token allowance for this run
	BudgetBase     int64   // Base budget (daily or remaining weekly)
	UsedPercent    float64 // Current used percentage
	ReserveAmount  int64   // Tokens reserved
	Mode           string  // "daily" or "weekly"
	RemainingDays  int     // Days until reset (weekly mode only)
	Multiplier     float64 // End-of-week multiplier (weekly mode only)
}

// CalculateAllowance determines how many tokens nightshift can use for this run.
func (m *Manager) CalculateAllowance(provider string) (*AllowanceResult, error) {
	weeklyBudget := int64(m.cfg.GetProviderBudget(provider))
	if weeklyBudget <= 0 {
		return nil, fmt.Errorf("invalid weekly budget for provider %s: %d", provider, weeklyBudget)
	}

	usedPercent, err := m.GetUsedPercent(provider)
	if err != nil {
		return nil, fmt.Errorf("getting used percent for %s: %w", provider, err)
	}

	mode := m.cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}

	maxPercent := m.cfg.Budget.MaxPercent
	if maxPercent <= 0 {
		maxPercent = config.DefaultMaxPercent
	}

	reservePercent := m.cfg.Budget.ReservePercent
	if reservePercent < 0 {
		reservePercent = config.DefaultReservePercent
	}

	var result *AllowanceResult

	switch mode {
	case "daily":
		result = m.calculateDailyAllowance(weeklyBudget, usedPercent, maxPercent)
	case "weekly":
		remainingDays, err := m.DaysUntilWeeklyReset(provider)
		if err != nil {
			return nil, fmt.Errorf("getting days until reset: %w", err)
		}
		result = m.calculateWeeklyAllowance(weeklyBudget, usedPercent, maxPercent, remainingDays)
	default:
		return nil, fmt.Errorf("invalid budget mode: %s", mode)
	}

	// Apply reserve enforcement
	result = m.applyReserve(result, reservePercent)

	return result, nil
}

// calculateDailyAllowance implements the daily mode budget algorithm.
// Daily mode: Each night uses up to max_percent of that day's budget (weekly/7).
func (m *Manager) calculateDailyAllowance(weeklyBudget int64, usedPercent float64, maxPercent int) *AllowanceResult {
	dailyBudget := weeklyBudget / 7
	availableToday := float64(dailyBudget) * (1 - usedPercent/100)
	nightshiftAllowance := availableToday * float64(maxPercent) / 100

	// Cap at available (can't use more than available)
	if nightshiftAllowance > availableToday {
		nightshiftAllowance = availableToday
	}

	return &AllowanceResult{
		Allowance:   int64(math.Max(0, nightshiftAllowance)),
		BudgetBase:  dailyBudget,
		UsedPercent: usedPercent,
		Mode:        "daily",
		Multiplier:  1.0,
	}
}

// calculateWeeklyAllowance implements the weekly mode budget algorithm.
// Weekly mode: Each night uses up to max_percent of REMAINING weekly budget.
func (m *Manager) calculateWeeklyAllowance(weeklyBudget int64, usedPercent float64, maxPercent int, remainingDays int) *AllowanceResult {
	if remainingDays <= 0 {
		remainingDays = 1 // Avoid division by zero
	}

	remainingWeekly := float64(weeklyBudget) * (1 - usedPercent/100)

	// Aggressive end-of-week multiplier
	multiplier := 1.0
	if m.cfg.Budget.AggressiveEndOfWeek && remainingDays <= 2 {
		// 2x on day before reset, 3x on last day
		multiplier = float64(3 - remainingDays)
	}

	nightshiftAllowance := (remainingWeekly / float64(remainingDays)) * float64(maxPercent) / 100 * multiplier

	return &AllowanceResult{
		Allowance:     int64(math.Max(0, nightshiftAllowance)),
		BudgetBase:    int64(remainingWeekly),
		UsedPercent:   usedPercent,
		Mode:          "weekly",
		RemainingDays: remainingDays,
		Multiplier:    multiplier,
	}
}

// applyReserve enforces the reserve percentage on the calculated allowance.
func (m *Manager) applyReserve(result *AllowanceResult, reservePercent int) *AllowanceResult {
	reserveAmount := float64(result.BudgetBase) * float64(reservePercent) / 100
	result.ReserveAmount = int64(reserveAmount)
	result.Allowance = int64(math.Max(0, float64(result.Allowance)-reserveAmount))
	return result
}

// GetUsedPercent retrieves the used percentage from the appropriate provider.
func (m *Manager) GetUsedPercent(provider string) (float64, error) {
	mode := m.cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}

	switch provider {
	case "claude":
		if m.claude == nil {
			return 0, fmt.Errorf("claude provider not configured")
		}
		weeklyBudget := int64(m.cfg.GetProviderBudget(provider))
		return m.claude.GetUsedPercent(mode, weeklyBudget)

	case "codex":
		if m.codex == nil {
			return 0, fmt.Errorf("codex provider not configured")
		}
		return m.codex.GetUsedPercent(mode)

	default:
		return 0, fmt.Errorf("unknown provider: %s", provider)
	}
}

// DaysUntilWeeklyReset calculates days remaining until the weekly budget resets.
// For Claude: assumes weekly reset on Sunday (7 - current weekday, or 7 if Sunday).
// For Codex: uses the secondary rate limit's resets_at timestamp.
func (m *Manager) DaysUntilWeeklyReset(provider string) (int, error) {
	now := m.nowFunc()

	switch provider {
	case "claude":
		// Claude resets weekly; assume Sunday reset
		// Weekday: Sunday=0, Monday=1, ..., Saturday=6
		weekday := int(now.Weekday())
		if weekday == 0 {
			return 7, nil // It's Sunday, next reset in 7 days
		}
		return 7 - weekday, nil

	case "codex":
		if m.codex == nil {
			return 7, nil // Default fallback
		}
		resetTime, err := m.codex.GetResetTime("weekly")
		if err != nil {
			return 7, nil // Fallback on error
		}
		if resetTime.IsZero() {
			return 7, nil // No reset time available
		}

		duration := resetTime.Sub(now)
		days := int(math.Ceil(duration.Hours() / 24))
		if days <= 0 {
			return 1, nil // At least 1 day
		}
		return days, nil

	default:
		return 7, nil // Default for unknown providers
	}
}

// Summary returns a human-readable summary of the budget state for a provider.
func (m *Manager) Summary(provider string) (string, error) {
	result, err := m.CalculateAllowance(provider)
	if err != nil {
		return "", err
	}

	weeklyBudget := m.cfg.GetProviderBudget(provider)

	if result.Mode == "daily" {
		return fmt.Sprintf(
			"%s: %.1f%% used today, %d tokens allowed (daily budget: %d, reserve: %d)",
			provider, result.UsedPercent, result.Allowance, result.BudgetBase, result.ReserveAmount,
		), nil
	}

	return fmt.Sprintf(
		"%s: %.1f%% used this week (%d days left), %d tokens allowed (weekly: %d, remaining: %d, reserve: %d, multiplier: %.1fx)",
		provider, result.UsedPercent, result.RemainingDays, result.Allowance,
		weeklyBudget, result.BudgetBase, result.ReserveAmount, result.Multiplier,
	), nil
}

// CanRun checks if there's enough budget to run a task with the given estimated cost.
func (m *Manager) CanRun(provider string, estimatedTokens int64) (bool, error) {
	result, err := m.CalculateAllowance(provider)
	if err != nil {
		return false, err
	}
	return result.Allowance >= estimatedTokens, nil
}

// Tracker provides backward compatibility for tracking actual spend.
// Deprecated: Use Manager for budget calculations.
type Tracker struct {
	spent map[string]int64
	limit int64
}

// NewTracker creates a budget tracker with the given limit.
// Deprecated: Use NewManager instead.
func NewTracker(limitCents int64) *Tracker {
	return &Tracker{
		spent: make(map[string]int64),
		limit: limitCents,
	}
}

// Record logs spending for a provider.
func (t *Tracker) Record(provider string, tokens int, costCents int64) {
	t.spent[provider] += costCents
}

// Remaining returns cents left in budget.
func (t *Tracker) Remaining() int64 {
	var total int64
	for _, v := range t.spent {
		total += v
	}
	return t.limit - total
}

// NewManagerFromProviders is a convenience constructor that accepts the concrete provider types.
func NewManagerFromProviders(cfg *config.Config, claude *providers.Claude, codex *providers.Codex) *Manager {
	var claudeProvider ClaudeUsageProvider
	var codexProvider CodexUsageProvider

	if claude != nil {
		claudeProvider = claude
	}
	if codex != nil {
		codexProvider = codex
	}

	return NewManager(cfg, claudeProvider, codexProvider)
}
