package snapshots

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/db"
	"github.com/marcusvorwaller/nightshift/internal/tmux"
)

type fakeClaude struct {
	weekly int64
	daily  int64
	err    error
}

func (f fakeClaude) GetWeeklyUsage() (int64, error) { return f.weekly, f.err }
func (f fakeClaude) GetTodayUsage() (int64, error)  { return f.daily, f.err }

type fakeScraper struct {
	claudePct float64
	codexPct  float64
}

func (f fakeScraper) ScrapeClaudeUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.UsageResult{Provider: "claude", WeeklyPct: f.claudePct, ScrapedAt: time.Now()}, nil
}

func (f fakeScraper) ScrapeCodexUsage(ctx context.Context) (tmux.UsageResult, error) {
	return tmux.UsageResult{Provider: "codex", WeeklyPct: f.codexPct, ScrapedAt: time.Now()}, nil
}

type fakeCodex struct {
	files []string
	err   error
}

func (f fakeCodex) ListSessionFiles() ([]string, error) { return f.files, f.err }

func TestTakeSnapshotInsertsClaude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	collector := NewCollector(database, fakeClaude{weekly: 700, daily: 120}, nil, fakeScraper{claudePct: 50}, time.Monday)

	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	latest, err := collector.GetLatest("claude", 1)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(latest))
	}

	snap := latest[0]
	if snap.LocalTokens != 700 {
		t.Fatalf("local tokens = %d", snap.LocalTokens)
	}
	if snap.LocalDaily != 120 {
		t.Fatalf("local daily = %d", snap.LocalDaily)
	}
	if snap.ScrapedPct == nil || *snap.ScrapedPct != 50 {
		t.Fatalf("scraped pct = %v", snap.ScrapedPct)
	}
	if snap.InferredBudget == nil || *snap.InferredBudget != 1400 {
		t.Fatalf("inferred budget = %v", snap.InferredBudget)
	}

	weekStart := startOfWeek(snap.Timestamp, time.Monday)
	if !snap.WeekStart.Equal(weekStart) {
		t.Fatalf("week_start = %v, want %v", snap.WeekStart, weekStart)
	}
}

func TestTakeSnapshotCodexSkipsInferredBudget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	collector := NewCollector(database, nil, fakeCodex{}, fakeScraper{codexPct: 42}, time.Monday)

	snap, err := collector.TakeSnapshot(context.Background(), "codex")
	if err != nil {
		t.Fatalf("take snapshot: %v", err)
	}

	if snap.LocalTokens != 0 {
		t.Fatalf("local tokens = %d, want 0", snap.LocalTokens)
	}
	if snap.ScrapedPct == nil || *snap.ScrapedPct != 42 {
		t.Fatalf("scraped pct = %v, want 42", snap.ScrapedPct)
	}
	// Codex has localWeekly=0, so inferred budget must be nil (not 0).
	if snap.InferredBudget != nil {
		t.Fatalf("inferred budget = %v, want nil (Codex has no local tokens)", snap.InferredBudget)
	}
}

func TestCodexTokenTotalsReturnsZero(t *testing.T) {
	// Codex session files don't contain raw token counts, only rate_limits
	// with used_percent. codexTokenTotals always returns 0.
	weekly, daily, err := codexTokenTotals(fakeCodex{files: []string{"/some/path.jsonl"}})
	if err != nil {
		t.Fatalf("codexTokenTotals: %v", err)
	}
	if weekly != 0 {
		t.Fatalf("weekly tokens = %d, want 0", weekly)
	}
	if daily != 0 {
		t.Fatalf("daily tokens = %d, want 0", daily)
	}
}

func TestPruneSnapshots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	collector := NewCollector(database, fakeClaude{}, nil, nil, time.Monday)

	oldTime := time.Now().AddDate(0, 0, -3)
	weekStart := startOfWeek(oldTime, time.Monday)
	weekNumber, year := weekStart.ISOWeek()
	if _, err := database.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"claude",
		oldTime,
		weekStart,
		10,
		2,
		nil,
		nil,
		int(oldTime.Weekday()),
		oldTime.Hour(),
		weekNumber,
		year,
	); err != nil {
		t.Fatalf("insert old snapshot: %v", err)
	}

	deleted, err := collector.Prune(1)
	if err != nil {
		t.Fatalf("prune snapshots: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 row deleted, got %d", deleted)
	}
}
