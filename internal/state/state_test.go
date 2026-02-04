package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// State file should exist after save
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	statePath := filepath.Join(tmpDir, stateFile)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("state file not created at %s", statePath)
	}
}

func TestProjectRunTracking(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	project := "/path/to/project"

	// Initially not processed
	if s.WasProcessedToday(project) {
		t.Error("WasProcessedToday() = true for new project, want false")
	}

	// Record run
	s.RecordProjectRun(project)

	// Now should be processed today
	if !s.WasProcessedToday(project) {
		t.Error("WasProcessedToday() = false after recording run, want true")
	}

	// LastProjectRun should be recent
	lastRun := s.LastProjectRun(project)
	if time.Since(lastRun) > time.Second {
		t.Errorf("LastProjectRun() = %v, expected recent time", lastRun)
	}
}

func TestTaskRunTracking(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	project := "/path/to/project"
	taskType := "lint"

	// Initially never run
	lastRun := s.LastTaskRun(project, taskType)
	if !lastRun.IsZero() {
		t.Errorf("LastTaskRun() = %v, want zero time for untracked task", lastRun)
	}

	// Record task run
	s.RecordTaskRun(project, taskType)

	// Now should have run time
	lastRun = s.LastTaskRun(project, taskType)
	if time.Since(lastRun) > time.Second {
		t.Errorf("LastTaskRun() = %v, expected recent time", lastRun)
	}
}

func TestDaysSinceLastRun(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	project := "/path/to/project"
	taskType := "lint"

	// Never run returns -1
	days := s.DaysSinceLastRun(project, taskType)
	if days != -1 {
		t.Errorf("DaysSinceLastRun() = %d for never-run task, want -1", days)
	}

	// Record run
	s.RecordTaskRun(project, taskType)

	// Same day returns 0
	days = s.DaysSinceLastRun(project, taskType)
	if days != 0 {
		t.Errorf("DaysSinceLastRun() = %d for today, want 0", days)
	}
}

func TestStalenessBonus(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	project := "/path/to/project"
	taskType := "lint"

	// Never run gets high bonus
	bonus := s.StalenessBonus(project, taskType)
	if bonus != 3.0 {
		t.Errorf("StalenessBonus() = %f for never-run task, want 3.0", bonus)
	}

	// Run today
	s.RecordTaskRun(project, taskType)

	// Today gives 0 bonus
	bonus = s.StalenessBonus(project, taskType)
	if bonus != 0.0 {
		t.Errorf("StalenessBonus() = %f for today, want 0.0", bonus)
	}
}

func TestAssignedTasks(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	taskID := "task-123"
	project := "/path/to/project"
	taskType := "lint"

	// Initially not assigned
	if s.IsAssigned(taskID) {
		t.Error("IsAssigned() = true for new task, want false")
	}

	// Mark assigned
	s.MarkAssigned(taskID, project, taskType)

	// Now should be assigned
	if !s.IsAssigned(taskID) {
		t.Error("IsAssigned() = false after marking, want true")
	}

	// GetAssigned should return info
	info, ok := s.GetAssigned(taskID)
	if !ok {
		t.Error("GetAssigned() ok = false, want true")
	}
	if info.TaskID != taskID {
		t.Errorf("GetAssigned().TaskID = %s, want %s", info.TaskID, taskID)
	}
	if info.TaskType != taskType {
		t.Errorf("GetAssigned().TaskType = %s, want %s", info.TaskType, taskType)
	}

	// Clear assigned
	s.ClearAssigned(taskID)

	// Should no longer be assigned
	if s.IsAssigned(taskID) {
		t.Error("IsAssigned() = true after clearing, want false")
	}
}

func TestClearAllAssigned(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Add multiple assigned tasks
	s.MarkAssigned("task-1", "/project", "lint")
	s.MarkAssigned("task-2", "/project", "docs")

	// Clear all
	s.ClearAllAssigned()

	// None should be assigned
	if s.IsAssigned("task-1") || s.IsAssigned("task-2") {
		t.Error("IsAssigned() = true after ClearAllAssigned(), want false")
	}
}

func TestListAssigned(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Add tasks
	s.MarkAssigned("task-1", "/project", "lint")
	s.MarkAssigned("task-2", "/project", "docs")

	// List should return both
	tasks := s.ListAssigned()
	if len(tasks) != 2 {
		t.Errorf("ListAssigned() returned %d tasks, want 2", len(tasks))
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create state and add data
	s1, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	project := "/path/to/project"
	s1.RecordProjectRun(project)
	s1.RecordTaskRun(project, "lint")
	s1.MarkAssigned("task-123", project, "lint")

	// Save
	if err := s1.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create new state from same dir
	s2, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() second instance error = %v", err)
	}

	// Should have loaded data
	if !s2.WasProcessedToday(project) {
		t.Error("Persistence: WasProcessedToday() = false, want true")
	}

	lastRun := s2.LastTaskRun(project, "lint")
	if lastRun.IsZero() {
		t.Error("Persistence: LastTaskRun() is zero, want recorded time")
	}

	if !s2.IsAssigned("task-123") {
		t.Error("Persistence: IsAssigned() = false, want true")
	}
}

func TestPathNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Record with trailing slash
	s.RecordProjectRun("/path/to/project/")

	// Should match without trailing slash
	if !s.WasProcessedToday("/path/to/project") {
		t.Error("Path normalization: trailing slash not normalized")
	}
}

func TestClearStaleAssignments(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Mark assigned
	s.MarkAssigned("task-1", "/project", "lint")

	// Clear with 0 max age should clear everything
	cleared := s.ClearStaleAssignments(0)
	if cleared != 1 {
		t.Errorf("ClearStaleAssignments() = %d, want 1", cleared)
	}

	if s.IsAssigned("task-1") {
		t.Error("IsAssigned() = true after clearing stale, want false")
	}
}

func TestIsSameDay(t *testing.T) {
	tests := []struct {
		name string
		t1   time.Time
		t2   time.Time
		want bool
	}{
		{
			name: "same day same time",
			t1:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			t2:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "same day different time",
			t1:   time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC),
			t2:   time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
			want: true,
		},
		{
			name: "different day",
			t1:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			t2:   time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "different month",
			t1:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			t2:   time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "different year",
			t1:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			t2:   time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSameDay(tt.t1, tt.t2); got != tt.want {
				t.Errorf("isSameDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectCount(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if s.ProjectCount() != 0 {
		t.Errorf("ProjectCount() = %d, want 0", s.ProjectCount())
	}

	s.RecordProjectRun("/project1")
	s.RecordProjectRun("/project2")

	if s.ProjectCount() != 2 {
		t.Errorf("ProjectCount() = %d, want 2", s.ProjectCount())
	}
}
