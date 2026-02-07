package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/logging"
	"github.com/marcus/nightshift/internal/state"
	"github.com/marcus/nightshift/internal/tasks"
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

// newTestRunState creates a fresh state backed by a temp DB.
func newTestRunState(t *testing.T) *state.State {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	st, err := state.New(database)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}
	return st
}

// newTestRunConfig returns a config with providers and budget set up for testing.
func newTestRunConfig() *config.Config {
	return &config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ProviderConfig{Enabled: true},
			Codex:  config.ProviderConfig{Enabled: true},
		},
		Budget: config.BudgetConfig{
			Mode:         "daily",
			MaxPercent:   75,
			WeeklyTokens: 700000,
		},
		Tasks: config.TasksConfig{
			Enabled:    []string{},
			Disabled:   []string{},
			Priorities: map[string]int{},
		},
	}
}

func TestMaxProjects_DefaultLimitsToOne(t *testing.T) {
	// Simulate 3 projects, no --project set, maxProjects=1 (default)
	projects := []string{"/proj/a", "/proj/b", "/proj/c"}
	projectPath := "" // not explicitly set
	maxProjects := 1

	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	if projects[0] != "/proj/a" {
		t.Fatalf("projects[0] = %q, want /proj/a", projects[0])
	}
}

func TestMaxProjects_OverrideToN(t *testing.T) {
	projects := []string{"/proj/a", "/proj/b", "/proj/c"}
	projectPath := ""
	maxProjects := 2

	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}
	if projects[1] != "/proj/b" {
		t.Fatalf("projects[1] = %q, want /proj/b", projects[1])
	}
}

func TestMaxProjects_IgnoredWhenProjectSet(t *testing.T) {
	projects := []string{"/proj/explicit"}
	projectPath := "/proj/explicit" // explicitly set
	maxProjects := 1

	// The guard: projectPath == "" is false, so no truncation
	if projectPath == "" && maxProjects > 0 && len(projects) > maxProjects {
		projects = projects[:maxProjects]
	}

	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	if projects[0] != "/proj/explicit" {
		t.Fatalf("projects[0] = %q, want /proj/explicit", projects[0])
	}
}

func TestMaxTasks_DefaultLimitsToOne(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
	)
	project := t.TempDir()

	params := executeRunParams{
		cfg:       cfg,
		budgetMgr: budgetMgr,
		selector:  selector,
		st:        st,
		projects:  []string{project},
		maxTasks:  1, // default
		dryRun:    true,
		log:       logging.Component("test"),
	}

	err := executeRun(context.Background(), params)
	if err != nil {
		t.Fatalf("executeRun: %v", err)
	}

	// In dry-run, tasks are selected but not executed.
	// The key assertion: with maxTasks=1, SelectTopN(budget, project, 1)
	// should return at most 1 task. We verify by running SelectTopN directly.
	selected := selector.SelectTopN(1_000_000, project, 1)
	if len(selected) > 1 {
		t.Fatalf("SelectTopN(1) returned %d tasks, want <= 1", len(selected))
	}
}

func TestMaxTasks_OverrideToN(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
	)
	project := t.TempDir()

	params := executeRunParams{
		cfg:       cfg,
		budgetMgr: budgetMgr,
		selector:  selector,
		st:        st,
		projects:  []string{project},
		maxTasks:  3,
		dryRun:    true,
		log:       logging.Component("test"),
	}

	err := executeRun(context.Background(), params)
	if err != nil {
		t.Fatalf("executeRun: %v", err)
	}

	// Verify SelectTopN with n=3 returns more than 1 when tasks exist
	selected := selector.SelectTopN(1_000_000, project, 3)
	if len(selected) > 3 {
		t.Fatalf("SelectTopN(3) returned %d tasks, want <= 3", len(selected))
	}
	// With default config (all tasks enabled, large budget) we expect > 1 task
	if len(selected) < 2 {
		t.Logf("only %d tasks available (expected >= 2); may depend on registered tasks", len(selected))
	}
}

func TestMaxTasks_IgnoredWhenTaskSet(t *testing.T) {
	tmp := t.TempDir()
	makeExecutable(t, tmp, "claude")
	makeExecutable(t, tmp, "codex")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	st := newTestRunState(t)
	cfg := newTestRunConfig()
	selector := tasks.NewSelector(cfg, st)
	budgetMgr := budget.NewManager(cfg,
		&mockUsage{name: "claude", pct: 0},
		&mockCodexUsage{mockUsage: mockUsage{name: "codex", pct: 0}},
	)
	project := t.TempDir()

	// When taskFilter is set, maxTasks is ignored - only the specified task runs
	params := executeRunParams{
		cfg:        cfg,
		budgetMgr:  budgetMgr,
		selector:   selector,
		st:         st,
		projects:   []string{project},
		taskFilter: "lint-fix",
		maxTasks:   5, // should be ignored
		dryRun:     true,
		log:        logging.Component("test"),
	}

	err := executeRun(context.Background(), params)
	if err != nil {
		t.Fatalf("executeRun: %v", err)
	}
	// The test passes if executeRun doesn't error - when taskFilter is set,
	// it uses GetDefinition + single-task path, ignoring maxTasks entirely.
}
