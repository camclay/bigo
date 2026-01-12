package types

import "time"

// Tier represents task complexity levels
type Tier int

const (
	TierTrivial  Tier = 0 // Simple edits, formatting, boilerplate
	TierSimple   Tier = 1 // Straightforward changes, clear patterns
	TierStandard Tier = 2 // Feature work, refactoring, most tasks
	TierComplex  Tier = 3 // Architecture, multi-file changes
	TierCritical Tier = 4 // Security, core logic, breaking changes
)

func (t Tier) String() string {
	switch t {
	case TierTrivial:
		return "TRIVIAL"
	case TierSimple:
		return "SIMPLE"
	case TierStandard:
		return "STANDARD"
	case TierComplex:
		return "COMPLEX"
	case TierCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Backend represents an execution backend
type Backend string

const (
	BackendClaudeOpus   Backend = "claude:opus"
	BackendClaudeSonnet Backend = "claude:sonnet"
	BackendClaudeHaiku  Backend = "claude:haiku"
	BackendOllamaFast   Backend = "ollama:fast"
	BackendOllama       Backend = "ollama:default"
	BackendOllamaReason Backend = "ollama:reasoning"
)

// TaskStatus represents the lifecycle state of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusAssigned   TaskStatus = "assigned"
	StatusWorking    TaskStatus = "working"
	StatusValidating TaskStatus = "validating"
	StatusApproved   TaskStatus = "approved"
	StatusRejected   TaskStatus = "rejected"
	StatusDone       TaskStatus = "done"
	StatusFailed     TaskStatus = "failed"
)

// Task represents a unit of work to be executed
type Task struct {
	ID          string
	ParentID    string
	Title       string
	Description string
	Tier        Tier
	Status      TaskStatus
	Backend     Backend
	ContextPath string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ClassificationResult holds the output of the task classifier
type ClassificationResult struct {
	Tier            Tier
	Confidence      float64
	RecommendedBackend Backend
	Reasoning       string
	Patterns        []string
	EstimatedLines  int
	EstimatedFiles  int
}

// ExecutionResult holds the output of a task execution
type ExecutionResult struct {
	TaskID     string
	Backend    Backend
	Success    bool
	Output     string
	Diff       string
	TokensUsed int
	CostUSD    float64
	DurationMs int64
	Error      string
}

// ValidationResult holds the output of a validation
type ValidationResult struct {
	ExecutionID string
	ValidatorID string
	Backend     Backend
	Approved    bool
	Findings    []Finding
}

// Finding represents an issue found during validation
type Finding struct {
	Severity    string // error, warning, info
	Location    string // file:line or general
	Message     string
	Suggestion  string
}

// Message represents a message on the internal bus
type Message struct {
	Type      string
	TaskID    string
	Payload   map[string]interface{}
	Timestamp time.Time
}

// TierConfig maps tiers to their execution configuration
type TierConfig struct {
	PrimaryBackend   Backend
	ValidatorBackend Backend
	ValidatorCount   int
	RequiredApprovals int
}

// DefaultTierConfigs returns the default tier routing configuration
func DefaultTierConfigs() map[Tier]TierConfig {
	return map[Tier]TierConfig{
		TierTrivial: {
			PrimaryBackend:   BackendOllamaFast,
			ValidatorBackend: "",
			ValidatorCount:   0,
			RequiredApprovals: 0,
		},
		TierSimple: {
			PrimaryBackend:   BackendOllama,
			ValidatorBackend: BackendOllama,
			ValidatorCount:   1,
			RequiredApprovals: 1,
		},
		TierStandard: {
			PrimaryBackend:   BackendClaudeSonnet,
			ValidatorBackend: BackendClaudeSonnet,
			ValidatorCount:   2,
			RequiredApprovals: 2,
		},
		TierComplex: {
			PrimaryBackend:   BackendClaudeSonnet,
			ValidatorBackend: BackendClaudeSonnet,
			ValidatorCount:   3,
			RequiredApprovals: 2,
		},
		TierCritical: {
			PrimaryBackend:   BackendClaudeOpus,
			ValidatorBackend: BackendClaudeSonnet,
			ValidatorCount:   5,
			RequiredApprovals: 4,
		},
	}
}
