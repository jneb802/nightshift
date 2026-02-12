package agents

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewGeminiAgent_Defaults(t *testing.T) {
	agent := NewGeminiAgent()

	if agent.binaryPath != "gemini" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "gemini")
	}
	if agent.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", agent.timeout, DefaultTimeout)
	}
	if agent.runner == nil {
		t.Error("expected non-nil runner")
	}
	if !agent.yolo {
		t.Error("expected yolo to be true by default")
	}
}

func TestNewGeminiAgent_WithOptions(t *testing.T) {
	mockRunner := &MockRunner{}
	agent := NewGeminiAgent(
		WithGeminiBinaryPath("/custom/gemini"),
		WithGeminiDefaultTimeout(5*time.Minute),
		WithGeminiRunner(mockRunner),
		WithGeminiYolo(false),
	)

	if agent.binaryPath != "/custom/gemini" {
		t.Errorf("binaryPath = %q, want %q", agent.binaryPath, "/custom/gemini")
	}
	if agent.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", agent.timeout, 5*time.Minute)
	}
	if agent.runner != mockRunner {
		t.Error("expected custom runner")
	}
	if agent.yolo {
		t.Error("expected yolo to be false")
	}
}

func TestGeminiAgent_Name(t *testing.T) {
	agent := NewGeminiAgent()
	if agent.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "gemini")
	}
}

func TestGeminiAgent_Execute_Success(t *testing.T) {
	mock := &MockRunner{
		Stdout:   "Task completed successfully",
		ExitCode: 0,
	}
	agent := NewGeminiAgent(WithGeminiRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt:  "fix the bug",
		WorkDir: "/project",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !result.IsSuccess() {
		t.Error("expected IsSuccess() to be true")
	}
	if result.Output != "Task completed successfully" {
		t.Errorf("Output = %q, want %q", result.Output, "Task completed successfully")
	}

	// Verify captured values
	if mock.CapturedName != "gemini" {
		t.Errorf("binary = %q, want %q", mock.CapturedName, "gemini")
	}
	wantArgs := []string{"-p", "fix the bug", "--yolo", "--output-format", "text"}
	if len(mock.CapturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", mock.CapturedArgs, wantArgs)
	}
	for i, arg := range wantArgs {
		if mock.CapturedArgs[i] != arg {
			t.Errorf("args[%d] = %q, want %q", i, mock.CapturedArgs[i], arg)
		}
	}
	if mock.CapturedDir != "/project" {
		t.Errorf("dir = %q, want %q", mock.CapturedDir, "/project")
	}
}

func TestGeminiAgent_Execute_NoYolo(t *testing.T) {
	mock := &MockRunner{
		Stdout:   "done",
		ExitCode: 0,
	}
	agent := NewGeminiAgent(WithGeminiRunner(mock), WithGeminiYolo(false))

	_, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "task",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify --yolo is NOT in args
	for _, arg := range mock.CapturedArgs {
		if arg == "--yolo" {
			t.Error("--yolo should not be in args when yolo=false")
		}
	}
}

func TestGeminiAgent_Execute_JSONOutput(t *testing.T) {
	mock := &MockRunner{
		Stdout:   `{"status":"success","files_changed":3}`,
		ExitCode: 0,
	}
	agent := NewGeminiAgent(WithGeminiRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "analyze code",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JSON == nil {
		t.Error("expected JSON to be extracted")
	}
	if string(result.JSON) != `{"status":"success","files_changed":3}` {
		t.Errorf("JSON = %s", result.JSON)
	}
}

func TestGeminiAgent_Execute_Timeout(t *testing.T) {
	mock := &MockRunner{
		Delay: 5 * time.Second,
	}
	agent := NewGeminiAgent(
		WithGeminiRunner(mock),
		WithGeminiDefaultTimeout(50*time.Millisecond),
	)

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "long task",
	})

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("Error = %q, want timeout message", result.Error)
	}
}

func TestGeminiAgent_Execute_ExitError(t *testing.T) {
	mock := &MockRunner{
		Stdout:   "",
		Stderr:   "command failed",
		ExitCode: 1,
		Err:      errors.New("exit status 1"),
	}
	agent := NewGeminiAgent(WithGeminiRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "bad task",
	})

	if err == nil {
		t.Error("expected error")
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.IsSuccess() {
		t.Error("expected IsSuccess() to be false")
	}
}

func TestGeminiAgent_Execute_BinaryNotFound(t *testing.T) {
	mock := &MockRunner{
		Err: errors.New("executable file not found"),
	}
	agent := NewGeminiAgent(
		WithGeminiBinaryPath("/nonexistent/gemini"),
		WithGeminiRunner(mock),
	)

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "test",
	})

	if err == nil {
		t.Error("expected error for missing binary")
	}
	if result == nil {
		t.Fatal("expected result even on error")
		return
	}
	if result.Error == "" {
		t.Error("expected error message in result")
	}
}

func TestGeminiAgent_Execute_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &MockRunner{
		Stdout:   "analyzed file",
		ExitCode: 0,
	}
	agent := NewGeminiAgent(WithGeminiRunner(mock))

	result, err := agent.Execute(context.Background(), ExecuteOptions{
		Prompt: "review code",
		Files:  []string{testFile},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(mock.CapturedStdin, "package main") {
		t.Error("expected file content in stdin")
	}
	if !strings.Contains(mock.CapturedStdin, "# Context Files") {
		t.Error("expected context header in stdin")
	}
	if result.Output != "analyzed file" {
		t.Errorf("Output = %q", result.Output)
	}
}

func TestGeminiAgent_Available(t *testing.T) {
	// Test with known available binary
	agent := NewGeminiAgent(WithGeminiBinaryPath("echo"))
	if !agent.Available() {
		t.Error("expected echo to be available")
	}

	// Test with nonexistent binary
	agent = NewGeminiAgent(WithGeminiBinaryPath("/nonexistent/binary"))
	if agent.Available() {
		t.Error("expected nonexistent binary to not be available")
	}
}

func TestGeminiAgent_Version(t *testing.T) {
	agent := NewGeminiAgent(WithGeminiBinaryPath("/nonexistent/gemini"))
	_, err := agent.Version()
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestGeminiAgent_ImplementsAgentInterface(t *testing.T) {
	var _ Agent = (*GeminiAgent)(nil)
}
