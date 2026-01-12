package ledger

import (
	"os"
	"testing"
)

func TestLedger_Init(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "ledger-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	l, err := Init(tmpfile.Name())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer l.Close()

	// Verify schema creation by querying metadata
	var val string
	err = l.db.QueryRow("SELECT value FROM metadata WHERE key='schema_version'").Scan(&val)
	if err != nil {
		t.Errorf("Failed to query metadata: %v", err)
	}
	if val != "1" {
		t.Errorf("Expected schema version 1, got %s", val)
	}
}

func TestLedger_Operations(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "ledger-ops-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	l, err := Init(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Create Task
	task := &Task{
		ID:            "task-1",
		Title:         "Test Task",
		Description:   "Description",
		Tier:          2,
		Status:        "pending",
		WorkerBackend: "gemini:pro",
	}
	if createErr := l.CreateTask(task); createErr != nil {
		t.Fatalf("CreateTask failed: %v", createErr)
	}

	// Get Task
	got, err := l.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got.Title != task.Title {
		t.Errorf("Expected title %s, got %s", task.Title, got.Title)
	}

	// Update Status
	if updateErr := l.UpdateTaskStatus("task-1", "done"); updateErr != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", updateErr)
	}
	got, err = l.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got.Status != "done" {
		t.Errorf("Expected status done, got %s", got.Status)
	}

	// Create Execution
	exec := &Execution{
		ID:         "exec-1",
		TaskID:     "task-1",
		Backend:    "gemini:pro",
		TokensUsed: 100,
		CostUSD:    0.01,
	}
	if execErr := l.CreateExecution(exec); execErr != nil {
		t.Fatalf("CreateExecution failed: %v", execErr)
	}

	// Check Stats
	stats, err := l.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", stats.TotalTasks)
	}
	if stats.CompletedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats.CompletedTasks)
	}
	if stats.TotalExecutions != 1 {
		t.Errorf("Expected 1 execution, got %d", stats.TotalExecutions)
	}
	if stats.GeminiTasks != 1 {
		t.Errorf("Expected 1 Gemini task, got %d", stats.GeminiTasks)
	}
	if stats.GeminiCost != 0.01 {
		t.Errorf("Expected Gemini cost 0.01, got %f", stats.GeminiCost)
	}
}
