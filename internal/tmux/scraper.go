package tmux

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrTmuxNotFound indicates tmux is not installed.
	ErrTmuxNotFound = errors.New("tmux not found")
)

// UsageResult captures scraped usage metadata.
type UsageResult struct {
	Provider  string
	WeeklyPct float64
	ScrapedAt time.Time
	RawOutput string
}

// ScrapeClaudeUsage starts Claude in tmux, runs /usage, and parses weekly usage percent.
func ScrapeClaudeUsage(ctx context.Context) (UsageResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return UsageResult{}, ErrTmuxNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	session := NewSession(uniqueSessionName("claude"), WithSize(120, 40))
	if err := session.Start(ctx); err != nil {
		return UsageResult{}, err
	}
	defer session.Kill(context.Background())

	// Launch Claude Code
	if err := session.SendKeys(ctx, "claude", "Enter"); err != nil {
		return UsageResult{}, err
	}

	// Wait for the TUI to render before sending any commands
	startupOutput, err := waitForSubstantialContent(ctx, session, 20*time.Second)
	if err != nil {
		return UsageResult{}, fmt.Errorf("claude startup: %w", err)
	}

	// Handle trust prompt if present in startup output
	if strings.Contains(StripANSI(startupOutput), "Do you trust") {
		if err := session.SendKeys(ctx, "Enter"); err != nil {
			return UsageResult{}, err
		}
		if err := ctxSleep(ctx, 3*time.Second); err != nil {
			return UsageResult{}, err
		}
	}

	// Brief pause to ensure CLI is ready for input
	if err := ctxSleep(ctx, 1*time.Second); err != nil {
		return UsageResult{}, err
	}

	// Type /usage and wait for autocomplete to populate before pressing Enter.
	// Claude Code shows a command picker when slash commands are typed.
	if err := session.SendKeys(ctx, "/usage"); err != nil {
		return UsageResult{}, err
	}
	if err := ctxSleep(ctx, 500*time.Millisecond); err != nil {
		return UsageResult{}, err
	}
	if err := session.SendKeys(ctx, "Enter"); err != nil {
		return UsageResult{}, err
	}

	// Wait for usage output
	output, err := session.WaitForPattern(ctx, claudeWeekRegex, 15*time.Second, 300*time.Millisecond, "-S", "-200")
	if err != nil {
		return UsageResult{}, err
	}

	clean := StripANSI(output)
	weeklyPct, err := parseClaudeWeeklyPct(clean)
	if err != nil {
		return UsageResult{}, err
	}

	return UsageResult{
		Provider:  "claude",
		WeeklyPct: weeklyPct,
		ScrapedAt: time.Now(),
		RawOutput: clean,
	}, nil
}

// ScrapeCodexUsage starts Codex in tmux, runs /status, and parses weekly usage percent.
func ScrapeCodexUsage(ctx context.Context) (UsageResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return UsageResult{}, ErrTmuxNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	session := NewSession(uniqueSessionName("codex"), WithSize(120, 40))
	if err := session.Start(ctx); err != nil {
		return UsageResult{}, err
	}
	defer session.Kill(context.Background())

	// Launch Codex
	if err := session.SendKeys(ctx, "codex", "Enter"); err != nil {
		return UsageResult{}, err
	}

	// Wait for the TUI to render
	startupOutput, err := waitForSubstantialContent(ctx, session, 20*time.Second)
	if err != nil {
		return UsageResult{}, fmt.Errorf("codex startup: %w", err)
	}

	// Handle Codex-specific prompts from startup output
	clean := StripANSI(startupOutput)
	if strings.Contains(clean, "Update available") {
		if err := session.SendKeys(ctx, "Down", "Enter"); err != nil {
			return UsageResult{}, err
		}
		if err := ctxSleep(ctx, 3*time.Second); err != nil {
			return UsageResult{}, err
		}
		// Re-capture after update prompt dismissed
		startupOutput, _ = session.CapturePane(ctx, "-S", "-50")
		clean = StripANSI(startupOutput)
	}
	if strings.Contains(clean, "allow Codex to work") {
		if err := session.SendKeys(ctx, "Enter"); err != nil {
			return UsageResult{}, err
		}
		if err := ctxSleep(ctx, 3*time.Second); err != nil {
			return UsageResult{}, err
		}
	}

	// Brief pause to ensure CLI is ready for input
	if err := ctxSleep(ctx, 1*time.Second); err != nil {
		return UsageResult{}, err
	}

	// Type /status and wait for autocomplete before pressing Enter.
	if err := session.SendKeys(ctx, "/status"); err != nil {
		return UsageResult{}, err
	}
	if err := ctxSleep(ctx, 500*time.Millisecond); err != nil {
		return UsageResult{}, err
	}
	if err := session.SendKeys(ctx, "Enter"); err != nil {
		return UsageResult{}, err
	}

	// Wait for status output
	output, err := session.WaitForPattern(ctx, codexWeekRegex, 15*time.Second, 300*time.Millisecond, "-S", "-200")
	if err != nil {
		return UsageResult{}, err
	}

	cleanOutput := StripANSI(output)
	weeklyPct, err := parseCodexWeeklyPct(cleanOutput)
	if err != nil {
		return UsageResult{}, err
	}

	return UsageResult{
		Provider:  "codex",
		WeeklyPct: weeklyPct,
		ScrapedAt: time.Now(),
		RawOutput: cleanOutput,
	}, nil
}

var claudeWeekRegex = regexp.MustCompile(`(?i)current\s+week`)
var codexWeekRegex = regexp.MustCompile(`(?i)weekly\s+limit`)

func parseClaudeWeeklyPct(output string) (float64, error) {
	output = StripANSI(output)
	// Match "Current week" followed by a percentage, possibly on the next line.
	// The (?s) flag makes . match newlines so the pattern crosses lines.
	re := regexp.MustCompile(`(?is)current\s+week\s*\(all\s+models\).*?(\d{1,3})%`)
	if match := re.FindStringSubmatch(output); len(match) == 2 {
		return parsePct(match[1])
	}
	// Fallback: any "Current week" header followed by a percentage
	re2 := regexp.MustCompile(`(?is)current\s+week.*?(\d{1,3})%`)
	if match := re2.FindStringSubmatch(output); len(match) == 2 {
		return parsePct(match[1])
	}
	return 0, errors.New("claude weekly usage percent not found")
}

func parseCodexWeeklyPct(output string) (float64, error) {
	output = StripANSI(output)
	// Codex /status shows "77% left" -- extract the number and qualifier.
	re := regexp.MustCompile(`(?i)weekly\s+limit[^\n]*?(\d{1,3})%\s*(left|used)?`)
	if match := re.FindStringSubmatch(output); len(match) >= 2 {
		pct, err := parsePct(match[1])
		if err != nil {
			return 0, err
		}
		// Convert "left" to "used" percentage
		if len(match) >= 3 && strings.EqualFold(match[2], "left") {
			pct = 100 - pct
		}
		return pct, nil
	}
	return 0, errors.New("codex weekly usage percent not found")
}

func parsePct(value string) (float64, error) {
	pct, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("parse percent: %w", err)
	}
	if pct < 0 || pct > 100 {
		return 0, fmt.Errorf("percent out of range: %d", pct)
	}
	return float64(pct), nil
}

func uniqueSessionName(provider string) string {
	return fmt.Sprintf("nightshift-usage-%s-%d", provider, time.Now().UnixNano())
}

// waitForSubstantialContent polls the pane until it has more than a bare
// shell prompt's worth of content, indicating the CLI TUI has rendered.
func waitForSubstantialContent(ctx context.Context, session *Session, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastOutput string
	for {
		select {
		case <-ctx.Done():
			return lastOutput, fmt.Errorf("timeout waiting for CLI (%d non-empty lines seen)",
				countNonEmptyLines(StripANSI(lastOutput)))
		case <-ticker.C:
			output, err := session.CapturePane(ctx, "-S", "-50")
			if err != nil {
				continue
			}
			lastOutput = output
			if countNonEmptyLines(StripANSI(output)) > 5 {
				return output, nil
			}
		}
	}
}

// countNonEmptyLines returns the number of non-blank lines in s.
func countNonEmptyLines(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// ctxSleep pauses for d or until ctx is cancelled.
func ctxSleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
