package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cammy/bigo/pkg/types"
)

// OllamaWorker executes tasks using an Ollama endpoint
type OllamaWorker struct {
	id           string
	endpoint     string
	model        string
	backend      types.Backend
	client       *http.Client
	available    bool
	opencodePath string
}

// OllamaConfig holds configuration for creating an Ollama worker
type OllamaConfig struct {
	Endpoint     string
	Model        string
	Backend      types.Backend
	OpenCodePath string
	Timeout      time.Duration
}

// NewOllamaWorker creates a new Ollama worker
func NewOllamaWorker(id string, cfg OllamaConfig) *OllamaWorker {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	return &OllamaWorker{
		id:           id,
		endpoint:     cfg.Endpoint,
		model:        cfg.Model,
		backend:      cfg.Backend,
		opencodePath: cfg.OpenCodePath,
		client: &http.Client{
			Timeout: timeout,
		},
		available: true,
	}
}

// Execute runs a task using Ollama
func (w *OllamaWorker) Execute(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
	w.available = false
	defer func() { w.available = true }()

	startTime := time.Now()

	// Build the prompt
	prompt := buildTaskPrompt(task)

	// Call Ollama API
	response, err := w.generate(ctx, prompt)
	if err != nil {
		return &types.ExecutionResult{
			TaskID:  task.ID,
			Backend: w.backend,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	duration := time.Since(startTime)

	return &types.ExecutionResult{
		TaskID:     task.ID,
		Backend:    w.backend,
		Success:    true,
		Output:     response.Response,
		TokensUsed: response.TotalTokens(),
		CostUSD:    0, // Ollama is free
		DurationMs: duration.Milliseconds(),
	}, nil
}

// Available returns whether the worker is available
func (w *OllamaWorker) Available() bool {
	return w.available
}

// Backend returns the worker's backend type
func (w *OllamaWorker) Backend() types.Backend {
	return w.backend
}

// CheckHealth verifies the Ollama endpoint is reachable
func (w *OllamaWorker) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", w.endpoint+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}

	return nil
}

// ollamaRequest represents a request to the Ollama generate API
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaResponse represents a response from the Ollama generate API
type ollamaResponse struct {
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	EvalCount       int    `json:"eval_count"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	TotalDuration   int64  `json:"total_duration"`
}

func (r *ollamaResponse) TotalTokens() int {
	return r.EvalCount + r.PromptEvalCount
}

func (w *OllamaWorker) generate(ctx context.Context, prompt string) (*ollamaResponse, error) {
	reqBody := ollamaRequest{
		Model:  w.model,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ollamaResp, nil
}

func buildTaskPrompt(task *types.Task) string {
	prompt := fmt.Sprintf(`You are an expert software engineer. Complete the following task:

## Task
%s

`, task.Title)

	if task.Description != "" {
		prompt += fmt.Sprintf(`## Details
%s

`, task.Description)
	}

	prompt += `## Instructions
- Provide clear, working code
- Include brief explanations for non-obvious decisions
- If the task is ambiguous, state your assumptions
- Format code properly with appropriate language tags

## Response
`

	return prompt
}
