package conductor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cammy/bigo/internal/config"
	"github.com/cammy/bigo/internal/ledger"
	"github.com/cammy/bigo/pkg/types"
)

// Conductor orchestrates task classification, execution, and validation
type Conductor struct {
	config     *config.Config
	ledger     *ledger.Ledger
	classifier *Classifier
	workers    map[types.Backend]Worker
}

// Worker interface for different backends
type Worker interface {
	Execute(ctx context.Context, task *types.Task) (*types.ExecutionResult, error)
	Available() bool
	Backend() types.Backend
	CheckQuota(ctx context.Context) error
}

// NewConductor creates a new conductor instance
func NewConductor(cfg *config.Config, l *ledger.Ledger) *Conductor {
	return &Conductor{
		config:     cfg,
		ledger:     l,
		classifier: NewClassifier(),
		workers:    make(map[types.Backend]Worker),
	}
}

// RegisterWorker adds a worker backend to the conductor
func (c *Conductor) RegisterWorker(w Worker) {
	c.workers[w.Backend()] = w
}

// Run executes a task through the full pipeline
func (c *Conductor) Run(ctx context.Context, title, description string) (*RunResult, error) {
	// Step 1: Classify
	classification := c.classifier.Classify(title, description)

	// Step 2: Create task in ledger
	task := &ledger.Task{
		ID:            generateID(),
		Title:         title,
		Description:   description,
		Tier:          int(classification.Tier),
		Status:        string(types.StatusPending),
		WorkerBackend: string(classification.RecommendedBackend),
	}

	if err := c.ledger.CreateTask(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	result := &RunResult{
		TaskID:         task.ID,
		Classification: classification,
		StartTime:      time.Now(),
	}

	// Step 3: Find available worker
	worker, ok := c.workers[classification.RecommendedBackend]
	if !ok || !worker.Available() {
		// Try fallback backends
		worker = c.findFallbackWorker(classification.Tier)
		if worker == nil {
			result.Error = "no available worker for this task tier"
			result.Status = types.StatusFailed
			return result, nil
		}
		result.ActualBackend = worker.Backend()
	} else {
		result.ActualBackend = classification.RecommendedBackend
	}

	// Step 4: Update status and execute
	if err := c.ledger.UpdateTaskStatus(task.ID, string(types.StatusWorking)); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	execResult, err := worker.Execute(ctx, &types.Task{
		ID:          task.ID,
		Title:       title,
		Description: description,
		Tier:        classification.Tier,
		Backend:     result.ActualBackend,
	})

	if err != nil {
		result.Error = err.Error()
		result.Status = types.StatusFailed
		c.ledger.UpdateTaskStatus(task.ID, string(types.StatusFailed))
		return result, nil
	}

	result.Execution = execResult

	// Check if execution reported failure
	if !execResult.Success {
		result.Error = execResult.Error
		result.Status = types.StatusFailed
		c.ledger.UpdateTaskStatus(task.ID, string(types.StatusFailed))
		return result, nil
	}
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Step 5: Record execution
	exec := &ledger.Execution{
		ID:         generateID(),
		TaskID:     task.ID,
		Backend:    string(result.ActualBackend),
		Output:     execResult.Output,
		TokensUsed: execResult.TokensUsed,
		CostUSD:    execResult.CostUSD,
		DurationMs: int(result.Duration.Milliseconds()),
		Status:     "completed",
	}

	if !execResult.Success {
		exec.Status = "failed"
		exec.ErrorMsg = execResult.Error
	}

	if err := c.ledger.CreateExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to record execution: %w", err)
	}

	// Step 6: Validation (if required for this tier)
	tierConfig := types.DefaultTierConfigs()[classification.Tier]
	if tierConfig.ValidatorCount > 0 {
		result.ValidationRequired = true
		// TODO: Implement validation pipeline
		result.ValidationPending = true
	}

	// Update final status
	if execResult.Success {
		if result.ValidationRequired && result.ValidationPending {
			result.Status = types.StatusValidating
			c.ledger.UpdateTaskStatus(task.ID, string(types.StatusValidating))
		} else {
			result.Status = types.StatusDone
			c.ledger.UpdateTaskStatus(task.ID, string(types.StatusDone))
		}
	} else {
		result.Status = types.StatusFailed
		c.ledger.UpdateTaskStatus(task.ID, string(types.StatusFailed))
	}

	return result, nil
}

// DryRun classifies a task without executing it
func (c *Conductor) DryRun(title, description string) *RunResult {
	classification := c.classifier.Classify(title, description)

	// Check worker availability
	worker, ok := c.workers[classification.RecommendedBackend]
	workerAvailable := ok && worker.Available()

	var fallbackBackend types.Backend
	if !workerAvailable {
		if fb := c.findFallbackWorker(classification.Tier); fb != nil {
			fallbackBackend = fb.Backend()
		}
	}

	return &RunResult{
		Classification:    classification,
		ActualBackend:     classification.RecommendedBackend,
		FallbackBackend:   fallbackBackend,
		WorkerAvailable:   workerAvailable,
		ValidationRequired: types.DefaultTierConfigs()[classification.Tier].ValidatorCount > 0,
		DryRun:            true,
	}
}

func (c *Conductor) findFallbackWorker(tier types.Tier) Worker {
	// Fallback priority based on tier
	var fallbacks []types.Backend

	switch tier {
	case types.TierTrivial, types.TierSimple:
		// For simple tasks, try any Ollama, then Haiku
		fallbacks = []types.Backend{
			types.BackendOllama,
			types.BackendOllamaFast,
			types.BackendClaudeHaiku,
		}
	case types.TierStandard:
		// For standard, try Sonnet then Ollama reasoning
		fallbacks = []types.Backend{
			types.BackendClaudeSonnet,
			types.BackendOllamaReason,
			types.BackendClaudeHaiku,
		}
	case types.TierComplex, types.TierCritical:
		// For complex/critical, only Claude
		fallbacks = []types.Backend{
			types.BackendClaudeOpus,
			types.BackendClaudeSonnet,
		}
	}

	for _, backend := range fallbacks {
		if w, ok := c.workers[backend]; ok && w.Available() {
			return w
		}
	}

	return nil
}

// RunResult contains the outcome of a task execution
type RunResult struct {
	TaskID             string
	Classification     *types.ClassificationResult
	ActualBackend      types.Backend
	FallbackBackend    types.Backend
	WorkerAvailable    bool
	Execution          *types.ExecutionResult
	Status             types.TaskStatus
	Error              string
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
	ValidationRequired bool
	ValidationPending  bool
	ValidationResults  []*types.ValidationResult
	DryRun             bool
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
