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

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	session := NewSession(uniqueSessionName("claude"), WithSize(120, 40))
	if err := session.Start(ctx); err != nil {
		return UsageResult{}, err
	}
	defer session.Kill(context.Background())

	if err := session.SendKeys(ctx, "claude", "Enter"); err != nil {
		return UsageResult{}, err
	}

	if err := handleClaudeTrustPrompt(ctx, session); err != nil {
		return UsageResult{}, err
	}

	if err := session.SendKeys(ctx, "/usage", "Enter"); err != nil {
		return UsageResult{}, err
	}

	output, err := session.WaitForPattern(ctx, claudeWeekRegex, 10*time.Second, 300*time.Millisecond, "-S", "-200")
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

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	session := NewSession(uniqueSessionName("codex"), WithSize(120, 40))
	if err := session.Start(ctx); err != nil {
		return UsageResult{}, err
	}
	defer session.Kill(context.Background())

	if err := session.SendKeys(ctx, "codex", "Enter"); err != nil {
		return UsageResult{}, err
	}

	if err := handleCodexPrompts(ctx, session); err != nil {
		return UsageResult{}, err
	}

	if err := session.SendKeys(ctx, "/status", "Enter"); err != nil {
		return UsageResult{}, err
	}

	output, err := session.WaitForPattern(ctx, codexWeekRegex, 10*time.Second, 300*time.Millisecond, "-S", "-200")
	if err != nil {
		return UsageResult{}, err
	}

	clean := StripANSI(output)
	weeklyPct, err := parseCodexWeeklyPct(clean)
	if err != nil {
		return UsageResult{}, err
	}

	return UsageResult{
		Provider:  "codex",
		WeeklyPct: weeklyPct,
		ScrapedAt: time.Now(),
		RawOutput: clean,
	}, nil
}

var percentRegex = regexp.MustCompile(`\b\d{1,3}%\b`)
var claudeWeekRegex = regexp.MustCompile(`(?i)current\s+week`)
var codexWeekRegex = regexp.MustCompile(`(?i)weekly\s+limit`)

func parseClaudeWeeklyPct(output string) (float64, error) {
	output = StripANSI(output)
	re := regexp.MustCompile(`(?i)current\s+week[^\n]*?(\d{1,3})%`)
	if match := re.FindStringSubmatch(output); len(match) == 2 {
		return parsePct(match[1])
	}
	if match := percentRegex.FindStringSubmatch(output); len(match) == 1 {
		return parsePct(strings.TrimSuffix(match[0], "%"))
	}
	return 0, errors.New("claude weekly usage percent not found")
}

func parseCodexWeeklyPct(output string) (float64, error) {
	output = StripANSI(output)
	re := regexp.MustCompile(`(?i)weekly\s+limit[^\n]*?(\d{1,3})%`)
	if match := re.FindStringSubmatch(output); len(match) == 2 {
		return parsePct(match[1])
	}
	if match := percentRegex.FindStringSubmatch(output); len(match) == 1 {
		return parsePct(strings.TrimSuffix(match[0], "%"))
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

func handleClaudeTrustPrompt(ctx context.Context, session *Session) error {
	output, err := session.CapturePane(ctx, "-S", "-50")
	if err != nil {
		return err
	}
	if strings.Contains(output, "Do you trust") {
		return session.SendKeys(ctx, "Enter")
	}
	return nil
}

func handleCodexPrompts(ctx context.Context, session *Session) error {
	output, err := session.CapturePane(ctx, "-S", "-50")
	if err != nil {
		return err
	}

	if strings.Contains(output, "Update available") {
		if err := session.SendKeys(ctx, "Down", "Enter"); err != nil {
			return err
		}
		output, _ = session.CapturePane(ctx, "-S", "-50")
	}

	if strings.Contains(output, "allow Codex to work") {
		if err := session.SendKeys(ctx, "Enter"); err != nil {
			return err
		}
	}

	return nil
}
