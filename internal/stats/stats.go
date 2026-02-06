// Package stats computes aggregate statistics from nightshift run data.
// It reads from existing report JSONs, run_history, snapshots, and projects tables.
package stats

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/reporting"
)

// Duration wraps time.Duration for clean JSON serialization as seconds.
type Duration struct {
	time.Duration
}

// MarshalJSON serializes Duration as integer seconds.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(d.Duration.Seconds()))
}

// UnmarshalJSON deserializes Duration from integer seconds.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var secs int64
	if err := json.Unmarshal(b, &secs); err != nil {
		return err
	}
	d.Duration = time.Duration(secs) * time.Second
	return nil
}

// String returns a human-readable duration string.
func (d Duration) String() string {
	dur := d.Duration
	if dur < time.Minute {
		return fmt.Sprintf("%ds", int(dur.Seconds()))
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm %ds", int(dur.Minutes()), int(dur.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(dur.Hours()), int(dur.Minutes())%60)
}

// StatsResult holds all computed statistics, JSON-serializable.
type StatsResult struct {
	// Run overview
	TotalRuns      int       `json:"total_runs"`
	FirstRunAt     time.Time `json:"first_run_at,omitempty"`
	LastRunAt      time.Time `json:"last_run_at,omitempty"`
	TotalDuration  Duration  `json:"total_duration"`
	AvgRunDuration Duration  `json:"avg_run_duration"`

	// Task outcomes
	TasksCompleted int     `json:"tasks_completed"`
	TasksFailed    int     `json:"tasks_failed"`
	TasksSkipped   int     `json:"tasks_skipped"`
	SuccessRate    float64 `json:"success_rate"`

	// PR output
	PRsCreated int      `json:"prs_created"`
	PRURLs     []string `json:"pr_urls,omitempty"`

	// Token usage
	TotalTokensUsed int `json:"total_tokens_used"`
	AvgTokensPerRun int `json:"avg_tokens_per_run"`

	// Budget
	BudgetProjection *BudgetProjection `json:"budget_projection,omitempty"`

	// Projects
	TotalProjects    int            `json:"total_projects"`
	ProjectBreakdown []ProjectStats `json:"project_breakdown,omitempty"`

	// Task types
	TaskTypeBreakdown map[string]int `json:"task_type_breakdown,omitempty"`
}

// BudgetProjection estimates remaining budget days from snapshot data.
type BudgetProjection struct {
	Provider         string  `json:"provider"`
	WeeklyBudget     int64   `json:"weekly_budget"`
	CurrentUsedPct   float64 `json:"current_used_pct"`
	AvgDailyUsage    int64   `json:"avg_daily_usage"`
	EstDaysRemaining int     `json:"est_days_remaining"`
	Source           string  `json:"source"`
}

// ProjectStats summarizes activity for a single project.
type ProjectStats struct {
	Name      string `json:"name"`
	RunCount  int    `json:"run_count"`
	TaskCount int    `json:"task_count"`
}

// Stats computes aggregate statistics from nightshift data sources.
type Stats struct {
	db         *db.DB
	reportsDir string
}

// New creates a Stats instance.
func New(database *db.DB, reportsDir string) *Stats {
	return &Stats{
		db:         database,
		reportsDir: reportsDir,
	}
}

// Compute aggregates all available data into a StatsResult.
func (s *Stats) Compute() (*StatsResult, error) {
	result := &StatsResult{
		TaskTypeBreakdown: make(map[string]int),
	}

	// Load report JSONs for task-level stats
	reports := s.loadReports()
	s.computeFromReports(result, reports)

	// Enrich from run_history DB (run count, date range, tokens)
	if s.db != nil {
		s.computeFromRunHistory(result)
		s.computeFromProjects(result)
		s.computeBudgetProjection(result)
	}

	// Compute averages
	if result.TotalRuns > 0 {
		result.AvgRunDuration = Duration{time.Duration(int64(result.TotalDuration.Duration) / int64(result.TotalRuns))}
		if result.TotalTokensUsed > 0 {
			result.AvgTokensPerRun = result.TotalTokensUsed / result.TotalRuns
		}
	}

	// Success rate
	totalTasks := result.TasksCompleted + result.TasksFailed + result.TasksSkipped
	if totalTasks > 0 {
		result.SuccessRate = float64(result.TasksCompleted) / float64(totalTasks) * 100
	}

	return result, nil
}

// loadReports reads all run-*.json files from the reports directory.
func (s *Stats) loadReports() []*reporting.RunResults {
	if s.reportsDir == "" {
		return nil
	}

	entries, err := os.ReadDir(s.reportsDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("stats: read reports dir: %v", err)
		}
		return nil
	}

	var results []*reporting.RunResults
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "run-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(s.reportsDir, name)
		r, err := reporting.LoadRunResults(path)
		if err != nil {
			log.Printf("stats: load report %s: %v", name, err)
			continue
		}
		results = append(results, r)
	}

	// Sort by start time ascending
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartTime.Before(results[j].StartTime)
	})

	return results
}

// computeFromReports extracts task-level stats from report JSON files.
func (s *Stats) computeFromReports(result *StatsResult, reports []*reporting.RunResults) {
	if len(reports) == 0 {
		return
	}

	projectTaskCounts := make(map[string]int)
	prURLSet := make(map[string]struct{})

	for _, r := range reports {
		// Duration
		if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
			result.TotalDuration.Duration += r.EndTime.Sub(r.StartTime)
		}

		// Token usage from report-level budget data
		if r.UsedBudget > 0 {
			result.TotalTokensUsed += r.UsedBudget
		}

		for _, task := range r.Tasks {
			switch task.Status {
			case "completed":
				result.TasksCompleted++
			case "failed":
				result.TasksFailed++
			case "skipped":
				result.TasksSkipped++
			}

			// Task type breakdown
			if task.TaskType != "" {
				result.TaskTypeBreakdown[task.TaskType]++
			}

			// PR detection
			if strings.EqualFold(task.OutputType, "pr") && task.OutputRef != "" {
				result.PRsCreated++
				if strings.HasPrefix(task.OutputRef, "http") {
					prURLSet[task.OutputRef] = struct{}{}
				}
			}

			// Per-project task counts
			if task.Project != "" {
				projectTaskCounts[filepath.Base(task.Project)]++
			}

			// Accumulate tokens from tasks if report-level budget is missing
			if r.UsedBudget == 0 && task.TokensUsed > 0 {
				result.TotalTokensUsed += task.TokensUsed
			}
		}
	}

	// Collect unique PR URLs (most recent first)
	if len(prURLSet) > 0 {
		urls := make([]string, 0, len(prURLSet))
		for url := range prURLSet {
			urls = append(urls, url)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(urls)))
		result.PRURLs = urls
	}

	// Build project breakdown from reports
	for name, count := range projectTaskCounts {
		result.ProjectBreakdown = append(result.ProjectBreakdown, ProjectStats{
			Name:      name,
			TaskCount: count,
		})
	}
	sort.Slice(result.ProjectBreakdown, func(i, j int) bool {
		return result.ProjectBreakdown[i].TaskCount > result.ProjectBreakdown[j].TaskCount
	})
}

// computeFromRunHistory queries the run_history table for run-level stats.
func (s *Stats) computeFromRunHistory(result *StatsResult) {
	sqlDB := s.db.SQL()
	if sqlDB == nil {
		return
	}

	// Total run count
	row := sqlDB.QueryRow(`SELECT COUNT(*) FROM run_history`)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("stats: count run_history: %v", err)
		return
	}
	result.TotalRuns = count

	if count == 0 {
		return
	}

	// First and last run times
	row = sqlDB.QueryRow(`SELECT MIN(start_time), MAX(start_time) FROM run_history`)
	var firstRun, lastRun sql.NullTime
	if err := row.Scan(&firstRun, &lastRun); err != nil {
		log.Printf("stats: run_history min/max: %v", err)
	} else {
		if firstRun.Valid {
			result.FirstRunAt = firstRun.Time
		}
		if lastRun.Valid {
			result.LastRunAt = lastRun.Time
		}
	}

	// Sum tokens from run_history if reports gave us nothing
	if result.TotalTokensUsed == 0 {
		row = sqlDB.QueryRow(`SELECT COALESCE(SUM(tokens_used), 0) FROM run_history`)
		var totalTokens int
		if err := row.Scan(&totalTokens); err != nil {
			log.Printf("stats: sum tokens: %v", err)
		} else {
			result.TotalTokensUsed = totalTokens
		}
	}
}

// computeFromProjects queries the projects table for project count and run counts.
func (s *Stats) computeFromProjects(result *StatsResult) {
	sqlDB := s.db.SQL()
	if sqlDB == nil {
		return
	}

	rows, err := sqlDB.Query(`SELECT path, run_count FROM projects ORDER BY run_count DESC`)
	if err != nil {
		log.Printf("stats: query projects: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	projectRunCounts := make(map[string]int)
	for rows.Next() {
		var path string
		var runCount int
		if err := rows.Scan(&path, &runCount); err != nil {
			log.Printf("stats: scan project: %v", err)
			continue
		}
		name := filepath.Base(path)
		projectRunCounts[name] = runCount
	}
	if err := rows.Err(); err != nil {
		log.Printf("stats: projects rows: %v", err)
	}

	result.TotalProjects = len(projectRunCounts)

	// Merge run_count from projects into existing project breakdown
	existing := make(map[string]*ProjectStats)
	for i := range result.ProjectBreakdown {
		existing[result.ProjectBreakdown[i].Name] = &result.ProjectBreakdown[i]
	}
	for name, runCount := range projectRunCounts {
		if ps, ok := existing[name]; ok {
			ps.RunCount = runCount
		} else {
			result.ProjectBreakdown = append(result.ProjectBreakdown, ProjectStats{
				Name:     name,
				RunCount: runCount,
			})
		}
	}

	// Re-sort by task count descending, then run count
	sort.Slice(result.ProjectBreakdown, func(i, j int) bool {
		if result.ProjectBreakdown[i].TaskCount != result.ProjectBreakdown[j].TaskCount {
			return result.ProjectBreakdown[i].TaskCount > result.ProjectBreakdown[j].TaskCount
		}
		return result.ProjectBreakdown[i].RunCount > result.ProjectBreakdown[j].RunCount
	})
}

// computeBudgetProjection estimates days remaining from recent snapshots.
func (s *Stats) computeBudgetProjection(result *StatsResult) {
	sqlDB := s.db.SQL()
	if sqlDB == nil {
		return
	}

	// Get the most recent snapshot with an inferred budget
	row := sqlDB.QueryRow(
		`SELECT provider, scraped_pct, inferred_budget
		 FROM snapshots
		 WHERE inferred_budget IS NOT NULL
		 ORDER BY timestamp DESC
		 LIMIT 1`,
	)
	var provider string
	var scrapedPct sql.NullFloat64
	var inferredBudget sql.NullInt64
	if err := row.Scan(&provider, &scrapedPct, &inferredBudget); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("stats: latest snapshot: %v", err)
		}
		return
	}

	if !inferredBudget.Valid || inferredBudget.Int64 <= 0 {
		return
	}

	// Compute average daily usage from snapshots in the last 7 days
	cutoff := time.Now().AddDate(0, 0, -7)
	row = sqlDB.QueryRow(
		`SELECT AVG(local_daily)
		 FROM snapshots
		 WHERE provider = ? AND timestamp >= ? AND local_daily > 0`,
		provider,
		cutoff,
	)
	var avgDaily sql.NullFloat64
	if err := row.Scan(&avgDaily); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("stats: avg daily usage: %v", err)
		}
		return
	}

	if !avgDaily.Valid || avgDaily.Float64 <= 0 {
		return
	}

	proj := &BudgetProjection{
		Provider:       provider,
		WeeklyBudget:   inferredBudget.Int64,
		AvgDailyUsage:  int64(avgDaily.Float64),
		Source:         "calibrated",
	}

	if scrapedPct.Valid {
		proj.CurrentUsedPct = scrapedPct.Float64
	}

	// Estimate days remaining: remaining budget / avg daily usage
	remainingBudget := float64(inferredBudget.Int64) * (1 - proj.CurrentUsedPct/100)
	if remainingBudget > 0 && avgDaily.Float64 > 0 {
		proj.EstDaysRemaining = int(remainingBudget / avgDaily.Float64)
	}

	result.BudgetProjection = proj
}
