package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/setup"
	"github.com/marcus/nightshift/internal/tasks"
)

func TestApplyBudgetEdit_MaxPercentBounds(t *testing.T) {
	m := &setupModel{
		cfg:         &config.Config{},
		budgetInput: textinput.New(),
	}

	m.budgetCursor = 1
	m.budgetInput.SetValue("101")
	if err := m.applyBudgetEdit(); err == nil {
		t.Fatal("expected max_percent > 100 to fail")
	}

	m.budgetInput.SetValue("100")
	if err := m.applyBudgetEdit(); err != nil {
		t.Fatalf("expected max_percent=100 to pass: %v", err)
	}
}

func TestApplyBudgetEdit_ReservePercentBounds(t *testing.T) {
	m := &setupModel{
		cfg:         &config.Config{},
		budgetInput: textinput.New(),
	}

	m.budgetCursor = 2
	m.budgetInput.SetValue("101")
	if err := m.applyBudgetEdit(); err == nil {
		t.Fatal("expected reserve_percent > 100 to fail")
	}

	m.budgetInput.SetValue("100")
	if err := m.applyBudgetEdit(); err != nil {
		t.Fatalf("expected reserve_percent=100 to pass: %v", err)
	}
}

func TestHandleProjectsInput_RejectsFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	m := &setupModel{
		projectEditing: true,
		projectInput:   textinput.New(),
	}
	m.projectInput.SetValue(filePath)

	model, _ := m.handleProjectsInput(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(*setupModel)
	if got.projectErr != "path must be a directory" {
		t.Fatalf("projectErr = %q, want %q", got.projectErr, "path must be a directory")
	}
}

func TestHandleTaskInput_RequiresSelection(t *testing.T) {
	m := &setupModel{
		taskItems: []taskItem{
			{
				def:      tasks.TaskDefinition{Type: tasks.TaskType("unit-test-task")},
				selected: false,
			},
		},
	}

	model, cmd := m.handleTaskInput(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(*setupModel)
	if cmd != nil {
		t.Fatal("expected no transition cmd when no tasks selected")
	}
	if got.taskErr != "select at least one task" {
		t.Fatalf("taskErr = %q, want %q", got.taskErr, "select at least one task")
	}
}

func TestHandleTaskInput_NoTasksDoesNotPanic(t *testing.T) {
	m := &setupModel{}
	if _, _ = m.handleTaskInput(tea.KeyMsg{Type: tea.KeySpace}); m.taskErr != "" {
		t.Fatalf("taskErr = %q, want empty for non-enter input", m.taskErr)
	}
}

func TestEnsurePathInShell_SubstringDoesNotBlockInsert(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(cfgPath, []byte("export PATH=\"$PATH:/opt/bin2\"\n"), 0644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	changed, err := ensurePathInShell(cfgPath, "zsh", "/opt/bin")
	if err != nil {
		t.Fatalf("ensurePathInShell: %v", err)
	}
	if !changed {
		t.Fatal("expected config to change when only substring path exists")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !shellConfigHasPath(string(data), "/opt/bin") {
		t.Fatal("expected new path token to be present")
	}
}

func TestEnsurePathInShell_ExactPathNoChange(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(cfgPath, []byte("export PATH=\"$PATH:/opt/bin\"\n"), 0644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	changed, err := ensurePathInShell(cfgPath, "zsh", "/opt/bin")
	if err != nil {
		t.Fatalf("ensurePathInShell: %v", err)
	}
	if changed {
		t.Fatal("expected no change when exact path token exists")
	}
}

func TestMakeTaskItems_UsesSortedDefinitions(t *testing.T) {
	cfg := &config.Config{}
	items := makeTaskItems(cfg, nil, setup.PresetBalanced)
	defs := tasks.AllDefinitionsSorted()

	if len(items) != len(defs) {
		t.Fatalf("len(items)=%d len(defs)=%d", len(items), len(defs))
	}

	for i := range defs {
		if items[i].def.Type != defs[i].Type {
			t.Fatalf("item[%d].Type=%q want %q", i, items[i].def.Type, defs[i].Type)
		}
	}
}

func TestMakeTaskItems_PreservesExplicitEnabledTasks(t *testing.T) {
	cfg := &config.Config{
		Tasks: config.TasksConfig{
			Enabled: []string{string(tasks.TaskBugFinder)},
		},
	}

	items := makeTaskItems(cfg, nil, setup.PresetBalanced)
	found := false
	for _, item := range items {
		if item.def.Type != tasks.TaskBugFinder {
			continue
		}
		found = true
		if !item.selected {
			t.Fatal("expected explicitly enabled task to remain selected")
		}
	}
	if !found {
		t.Fatal("expected bug-finder task to exist in setup list")
	}
}

func TestRenderEnvChecks_IncludesGemini(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			Gemini: config.ProviderConfig{
				Enabled:  true,
				DataPath: dir,
			},
		},
	}
	out := renderEnvChecks(cfg)
	if !strings.Contains(out, "Gemini data path") {
		t.Fatalf("expected Gemini data path check in output, got:\n%s", out)
	}
}

func TestHandleSafetyInput_TogglesGemini(t *testing.T) {
	m := &setupModel{
		cfg:          &config.Config{},
		safetyCursor: 2,
	}

	if m.cfg.Providers.Gemini.Yolo {
		t.Fatal("expected Yolo to start as false")
	}

	m.handleSafetyInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.cfg.Providers.Gemini.Yolo {
		t.Fatal("expected Yolo to be toggled to true")
	}

	m.handleSafetyInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if m.cfg.Providers.Gemini.Yolo {
		t.Fatal("expected Yolo to be toggled back to false")
	}
}
