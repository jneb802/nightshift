// gemini.go implements the Agent interface for Google Gemini CLI.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GeminiAgent spawns Gemini CLI for task execution.
type GeminiAgent struct {
	binaryPath string        // Path to gemini binary (default: "gemini")
	timeout    time.Duration // Default timeout
	runner     CommandRunner // Command executor (for testing)
	yolo       bool          // Pass --yolo to bypass confirmations
}

// GeminiOption configures a GeminiAgent.
type GeminiOption func(*GeminiAgent)

// WithGeminiBinaryPath sets a custom path to the gemini binary.
func WithGeminiBinaryPath(path string) GeminiOption {
	return func(a *GeminiAgent) {
		a.binaryPath = path
	}
}

// WithGeminiDefaultTimeout sets the default execution timeout.
func WithGeminiDefaultTimeout(d time.Duration) GeminiOption {
	return func(a *GeminiAgent) {
		a.timeout = d
	}
}

// WithGeminiYolo sets whether to pass --yolo to bypass confirmations.
func WithGeminiYolo(enabled bool) GeminiOption {
	return func(a *GeminiAgent) {
		a.yolo = enabled
	}
}

// WithGeminiRunner sets a custom command runner (for testing).
func WithGeminiRunner(r CommandRunner) GeminiOption {
	return func(a *GeminiAgent) {
		a.runner = r
	}
}

// NewGeminiAgent creates a Gemini CLI agent.
func NewGeminiAgent(opts ...GeminiOption) *GeminiAgent {
	a := &GeminiAgent{
		binaryPath: "gemini",
		timeout:    DefaultTimeout,
		runner:     &ExecRunner{},
		yolo:       true,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns "gemini".
func (a *GeminiAgent) Name() string {
	return "gemini"
}

// Execute runs gemini with the given prompt in non-interactive mode.
func (a *GeminiAgent) Execute(ctx context.Context, opts ExecuteOptions) (*ExecuteResult, error) {
	start := time.Now()

	// Determine timeout
	timeout := a.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command args for headless/non-interactive execution
	args := []string{"-p", opts.Prompt}
	if a.yolo {
		args = append(args, "--yolo")
	}
	args = append(args, "--output-format", "text")

	// Build stdin content from files if provided
	var stdinContent string
	if len(opts.Files) > 0 {
		var err error
		stdinContent, err = a.buildFileContext(opts.Files)
		if err != nil {
			return &ExecuteResult{
				Error:    fmt.Sprintf("building file context: %v", err),
				Duration: time.Since(start),
			}, err
		}
	}

	// Run command
	stdout, stderr, exitCode, err := a.runner.Run(ctx, a.binaryPath, args, opts.WorkDir, stdinContent)

	result := &ExecuteResult{
		Output:   stdout,
		ExitCode: exitCode,
		Duration: time.Since(start),
	}

	// Check for context timeout
	if ctx.Err() == context.DeadlineExceeded {
		result.Error = fmt.Sprintf("timeout after %v", timeout)
		result.ExitCode = -1
		return result, ctx.Err()
	}

	// Check for other errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = stderr
		} else {
			result.Error = err.Error()
		}
		return result, err
	}

	// Try to parse JSON output
	result.JSON = a.extractJSON([]byte(stdout))

	return result, nil
}

// buildFileContext reads files and formats them as context.
func (a *GeminiAgent) buildFileContext(files []string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Context Files\n\n")

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}

		// Use absolute path for cleaner output
		displayPath := path
		if abs, err := filepath.Abs(path); err == nil {
			displayPath = abs
		}

		fmt.Fprintf(&sb, "## File: %s\n\n```\n%s\n```\n\n", displayPath, string(content))
	}

	return sb.String(), nil
}

// extractJSON attempts to find and parse JSON from the output.
// Returns nil if no valid JSON found.
func (a *GeminiAgent) extractJSON(output []byte) []byte {
	// Try to parse the entire output as JSON
	if json.Valid(output) {
		return output
	}

	// Look for JSON object or array in output
	start := -1
	var opener, closer byte

	for i, b := range output {
		if b == '{' || b == '[' {
			start = i
			opener = b
			if b == '{' {
				closer = '}'
			} else {
				closer = ']'
			}
			break
		}
	}

	if start == -1 {
		return nil
	}

	// Find matching closer by counting nesting
	depth := 0
	for i := start; i < len(output); i++ {
		if output[i] == opener {
			depth++
		} else if output[i] == closer {
			depth--
			if depth == 0 {
				candidate := output[start : i+1]
				if json.Valid(candidate) {
					return candidate
				}
				break
			}
		}
	}

	return nil
}

// Available checks if the gemini binary is available in PATH.
func (a *GeminiAgent) Available() bool {
	_, err := exec.LookPath(a.binaryPath)
	return err == nil
}

// Version returns the gemini CLI version.
func (a *GeminiAgent) Version() (string, error) {
	cmd := exec.Command(a.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
