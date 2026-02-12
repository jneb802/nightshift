package providers

import (
	"testing"
)

func TestGeminiProvider_Name(t *testing.T) {
	provider := NewGemini()
	if provider.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "gemini")
	}
}

func TestGeminiProvider_Cost(t *testing.T) {
	provider := NewGemini()
	input, output := provider.Cost()
	if input != 13 {
		t.Errorf("input cost = %d, want 13", input)
	}
	if output != 100 {
		t.Errorf("output cost = %d, want 100", output)
	}
}

func TestGeminiProvider_DataPath(t *testing.T) {
	path := "/custom/path"
	provider := NewGeminiWithPath(path)
	if provider.DataPath() != path {
		t.Errorf("DataPath() = %q, want %q", provider.DataPath(), path)
	}
}

func TestGeminiGetUsedPercent_NoData(t *testing.T) {
	provider := NewGeminiWithPath(t.TempDir())
	pct, err := provider.GetUsedPercent("daily", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}
	if pct != 0 {
		t.Errorf("expected 0 for no data, got %.1f", pct)
	}
}

func TestGeminiGetUsedPercent_Weekly_NoData(t *testing.T) {
	provider := NewGeminiWithPath(t.TempDir())
	pct, err := provider.GetUsedPercent("weekly", 700000)
	if err != nil {
		t.Fatalf("GetUsedPercent error: %v", err)
	}
	if pct != 0 {
		t.Errorf("expected 0 for no data, got %.1f", pct)
	}
}

func TestGeminiGetUsedPercent_InvalidMode(t *testing.T) {
	provider := NewGeminiWithPath(t.TempDir())
	_, err := provider.GetUsedPercent("monthly", 700000)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestGeminiGetTodayTokens_NoData(t *testing.T) {
	provider := NewGeminiWithPath(t.TempDir())
	tokens, err := provider.GetTodayTokens()
	if err != nil {
		t.Fatalf("GetTodayTokens error: %v", err)
	}
	if tokens != 0 {
		t.Errorf("expected 0, got %d", tokens)
	}
}

func TestGeminiGetWeeklyTokens_NoData(t *testing.T) {
	provider := NewGeminiWithPath(t.TempDir())
	tokens, err := provider.GetWeeklyTokens()
	if err != nil {
		t.Fatalf("GetWeeklyTokens error: %v", err)
	}
	if tokens != 0 {
		t.Errorf("expected 0, got %d", tokens)
	}
}

func TestGeminiGetTodayTokens_MissingDataPath(t *testing.T) {
	provider := NewGeminiWithPath("/nonexistent/path")
	tokens, err := provider.GetTodayTokens()
	if err != nil {
		t.Fatalf("GetTodayTokens error: %v", err)
	}
	if tokens != 0 {
		t.Errorf("expected 0 for missing path, got %d", tokens)
	}
}
