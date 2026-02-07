package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/logging"
)

type mockUsage struct {
	name string
	pct  float64
}

func (m *mockUsage) Name() string { return m.name }

func (m *mockUsage) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	return m.pct, nil
}

type mockCodexUsage struct {
	mockUsage
}

func (m *mockCodexUsage) GetResetTime(mode string) (time.Time, error) {
	return time.Time{}, nil
}

func TestSelectProvider_PreferenceOrder(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+origPath)

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Preference: []string{"codex", "claude"},
			Claude:     config.ProviderConfig{Enabled: true},
			Codex:      config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}

	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex)

	choice, err := selectProvider(cfg, budgetMgr, logging.Component("test"))
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if choice.name != "codex" {
		t.Fatalf("provider = %s, want codex", choice.name)
	}
}

func TestSelectProvider_FallbackOnBudget(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+origPath)

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Preference: []string{"codex", "claude"},
			Claude:     config.ProviderConfig{Enabled: true},
			Codex:      config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}

	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 100}}
	budgetMgr := budget.NewManager(cfg, claude, codex)

	choice, err := selectProvider(cfg, budgetMgr, logging.Component("test"))
	if err != nil {
		t.Fatalf("selectProvider error: %v", err)
	}
	if choice.name != "claude" {
		t.Fatalf("provider = %s, want claude", choice.name)
	}
}

func TestSelectProvider_NoProvidersEnabled(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: false},
			Codex:  config.ProviderConfig{Enabled: false},
		},
	}
	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "no providers enabled in config" {
		t.Fatalf("error = %q, want %q", got, "no providers enabled in config")
	}
}

func TestSelectProvider_AllBudgetExhausted(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	claude := &mockUsage{name: "claude", pct: 100}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 100}}
	budgetMgr := budget.NewManager(cfg, claude, codex)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "budget exhausted") {
		t.Fatalf("error = %q, want it to contain 'budget exhausted'", got)
	}
}

func TestSelectProvider_CLINotInPath(t *testing.T) {
	// Empty PATH so no CLIs are found
	t.Setenv("PATH", t.TempDir())

	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
	}
	claude := &mockUsage{name: "claude", pct: 0}
	codex := &mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}}
	budgetMgr := budget.NewManager(cfg, claude, codex)

	_, err := selectProvider(cfg, budgetMgr, logging.Component("test"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "CLI not in PATH") {
		t.Fatalf("error = %q, want it to contain 'CLI not in PATH'", got)
	}
}

func makeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
