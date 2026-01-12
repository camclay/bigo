package workers

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cammy/bigo/pkg/types"
)

// TestCheckQuota logic by mocking the exec.Command behavior
// Since we can't easily mock exec.Command globally without refactoring,
// we will use a small trick: point cliPath to a script that exits with specific output.

func TestClaudeWorker_CheckQuota(t *testing.T) {
	// Create a temporary mock script
	scriptPath := "./mock_claude.sh"
	
tests := []struct {
		name          string
		scriptContent string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Quota OK",
			scriptContent: "#!/bin/bash\nexit 0",
			expectError:   false,
		},
		{
			name:          "Quota Exceeded",
			scriptContent: "#!/bin/bash\necho 'Error: quota exceeded' >&2\nexit 1",
			expectError:   true,
			errorContains: "quota exceeded",
		},
		{
			name:          "Insufficient Credits",
			scriptContent: "#!/bin/bash\necho 'Insufficient credits' >&2\nexit 1",
			expectError:   true,
			errorContains: "quota exceeded or payment required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.WriteFile(scriptPath, []byte(tt.scriptContent), 0755)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(scriptPath)

			worker := NewClaudeWorker("test", ClaudeConfig{
				CLIPath: scriptPath,
				Model:   "sonnet",
				Backend: types.BackendClaudeSonnet,
			})

			err = worker.CheckQuota(context.Background())
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
