package workers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cammy/bigo/pkg/types"
)

// ClaudeWorker executes tasks using Claude Code CLI
type ClaudeWorker struct {
	id        string
	model     string
	backend   types.Backend
	available bool
	cliPath   string
	timeout   time.Duration
}

// ClaudeConfig holds configuration for creating a Claude worker
type ClaudeConfig struct {
	Model   string
	Backend types.Backend
	CLIPath string
	Timeout time.Duration
}

// NewClaudeWorker creates a new Claude worker
func NewClaudeWorker(id string, cfg ClaudeConfig) *ClaudeWorker {
	cliPath := cfg.CLIPath
	if cliPath == "" {
		cliPath = "claude" // Default to PATH lookup
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	return &ClaudeWorker{
		id:        id,
		model:     cfg.Model,
		backend:   cfg.Backend,
		cliPath:   cliPath,
		timeout:   timeout,
		available: true,
	}
}

// Execute runs a task using Claude Code CLI
func (w *ClaudeWorker) Execute(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
	w.available = false
	defer func() { w.available = true }()

	startTime := time.Now()

	// Build the prompt
	prompt := buildClaudePrompt(task)

	// Execute via Claude CLI
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	args := []string{
		"--print",           // Print response only
		"--model", w.model,  // Specify model
	}

	cmd := exec.CommandContext(ctx, w.cliPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &types.ExecutionResult{
				TaskID:  task.ID,
				Backend: w.backend,
				Success: false,
				Error:   fmt.Sprintf("claude exited with code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
			}, nil
		}
		return &types.ExecutionResult{
			TaskID:  task.ID,
			Backend: w.backend,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	duration := time.Since(startTime)

	// Estimate cost based on model and output length
	// These are rough estimates
	cost := estimateCost(w.model, len(prompt), len(output))

	return &types.ExecutionResult{
		TaskID:     task.ID,
		Backend:    w.backend,
		Success:    true,
		Output:     string(output),
		TokensUsed: estimateTokens(len(prompt) + len(output)),
		CostUSD:    cost,
		DurationMs: duration.Milliseconds(),
	}, nil
}

// CheckQuota verifies if the worker has sufficient quota
func (w *ClaudeWorker) CheckQuota(ctx context.Context) error {
	// Try a minimal execution to check if we can access the API
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// "hi" is a minimal prompt to check connectivity and quota
	args := []string{
		"--print",
		"--model", w.model,
		"hi",
	}

	cmd := exec.CommandContext(ctx, w.cliPath, args...)
	// We don't care about the output, just the exit code
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if strings.Contains(strings.ToLower(outputStr), "credit") || 
		   strings.Contains(strings.ToLower(outputStr), "quota") || 
		   strings.Contains(strings.ToLower(outputStr), "balance") ||
		   strings.Contains(strings.ToLower(outputStr), "payment") {
			return fmt.Errorf("quota exceeded or payment required: %v", err)
		}
		// Fallback: any error might indicate an issue, but we want to be specific if possible.
		// For now, if a simple "hi" fails, we assume it's unusable.
		return fmt.Errorf("quota check failed: %v - %s", err, outputStr)
	}
	
	return nil
}

// Available returns whether the worker is available
func (w *ClaudeWorker) Available() bool {
	return w.available
}

// Backend returns the worker's backend type
func (w *ClaudeWorker) Backend() types.Backend {
	return w.backend
}

// CheckHealth verifies the Claude CLI is available
func (w *ClaudeWorker) CheckHealth(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, w.cliPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI not available: %w", err)
	}
	return nil
}

func buildClaudePrompt(task *types.Task) string {
	prompt := task.Title

	if task.Description != "" {
		prompt += "\n\n" + task.Description
	}

	return prompt
}

func estimateCost(model string, inputLen, outputLen int) float64 {
	// Rough token estimate (4 chars per token)
	inputTokens := float64(inputLen) / 4
	outputTokens := float64(outputLen) / 4

	// Pricing per 1K tokens (approximate)
	var inputPrice, outputPrice float64
	switch {
	case strings.Contains(model, "opus"):
		inputPrice = 0.015  // $15/1M input
		outputPrice = 0.075 // $75/1M output
	case strings.Contains(model, "sonnet"):
		inputPrice = 0.003  // $3/1M input
		outputPrice = 0.015 // $15/1M output
	case strings.Contains(model, "haiku"):
		inputPrice = 0.00025 // $0.25/1M input
		outputPrice = 0.00125 // $1.25/1M output
	default:
		inputPrice = 0.003
		outputPrice = 0.015
	}

	return (inputTokens * inputPrice / 1000) + (outputTokens * outputPrice / 1000)
}

func estimateTokens(charCount int) int {
	// Rough estimate: 4 characters per token
	return charCount / 4
}
