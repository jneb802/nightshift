// gemini.go implements the Provider interface for Google Gemini CLI.
package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Gemini wraps the Gemini CLI as a provider.
type Gemini struct {
	dataPath string // Path to ~/.gemini
}

// NewGemini creates a Gemini provider.
func NewGemini() *Gemini {
	home, _ := os.UserHomeDir()
	return &Gemini{
		dataPath: filepath.Join(home, ".gemini"),
	}
}

// NewGeminiWithPath creates a Gemini provider with a custom data path.
func NewGeminiWithPath(dataPath string) *Gemini {
	return &Gemini{
		dataPath: dataPath,
	}
}

// Name returns "gemini".
func (g *Gemini) Name() string {
	return "gemini"
}

// Execute runs a task via Gemini CLI.
func (g *Gemini) Execute(ctx context.Context, task Task) (Result, error) {
	// TODO: Implement - spawn gemini CLI process
	return Result{}, nil
}

// Cost returns Gemini's token pricing (hundredths of a cent per 1K tokens).
// Based on Gemini 2.5 Pro pricing: ~$1.25/M input, ~$10/M output.
func (g *Gemini) Cost() (inputCents, outputCents int64) {
	return 13, 100
}

// DataPath returns the configured data path.
func (g *Gemini) DataPath() string {
	return g.dataPath
}

// GetUsedPercent returns the used percentage based on mode.
// For Gemini, this is a token-based calculation against the weekly budget.
// Returns 0 if no parseable session data is found.
func (g *Gemini) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	switch mode {
	case "daily":
		if weeklyBudget > 0 {
			tokens, err := g.GetTodayTokens()
			if err == nil && tokens > 0 {
				dailyBudget := weeklyBudget / 7
				if dailyBudget > 0 {
					return float64(tokens) / float64(dailyBudget) * 100, nil
				}
			}
		}
		return 0, nil
	case "weekly":
		if weeklyBudget > 0 {
			tokens, err := g.GetWeeklyTokens()
			if err == nil && tokens > 0 {
				return float64(tokens) / float64(weeklyBudget) * 100, nil
			}
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

// GetTodayTokens returns total tokens used today.
// Scans Gemini session files from today's date.
// Returns 0 if no parseable session data is found.
func (g *Gemini) GetTodayTokens() (int64, error) {
	now := time.Now()
	return g.getTokensForDate(now)
}

// GetWeeklyTokens returns total tokens used in the last 7 days.
// Returns 0 if no parseable session data is found.
func (g *Gemini) GetWeeklyTokens() (int64, error) {
	now := time.Now()
	var total int64
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i)
		tokens, err := g.getTokensForDate(date)
		if err != nil {
			continue
		}
		total += tokens
	}
	return total, nil
}

// getTokensForDate returns tokens used on a specific date.
// Gemini CLI stores session data in ~/.gemini/tmp/<hash>/chats/ but the
// exact format may vary. For now, return 0 gracefully — calibration
// snapshots will fill the gap once the user has Gemini sessions.
func (g *Gemini) getTokensForDate(_ time.Time) (int64, error) {
	// Check if the data directory exists at all
	if _, err := os.Stat(g.dataPath); os.IsNotExist(err) {
		return 0, nil
	}
	// TODO: Parse Gemini session files for token data once format is confirmed.
	// The --output-format json response includes stats.models.<name>.tokens
	// which can be parsed from session files.
	return 0, nil
}
