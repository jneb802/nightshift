package tasks

import (
	"os"
	"testing"

	"github.com/marcusvorwaller/nightshift/internal/config"
	"github.com/marcusvorwaller/nightshift/internal/state"
)

func setupTestSelector(t *testing.T) (*Selector, *state.State, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	st, err := state.New(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create state: %v", err)
	}

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled:    []string{}, // Empty means all enabled
			Disabled:   []string{},
			Priorities: map[string]int{},
		},
	}

	return NewSelector(cfg, st), st, tmpDir
}

func TestScoreTask(t *testing.T) {
	sel, st, tmpDir := setupTestSelector(t)
	defer os.RemoveAll(tmpDir)

	project := "/test/project"

	// Base case: no bonuses
	score := sel.ScoreTask(TaskLintFix, project)
	// Staleness bonus for never-run task is 3.0 (from state.StalenessBonus)
	if score < 2.9 || score > 3.1 {
		t.Errorf("expected score ~3.0 for never-run task, got %f", score)
	}

	// Record a recent run to reduce staleness bonus
	st.RecordTaskRun(project, string(TaskLintFix))
	score = sel.ScoreTask(TaskLintFix, project)
	if score > 0.1 {
		t.Errorf("expected score ~0 for just-run task, got %f", score)
	}

	// Add context mention bonus
	sel.SetContextMentions([]string{string(TaskLintFix)})
	score = sel.ScoreTask(TaskLintFix, project)
	if score < 1.9 || score > 2.1 {
		t.Errorf("expected score ~2.0 with context bonus, got %f", score)
	}

	// Add task source bonus
	sel.SetTaskSources([]string{string(TaskLintFix)})
	score = sel.ScoreTask(TaskLintFix, project)
	if score < 4.9 || score > 5.1 {
		t.Errorf("expected score ~5.0 with context+source bonus, got %f", score)
	}
}

func TestScoreTaskWithConfigPriority(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := state.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	// Set priority in config
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Priorities: map[string]int{
				string(TaskLintFix): 5,
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix)) // Remove staleness bonus

	score := sel.ScoreTask(TaskLintFix, project)
	if score < 4.9 || score > 5.1 {
		t.Errorf("expected score ~5.0 with config priority, got %f", score)
	}
}

func TestFilterEnabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := state.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	tests := []struct {
		name     string
		enabled  []string
		disabled []string
		tasks    []TaskDefinition
		wantLen  int
	}{
		{
			name:    "all enabled by default",
			tasks:   []TaskDefinition{{Type: TaskLintFix}, {Type: TaskBugFinder}},
			wantLen: 2,
		},
		{
			name:     "explicit enabled list",
			enabled:  []string{string(TaskLintFix)},
			tasks:    []TaskDefinition{{Type: TaskLintFix}, {Type: TaskBugFinder}},
			wantLen:  1,
		},
		{
			name:     "disabled takes precedence",
			disabled: []string{string(TaskLintFix)},
			tasks:    []TaskDefinition{{Type: TaskLintFix}, {Type: TaskBugFinder}},
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Tasks: config.TasksConfig{
					Enabled:  tt.enabled,
					Disabled: tt.disabled,
				},
			}
			sel := NewSelector(cfg, st)
			got := sel.FilterEnabled(tt.tasks)
			if len(got) != tt.wantLen {
				t.Errorf("FilterEnabled() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestFilterByBudget(t *testing.T) {
	sel, _, tmpDir := setupTestSelector(t)
	defer os.RemoveAll(tmpDir)

	tasks := []TaskDefinition{
		{Type: TaskLintFix, CostTier: CostLow},         // 10-50k
		{Type: TaskBugFinder, CostTier: CostHigh},      // 150-500k
		{Type: TaskMigrationRehearsal, CostTier: CostVeryHigh}, // 500k+
	}

	tests := []struct {
		budget  int64
		wantLen int
	}{
		{budget: 100_000, wantLen: 1},   // Only low cost fits
		{budget: 500_000, wantLen: 2},   // Low and high fit
		{budget: 1_000_000, wantLen: 3}, // All fit
		{budget: 10_000, wantLen: 0},    // None fit
	}

	for _, tt := range tests {
		got := sel.FilterByBudget(tasks, tt.budget)
		if len(got) != tt.wantLen {
			t.Errorf("FilterByBudget(%d) len = %d, want %d", tt.budget, len(got), tt.wantLen)
		}
	}
}

func TestFilterUnassigned(t *testing.T) {
	sel, st, tmpDir := setupTestSelector(t)
	defer os.RemoveAll(tmpDir)

	project := "/test/project"
	tasks := []TaskDefinition{
		{Type: TaskLintFix},
		{Type: TaskBugFinder},
		{Type: TaskDeadCode},
	}

	// Mark one as assigned
	taskID := makeTaskID(string(TaskLintFix), project)
	st.MarkAssigned(taskID, project, string(TaskLintFix))

	got := sel.FilterUnassigned(tasks, project)
	if len(got) != 2 {
		t.Errorf("FilterUnassigned() len = %d, want 2", len(got))
	}

	// Verify the assigned one is filtered out
	for _, task := range got {
		if task.Type == TaskLintFix {
			t.Error("FilterUnassigned() did not filter out assigned task")
		}
	}
}

func TestSelectNext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := state.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	// Enable only specific tasks for predictable testing
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
			},
			Priorities: map[string]int{
				string(TaskLintFix):     5,
				string(TaskDocsBackfill): 1,
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	// Run both tasks recently to remove staleness bonus
	st.RecordTaskRun(project, string(TaskLintFix))
	st.RecordTaskRun(project, string(TaskDocsBackfill))

	// Select should return highest priority task
	task := sel.SelectNext(100_000, project)
	if task == nil {
		t.Fatal("SelectNext() returned nil")
	}
	if task.Definition.Type != TaskLintFix {
		t.Errorf("SelectNext() = %s, want %s", task.Definition.Type, TaskLintFix)
	}
}

func TestSelectNextNoBudget(t *testing.T) {
	sel, _, tmpDir := setupTestSelector(t)
	defer os.RemoveAll(tmpDir)

	// Budget too low for any task
	task := sel.SelectNext(1000, "/test/project")
	if task != nil {
		t.Errorf("SelectNext() with tiny budget should return nil, got %v", task)
	}
}

func TestSelectAndAssign(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := state.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{string(TaskLintFix)},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"

	// First selection should work
	task1 := sel.SelectAndAssign(100_000, project)
	if task1 == nil {
		t.Fatal("First SelectAndAssign() returned nil")
	}

	// Verify task is now assigned
	taskID := makeTaskID(string(task1.Definition.Type), project)
	if !st.IsAssigned(taskID) {
		t.Error("Task should be marked as assigned")
	}

	// Second selection should return nil (only task is assigned)
	task2 := sel.SelectAndAssign(100_000, project)
	if task2 != nil {
		t.Errorf("Second SelectAndAssign() should return nil, got %v", task2)
	}
}

func TestSelectTopN(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := state.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
				string(TaskDeadCode),
			},
			Priorities: map[string]int{
				string(TaskLintFix):     10,
				string(TaskDocsBackfill): 5,
				string(TaskDeadCode):     1,
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"
	// Run all tasks recently
	st.RecordTaskRun(project, string(TaskLintFix))
	st.RecordTaskRun(project, string(TaskDocsBackfill))
	st.RecordTaskRun(project, string(TaskDeadCode))

	// Get top 2
	tasks := sel.SelectTopN(1_000_000, project, 2)
	if len(tasks) != 2 {
		t.Fatalf("SelectTopN(2) len = %d, want 2", len(tasks))
	}

	// Verify ordering
	if tasks[0].Definition.Type != TaskLintFix {
		t.Errorf("First task should be lint-fix, got %s", tasks[0].Definition.Type)
	}
	if tasks[1].Definition.Type != TaskDocsBackfill {
		t.Errorf("Second task should be docs-backfill, got %s", tasks[1].Definition.Type)
	}

	// Verify scores are descending
	if tasks[0].Score < tasks[1].Score {
		t.Error("Tasks should be sorted by score descending")
	}
}

func TestStalenessAffectsSelection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "selector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := state.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{
				string(TaskLintFix),
				string(TaskDocsBackfill),
			},
			Priorities: map[string]int{
				string(TaskLintFix):     1,  // Lower base priority
				string(TaskDocsBackfill): 1, // Same base priority
			},
		},
	}
	sel := NewSelector(cfg, st)

	project := "/test/project"

	// Run lint-fix recently, never run docs-backfill
	st.RecordTaskRun(project, string(TaskLintFix))
	// docs-backfill never run -> higher staleness bonus

	task := sel.SelectNext(100_000, project)
	if task == nil {
		t.Fatal("SelectNext() returned nil")
	}
	// docs-backfill should win due to staleness bonus (never run = +3.0)
	if task.Definition.Type != TaskDocsBackfill {
		t.Errorf("Stale task should be selected, got %s", task.Definition.Type)
	}
}

func TestMakeTaskID(t *testing.T) {
	id := makeTaskID("lint-fix", "/test/project")
	want := "lint-fix:/test/project"
	if id != want {
		t.Errorf("makeTaskID() = %s, want %s", id, want)
	}
}

func TestSetContextMentions(t *testing.T) {
	sel, st, tmpDir := setupTestSelector(t)
	defer os.RemoveAll(tmpDir)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix))

	// Without context mentions
	score1 := sel.ScoreTask(TaskLintFix, project)

	// With context mentions
	sel.SetContextMentions([]string{string(TaskLintFix)})
	score2 := sel.ScoreTask(TaskLintFix, project)

	if score2-score1 < 1.9 || score2-score1 > 2.1 {
		t.Errorf("Context mention should add ~2.0 to score, got diff %f", score2-score1)
	}
}

func TestSetTaskSources(t *testing.T) {
	sel, st, tmpDir := setupTestSelector(t)
	defer os.RemoveAll(tmpDir)

	project := "/test/project"
	st.RecordTaskRun(project, string(TaskLintFix))

	// Without task sources
	score1 := sel.ScoreTask(TaskLintFix, project)

	// With task sources
	sel.SetTaskSources([]string{string(TaskLintFix)})
	score2 := sel.ScoreTask(TaskLintFix, project)

	if score2-score1 < 2.9 || score2-score1 > 3.1 {
		t.Errorf("Task source should add ~3.0 to score, got diff %f", score2-score1)
	}
}

