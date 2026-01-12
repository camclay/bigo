package ledger

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Ledger manages the SQLite database for task state persistence
type Ledger struct {
	db   *sql.DB
	path string
}

// Stats holds aggregated statistics from the ledger
type Stats struct {
	TotalTasks       int
	PendingTasks     int
	CompletedTasks   int
	TotalExecutions  int
	ClaudeTasks      int
	ClaudeCost       float64
	GeminiTasks      int
	GeminiCost       float64
	OllamaTasks      int
	OllamaCost       float64
	EstimatedSavings float64
	SavingsPercent   float64
}

// Init creates a new ledger database with the schema
func Init(path string) (*Ledger, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Ledger{db: db, path: path}, nil
}

// Open opens an existing ledger database
func Open(path string) (*Ledger, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Ledger{db: db, path: path}, nil
}

// Close closes the database connection
func (l *Ledger) Close() error {
	return l.db.Close()
}

// GetStats returns aggregated statistics
func (l *Ledger) GetStats() (*Stats, error) {
	stats := &Stats{}

	// Total tasks
	err := l.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&stats.TotalTasks)
	if err != nil {
		return nil, err
	}

	// Pending tasks
	if err := l.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status IN ('pending', 'assigned', 'working', 'validating')").Scan(&stats.PendingTasks); err != nil {
		return nil, err
	}

	// Completed tasks
	if err := l.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'done'").Scan(&stats.CompletedTasks); err != nil {
		return nil, err
	}

	// Total executions
	if err := l.db.QueryRow("SELECT COUNT(*) FROM executions").Scan(&stats.TotalExecutions); err != nil {
		return nil, err
	}

	// Claude stats
	if err := l.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(cost_usd), 0)
		FROM executions
		WHERE backend LIKE 'claude:%'
	`).Scan(&stats.ClaudeTasks, &stats.ClaudeCost); err != nil {
		return nil, err
	}

	// Gemini stats
	if err := l.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(cost_usd), 0)
		FROM executions
		WHERE backend LIKE 'gemini:%'
	`).Scan(&stats.GeminiTasks, &stats.GeminiCost); err != nil {
		return nil, err
	}

	// Ollama stats
	if err := l.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(cost_usd), 0)
		FROM executions
		WHERE backend LIKE 'ollama:%'
	`).Scan(&stats.OllamaTasks, &stats.OllamaCost); err != nil {
		return nil, err
	}

	// Calculate savings (estimate what Claude would have cost for all tasks)
	// Assuming average Claude cost per task of $0.05 for simple tasks
	nonClaudeTasks := stats.OllamaTasks + stats.GeminiTasks
	stats.EstimatedSavings = float64(nonClaudeTasks)*0.05 - stats.GeminiCost
	if stats.ClaudeCost+stats.GeminiCost+stats.EstimatedSavings > 0 {
		totalEstClaudeCost := stats.ClaudeCost + stats.GeminiCost + stats.EstimatedSavings
		stats.SavingsPercent = (stats.EstimatedSavings / totalEstClaudeCost) * 100
	}

	return stats, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	-- Tasks table
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		parent_id TEXT REFERENCES tasks(id),
		title TEXT NOT NULL,
		description TEXT,
		tier INTEGER DEFAULT 2,
		status TEXT DEFAULT 'pending',
		worker_backend TEXT,
		context_path TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Executions table
	CREATE TABLE IF NOT EXISTS executions (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id),
		worker_id TEXT,
		backend TEXT NOT NULL,
		input_hash TEXT,
		output TEXT,
		tokens_used INTEGER DEFAULT 0,
		cost_usd REAL DEFAULT 0,
		duration_ms INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending',
		error_msg TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Validations table
	CREATE TABLE IF NOT EXISTS validations (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL REFERENCES executions(id),
		validator_id TEXT,
		backend TEXT NOT NULL,
		verdict TEXT,
		findings TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_tier ON tasks(tier);
	CREATE INDEX IF NOT EXISTS idx_executions_task ON executions(task_id);
	CREATE INDEX IF NOT EXISTS idx_executions_backend ON executions(backend);
	CREATE INDEX IF NOT EXISTS idx_validations_execution ON validations(execution_id);

	-- Metadata table for settings
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Insert version
	INSERT OR REPLACE INTO metadata (key, value, updated_at)
	VALUES ('schema_version', '1', CURRENT_TIMESTAMP);
	`

	_, err := db.Exec(schema)
	return err
}

// Task represents a task in the ledger
type Task struct {
	ID            string
	ParentID      *string
	Title         string
	Description   string
	Tier          int
	Status        string
	WorkerBackend string
	ContextPath   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateTask inserts a new task into the ledger
func (l *Ledger) CreateTask(task *Task) error {
	_, err := l.db.Exec(`
		INSERT INTO tasks (id, parent_id, title, description, tier, status, worker_backend, context_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.ParentID, task.Title, task.Description, task.Tier, task.Status, task.WorkerBackend, task.ContextPath)
	return err
}

// UpdateTaskStatus updates the status of a task
func (l *Ledger) UpdateTaskStatus(id, status string) error {
	_, err := l.db.Exec(`
		UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, status, id)
	return err
}

// GetTask retrieves a task by ID
func (l *Ledger) GetTask(id string) (*Task, error) {
	task := &Task{}
	err := l.db.QueryRow(`
		SELECT id, parent_id, title, description, tier, status, worker_backend, context_path, created_at, updated_at
		FROM tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.ParentID, &task.Title, &task.Description, &task.Tier, &task.Status,
		&task.WorkerBackend, &task.ContextPath, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// Execution represents a task execution attempt
type Execution struct {
	ID         string
	TaskID     string
	WorkerID   string
	Backend    string
	InputHash  string
	Output     string
	TokensUsed int
	CostUSD    float64
	DurationMs int
	Status     string
	ErrorMsg   string
	CreatedAt  time.Time
}

// CreateExecution records a new execution attempt
func (l *Ledger) CreateExecution(exec *Execution) error {
	_, err := l.db.Exec(`
		INSERT INTO executions (id, task_id, worker_id, backend, input_hash, output, tokens_used, cost_usd, duration_ms, status, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, exec.ID, exec.TaskID, exec.WorkerID, exec.Backend, exec.InputHash, exec.Output,
		exec.TokensUsed, exec.CostUSD, exec.DurationMs, exec.Status, exec.ErrorMsg)
	return err
}
