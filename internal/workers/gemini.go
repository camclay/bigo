package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cammy/bigo/pkg/types"
)

// GeminiWorker executes tasks using Google's Gemini API
type GeminiWorker struct {
	id        string
	apiKey    string
	model     string
	backend   types.Backend
	client    *http.Client
	available bool
}

// GeminiConfig holds configuration for creating a Gemini worker
type GeminiConfig struct {
	APIKey  string
	Model   string
	Backend types.Backend
	Timeout time.Duration
}

// NewGeminiWorker creates a new Gemini worker
func NewGeminiWorker(id string, cfg GeminiConfig) *GeminiWorker {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	return &GeminiWorker{
		id:      id,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		backend: cfg.Backend,
		client: &http.Client{
			Timeout: timeout,
		},
		available: true,
	}
}

// Execute runs a task using Gemini
func (w *GeminiWorker) Execute(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
	w.available = false
	defer func() { w.available = true }()

	startTime := time.Now()

	// Build the prompt
	prompt := buildTaskPrompt(task)

	// Call Gemini API
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

	output := ""
	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		output = response.Candidates[0].Content.Parts[0].Text
	}

	tokensUsed := 0
	if response.UsageMetadata.TotalTokenCount > 0 {
		tokensUsed = response.UsageMetadata.TotalTokenCount
	} else {
		// Fallback estimate
		tokensUsed = estimateTokens(len(prompt) + len(output))
	}

	return &types.ExecutionResult{
		TaskID:     task.ID,
		Backend:    w.backend,
		Success:    true,
		Output:     output,
		TokensUsed: tokensUsed,
		CostUSD:    w.estimateCost(tokensUsed),
		DurationMs: duration.Milliseconds(),
	}, nil
}

// CheckQuota verifies if the worker has sufficient quota
func (w *GeminiWorker) CheckQuota(ctx context.Context) error {
	// Try a minimal generation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := w.generate(ctx, "hi")
	if err != nil {
		// If we get an error, check if it looks like a quota error
		// The generate method wraps errors, so we look at the string
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "429") ||
			strings.Contains(errStr, "quota") ||
			strings.Contains(errStr, "resource exhausted") {
			return fmt.Errorf("quota exceeded: %w", err)
		}
		return fmt.Errorf("quota check failed: %w", err)
	}

	return nil
}

// Available returns whether the worker is available
func (w *GeminiWorker) Available() bool {
	return w.available
}

// Backend returns the worker's backend type
func (w *GeminiWorker) Backend() types.Backend {
	return w.backend
}

// CheckHealth verifies the Gemini API is reachable (by listing models or a simple ping)
// Since there's no direct "ping", we can assume it's healthy if we have an API key.
// A better check would be a minimal generation call or list models.
func (w *GeminiWorker) CheckHealth(ctx context.Context) error {
	// Simple check: if API key is missing, it's definitely not healthy
	if w.apiKey == "" {
		return fmt.Errorf("missing Gemini API key")
	}
	return nil
}

// geminiRequest represents a request to the Gemini API
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse represents a response from the Gemini API
type geminiResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func (w *GeminiWorker) generate(ctx context.Context, prompt string) (*geminiResponse, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", w.model, w.apiKey)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Gemini returned status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("Gemini returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &geminiResp, nil
}

func (w *GeminiWorker) estimateCost(tokens int) float64 {
	// Pricing (approximate, e.g., for Gemini 1.5 Flash/Pro)
	// Flash is very cheap, Pro is moderate.
	// This is a rough estimation.

	// Flash: ~$0.35 / 1M tokens (input), ~$1.05 / 1M tokens (output)
	// Pro: ~$3.50 / 1M tokens (input), ~$10.50 / 1M tokens (output)

	// Simplify to an average per token
	ratePer1M := 0.5 // Default to Flash-ish
	if w.backend == types.BackendGeminiPro {
		ratePer1M = 7.0
	}

	return float64(tokens) * ratePer1M / 1_000_000
}
