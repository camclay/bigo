package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cammy/bigo/pkg/types"
)

// mockTransport implements http.RoundTripper
type mockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestGeminiWorker_Execute(t *testing.T) {
	// Setup mock response
	mockResp := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: "This is a mocked response."},
					},
				},
			},
		},
		UsageMetadata: geminiUsageMetadata{
			TotalTokenCount: 123,
		},
	}
	respBody, _ := json.Marshal(mockResp)

	// Setup worker
	cfg := GeminiConfig{
		APIKey:  "test-key",
		Model:   "gemini-pro",
		Backend: types.BackendGeminiPro,
		Timeout: 1 * time.Second,
	}
	worker := NewGeminiWorker("worker-1", cfg)

	// Inject mock transport
	worker.client.Transport = &mockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Verify URL contains key and model
			if !strings.Contains(req.URL.String(), "gemini-pro") {
				t.Errorf("URL does not contain model: %s", req.URL.String())
			}
			if !strings.Contains(req.URL.String(), "key=test-key") {
				t.Errorf("URL does not contain API key: %s", req.URL.String())
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(respBody)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// Create task
	task := &types.Task{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Do something",
		Tier:        types.TierSimple,
	}

	// Execute
	ctx := context.Background()
	result, err := worker.Execute(ctx, task)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("Expected success")
	}
	if result.Output != "This is a mocked response." {
		t.Errorf("Expected output 'This is a mocked response.', got '%s'", result.Output)
	}
	if result.TokensUsed != 123 {
		t.Errorf("Expected 123 tokens, got %d", result.TokensUsed)
	}
	if result.Backend != types.BackendGeminiPro {
		t.Errorf("Expected backend %s, got %s", types.BackendGeminiPro, result.Backend)
	}
}

func TestGeminiWorker_CheckQuota(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		respBody      string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Quota OK",
			statusCode:  http.StatusOK,
			respBody:    `{}`,
			expectError: false,
		},
		{
			name:          "Quota Exceeded 429",
			statusCode:    http.StatusTooManyRequests, // 429
			respBody:      `{"error": {"message": "quota exceeded"}}`,
			expectError:   true,
			errorContains: "quota exceeded",
		},
		{
			name:          "Other Error",
			statusCode:    http.StatusInternalServerError,
			respBody:      `{"error": {"message": "internal error"}}`,
			expectError:   true,
			errorContains: "quota check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GeminiConfig{APIKey: "key", Model: "model"}
			worker := NewGeminiWorker("w", cfg)

			worker.client.Transport = &mockTransport{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.respBody)),
						Header:     make(http.Header),
					}, nil
				},
			}

			err := worker.CheckQuota(context.Background())
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
