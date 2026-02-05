package tmux

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	outputs [][]byte
	calls   int
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if len(f.outputs) == 0 {
		return []byte(""), nil
	}
	if f.calls >= len(f.outputs) {
		return f.outputs[len(f.outputs)-1], nil
	}
	out := f.outputs[f.calls]
	f.calls++
	return out, nil
}

func TestStripANSI(t *testing.T) {
	input := "hello \x1b[31mred\x1b[0m world"
	got := StripANSI(input)
	if got != "hello red world" {
		t.Fatalf("StripANSI() = %q", got)
	}
}

func TestWaitForPattern(t *testing.T) {
	runner := &fakeRunner{
		outputs: [][]byte{
			[]byte("no match"),
			[]byte("still no"),
			[]byte("usage 42%"),
		},
	}
	session := NewSession("test", WithRunner(runner))

	ctx := context.Background()
	out, err := session.WaitForPattern(ctx, regexp.MustCompile(`42%`), 50*time.Millisecond, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForPattern error: %v", err)
	}
	if !strings.Contains(out, "42%") {
		t.Fatalf("WaitForPattern output = %q", out)
	}
}

func TestParseClaudeWeeklyPct(t *testing.T) {
	output := `
Current week (all models)   45%
Current week (Sonnet only)  12%
`
	pct, err := parseClaudeWeeklyPct(output)
	if err != nil {
		t.Fatalf("parseClaudeWeeklyPct error: %v", err)
	}
	if pct != 45 {
		t.Fatalf("expected 45, got %v", pct)
	}
}

func TestParseCodexWeeklyPct(t *testing.T) {
	output := `
5h limit: 80% left
Weekly limit: 30% left
`
	pct, err := parseCodexWeeklyPct(output)
	if err != nil {
		t.Fatalf("parseCodexWeeklyPct error: %v", err)
	}
	if pct != 30 {
		t.Fatalf("expected 30, got %v", pct)
	}
}
