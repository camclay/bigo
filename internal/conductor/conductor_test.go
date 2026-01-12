package conductor

import (
	"context"
	"os"
	"testing"

	"github.com/cammy/bigo/internal/config"
	"github.com/cammy/bigo/internal/ledger"
	"github.com/cammy/bigo/pkg/types"
)

// MockWorker implements Worker interface
type MockWorker struct {
	BackendType    types.Backend
	ExecuteFunc    func(ctx context.Context, task *types.Task) (*types.ExecutionResult, error)
	AvailableFunc  func() bool
	CheckQuotaFunc func(ctx context.Context) error
}

func (m *MockWorker) Execute(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, task)
	}
	return &types.ExecutionResult{Success: true}, nil
}
func (m *MockWorker) Available() bool {
	if m.AvailableFunc != nil {
		return m.AvailableFunc()
	}
	return true
}
func (m *MockWorker) Backend() types.Backend {
	return m.BackendType
}
func (m *MockWorker) CheckQuota(ctx context.Context) error {
	if m.CheckQuotaFunc != nil {
		return m.CheckQuotaFunc(ctx)
	}
	return nil
}

func TestConductor_Run(t *testing.T) {
	// Setup Ledger (using real sqlite on temp file)
	tmpfile, err := os.CreateTemp("", "conductor-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	l, err := ledger.Init(tmpfile.Name())
	if err != nil {
		t.Fatalf("Ledger init failed: %v", err)
	}
	defer l.Close()

	cfg := &config.Config{} // Empty config is fine for now

	conductor := NewConductor(cfg, l)

	// Register mock workers
	// Trivial -> OllamaFast
	// Simple -> Ollama
	// Standard -> ClaudeSonnet

	mockOllama := &MockWorker{
		BackendType: types.BackendOllama,
		ExecuteFunc: func(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
			return &types.ExecutionResult{
				TaskID:     task.ID,
				Backend:    types.BackendOllama,
				Success:    true,
				Output:     "Ollama Output",
				TokensUsed: 10,
			}, nil
		},
	}
	conductor.RegisterWorker(mockOllama)

	mockOllamaFast := &MockWorker{
		BackendType: types.BackendOllamaFast,
		ExecuteFunc: func(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
			return &types.ExecutionResult{
				TaskID:     task.ID,
				Backend:    types.BackendOllamaFast,
				Success:    true,
				Output:     "Ollama Fast Output",
				TokensUsed: 5,
			}, nil
		},
	}
	conductor.RegisterWorker(mockOllamaFast)

	// Test Case 1: Simple Task (Expect Ollama)
	// "add simple function" -> TierSimple -> BackendOllama
	t.Run("Simple Task", func(t *testing.T) {
		ctx := context.Background()
		res, err := conductor.Run(ctx, "Add simple function", "This checks something simple")
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if res.ActualBackend != types.BackendOllama {
			t.Errorf("Expected backend %s, got %s", types.BackendOllama, res.ActualBackend)
		}
		if res.Execution.Output != "Ollama Output" {
			t.Errorf("Expected output 'Ollama Output', got '%s'", res.Execution.Output)
		}
	})

	// Test Case 2: Trivial Task (Expect OllamaFast)
	// "Fix typo" -> TierTrivial -> BackendOllamaFast
	t.Run("Trivial Task", func(t *testing.T) {
		ctx := context.Background()
		res, err := conductor.Run(ctx, "Fix typo", "spelling mistake")
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if res.ActualBackend != types.BackendOllamaFast {
			t.Errorf("Expected backend %s, got %s", types.BackendOllamaFast, res.ActualBackend)
		}
		if res.Execution.Output != "Ollama Fast Output" {
			t.Errorf("Expected output 'Ollama Fast Output', got '%s'", res.Execution.Output)
		}
	})

	// Test Case 3: Fallback
	// "Standard Feature" -> TierStandard -> ClaudeSonnet (Missing) -> Fallback
	// Fallback for Standard: Sonnet -> OllamaReason -> Haiku.
	// Wait, conductor.go says:
	/*
	   case types.TierStandard:
	       // For standard, try Sonnet then Ollama reasoning
	       fallbacks = []types.Backend{
	           types.BackendClaudeSonnet,
	           types.BackendOllamaReason,
	           types.BackendClaudeHaiku,
	       }
	*/
	// If none of these are registered, it returns error "no available worker for this task tier".
	// My mockOllama is BackendOllama (default), not OllamaReason.

	// Let's register OllamaReason to test fallback.
	mockOllamaReason := &MockWorker{
		BackendType: types.BackendOllamaReason,
		ExecuteFunc: func(ctx context.Context, task *types.Task) (*types.ExecutionResult, error) {
			return &types.ExecutionResult{
				TaskID:  task.ID,
				Backend: types.BackendOllamaReason,
				Success: true,
				Output:  "Ollama Reason Output",
			}, nil
		},
	}
	conductor.RegisterWorker(mockOllamaReason)

	t.Run("Fallback", func(t *testing.T) {
		ctx := context.Background()
		// "Implement new feature" -> TierStandard -> ClaudeSonnet (Primary)
		res, err := conductor.Run(ctx, "Implement new feature", "Standard logic")
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		// Should fall back to OllamaReason since Sonnet is not registered
		if res.ActualBackend != types.BackendOllamaReason {
			t.Errorf("Expected fallback to %s, got %s", types.BackendOllamaReason, res.ActualBackend)
		}
	})
}
