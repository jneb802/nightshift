package snapshots

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/marcusvorwaller/nightshift/internal/db"
	"github.com/marcusvorwaller/nightshift/internal/providers"
	"github.com/marcusvorwaller/nightshift/internal/tmux"
)

// UsageScraper defines tmux usage scraping behavior.
type UsageScraper interface {
	ScrapeClaudeUsage(ctx context.Context) (tmux.UsageResult, error)
	ScrapeCodexUsage(ctx context.Context) (tmux.UsageResult, error)
}

// ClaudeUsage defines local usage access for Claude.
type ClaudeUsage interface {
	GetWeeklyUsage() (int64, error)
	GetTodayUsage() (int64, error)
}

// CodexUsage defines local usage access for Codex.
type CodexUsage interface {
	ListSessionFiles() ([]string, error)
}

// Snapshot represents a stored usage snapshot.
type Snapshot struct {
	ID             int64
	Provider       string
	Timestamp      time.Time
	WeekStart      time.Time
	LocalTokens    int64
	LocalDaily     int64
	ScrapedPct     *float64
	InferredBudget *int64
	DayOfWeek      int
	HourOfDay      int
	WeekNumber     int
	Year           int
}

// HourlyAverage represents average daily tokens by hour.
type HourlyAverage struct {
	Hour           int
	AvgDailyTokens float64
}

// Collector gathers and stores usage snapshots.
type Collector struct {
	db           *db.DB
	claude       ClaudeUsage
	codex        CodexUsage
	scraper      UsageScraper
	weekStartDay time.Weekday
}

// NewCollector creates a snapshot collector.
func NewCollector(database *db.DB, claude ClaudeUsage, codex CodexUsage, scraper UsageScraper, weekStartDay time.Weekday) *Collector {
	if weekStartDay < time.Sunday || weekStartDay > time.Saturday {
		weekStartDay = time.Monday
	}
	return &Collector{
		db:           database,
		claude:       claude,
		codex:        codex,
		scraper:      scraper,
		weekStartDay: weekStartDay,
	}
}

// TakeSnapshot collects and stores a snapshot for the provider.
func (c *Collector) TakeSnapshot(ctx context.Context, provider string) (Snapshot, error) {
	if c == nil || c.db == nil {
		return Snapshot{}, errors.New("db is nil")
	}

	provider = strings.ToLower(provider)
	now := time.Now()

	var localWeekly, localDaily int64
	var err error
	var scrapedPct *float64

	switch provider {
	case "claude":
		if c.claude == nil {
			return Snapshot{}, errors.New("claude provider is nil")
		}
		localWeekly, err = c.claude.GetWeeklyUsage()
		if err != nil {
			return Snapshot{}, err
		}
		localDaily, err = c.claude.GetTodayUsage()
		if err != nil {
			return Snapshot{}, err
		}
		if c.scraper != nil {
			if result, err := c.scraper.ScrapeClaudeUsage(ctx); err == nil {
				pct := result.WeeklyPct
				if pct >= 0 && pct <= 100 {
					scrapedPct = &pct
				}
			}
		}
	case "codex":
		if c.codex == nil {
			return Snapshot{}, errors.New("codex provider is nil")
		}
		localWeekly, localDaily, err = codexTokenTotals(c.codex)
		if err != nil {
			return Snapshot{}, err
		}
		if c.scraper != nil {
			if result, err := c.scraper.ScrapeCodexUsage(ctx); err == nil {
				pct := result.WeeklyPct
				if pct >= 0 && pct <= 100 {
					scrapedPct = &pct
				}
			}
		}
	default:
		return Snapshot{}, fmt.Errorf("unknown provider: %s", provider)
	}

	weekStart := startOfWeek(now, c.weekStartDay)
	dayOfWeek := int(now.Weekday())
	hourOfDay := now.Hour()
	weekNumber, year := weekStart.ISOWeek()

	var inferredBudget *int64
	if scrapedPct != nil && *scrapedPct > 0 {
		budget := int64(math.Round(float64(localWeekly) / (*scrapedPct / 100)))
		inferredBudget = &budget
	}

	result, err := c.db.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider,
		now,
		weekStart,
		localWeekly,
		localDaily,
		nullFloat(scrapedPct),
		nullInt(inferredBudget),
		dayOfWeek,
		hourOfDay,
		weekNumber,
		year,
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("insert snapshot: %w", err)
	}

	id, _ := result.LastInsertId()

	return Snapshot{
		ID:             id,
		Provider:       provider,
		Timestamp:      now,
		WeekStart:      weekStart,
		LocalTokens:    localWeekly,
		LocalDaily:     localDaily,
		ScrapedPct:     scrapedPct,
		InferredBudget: inferredBudget,
		DayOfWeek:      dayOfWeek,
		HourOfDay:      hourOfDay,
		WeekNumber:     weekNumber,
		Year:           year,
	}, nil
}

// GetLatest returns the latest snapshots for a provider.
func (c *Collector) GetLatest(provider string, n int) ([]Snapshot, error) {
	if n <= 0 {
		return []Snapshot{}, nil
	}
	rows, err := c.db.SQL().Query(
		`SELECT id, provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year
		 FROM snapshots
		 WHERE provider = ?
		 ORDER BY timestamp DESC
		 LIMIT ?`,
		strings.ToLower(provider),
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("query latest snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []Snapshot
	for rows.Next() {
		snapshot, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return snapshots, nil
}

// GetSinceWeekStart returns snapshots from the current week.
func (c *Collector) GetSinceWeekStart(provider string) ([]Snapshot, error) {
	weekStart := startOfWeek(time.Now(), c.weekStartDay)
	rows, err := c.db.SQL().Query(
		`SELECT id, provider, timestamp, week_start, local_tokens, local_daily, scraped_pct, inferred_budget, day_of_week, hour_of_day, week_number, year
		 FROM snapshots
		 WHERE provider = ? AND week_start = ?
		 ORDER BY timestamp ASC`,
		strings.ToLower(provider),
		weekStart,
	)
	if err != nil {
		return nil, fmt.Errorf("query week snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []Snapshot
	for rows.Next() {
		snapshot, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return snapshots, nil
}

// GetHourlyAverages returns average daily tokens per hour.
func (c *Collector) GetHourlyAverages(provider string, lookbackDays int) ([]HourlyAverage, error) {
	if lookbackDays <= 0 {
		return []HourlyAverage{}, nil
	}
	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	rows, err := c.db.SQL().Query(
		`SELECT hour_of_day, AVG(local_daily)
		 FROM snapshots
		 WHERE provider = ? AND timestamp >= ?
		 GROUP BY hour_of_day
		 ORDER BY hour_of_day ASC`,
		strings.ToLower(provider),
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query hourly averages: %w", err)
	}
	defer rows.Close()

	averages := make([]HourlyAverage, 0)
	for rows.Next() {
		var avg HourlyAverage
		if err := rows.Scan(&avg.Hour, &avg.AvgDailyTokens); err != nil {
			return nil, fmt.Errorf("scan hourly average: %w", err)
		}
		averages = append(averages, avg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly averages: %w", err)
	}
	return averages, nil
}

// Prune deletes snapshots older than retentionDays.
func (c *Collector) Prune(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result, err := c.db.SQL().Exec(`DELETE FROM snapshots WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune snapshots: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune snapshots: %w", err)
	}
	return deleted, nil
}

func scanSnapshot(rows *sql.Rows) (Snapshot, error) {
	var snapshot Snapshot
	var scraped sql.NullFloat64
	var inferred sql.NullInt64
	if err := rows.Scan(
		&snapshot.ID,
		&snapshot.Provider,
		&snapshot.Timestamp,
		&snapshot.WeekStart,
		&snapshot.LocalTokens,
		&snapshot.LocalDaily,
		&scraped,
		&inferred,
		&snapshot.DayOfWeek,
		&snapshot.HourOfDay,
		&snapshot.WeekNumber,
		&snapshot.Year,
	); err != nil {
		return Snapshot{}, fmt.Errorf("scan snapshot: %w", err)
	}
	if scraped.Valid {
		snapshot.ScrapedPct = &scraped.Float64
	}
	if inferred.Valid {
		value := inferred.Int64
		snapshot.InferredBudget = &value
	}
	return snapshot, nil
}

func startOfWeek(now time.Time, weekStartDay time.Weekday) time.Time {
	if weekStartDay < time.Sunday || weekStartDay > time.Saturday {
		weekStartDay = time.Monday
	}

	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	delta := (7 + int(now.Weekday()) - int(weekStartDay)) % 7
	return now.AddDate(0, 0, -delta)
}

func nullFloat(value *float64) any {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func nullInt(value *int64) any {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func codexTokenTotals(codex CodexUsage) (int64, int64, error) {
	sessions, err := codex.ListSessionFiles()
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -6)

	var weekly int64
	var daily int64

	for _, path := range sessions {
		date, ok := codexSessionDate(path)
		if !ok {
			continue
		}
		tokens, err := codexSessionTokens(path)
		if err != nil {
			return 0, 0, err
		}
		if !date.Before(weekStart) && !date.After(today) {
			weekly += tokens
		}
		if date.Equal(today) {
			daily += tokens
		}
	}

	return weekly, daily, nil
}

func codexSessionDate(path string) (time.Time, bool) {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i+3 < len(parts); i++ {
		if parts[i] != "sessions" {
			continue
		}
		year, err1 := strconv.Atoi(parts[i+1])
		month, err2 := strconv.Atoi(parts[i+2])
		day, err3 := strconv.Atoi(parts[i+3])
		if err1 != nil || err2 != nil || err3 != nil {
			return time.Time{}, false
		}
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), true
	}
	return time.Time{}, false
}

func codexSessionTokens(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open codex session: %w", err)
	}
	defer file.Close()

	var total int64
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry providers.CodexSessionEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.TokenCount != nil {
			total += *entry.TokenCount
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan codex session: %w", err)
	}
	return total, nil
}
