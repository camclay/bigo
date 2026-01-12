package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cammy/bigo/internal/ledger"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show task status and statistics",
	Long:  `Displays current task queue, execution history, and cost savings.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	ledgerPath := filepath.Join(cwd, ".bigo", "ledger.db")
	if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
		return fmt.Errorf("BigO not initialized. Run 'bigo init' first")
	}

	db, err := ledger.Open(ledgerPath)
	if err != nil {
		return fmt.Errorf("failed to open ledger: %w", err)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Println("BigO Status")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Tasks:      %d total (%d pending, %d completed)\n",
		stats.TotalTasks, stats.PendingTasks, stats.CompletedTasks)
	fmt.Printf("Executions: %d total\n", stats.TotalExecutions)
	fmt.Println("───────────────────────────────────────")
	fmt.Println("Cost Breakdown:")
	fmt.Printf("  Claude:   $%.4f (%d tasks)\n", stats.ClaudeCost, stats.ClaudeTasks)
	fmt.Printf("  Ollama:   $%.4f (%d tasks)\n", stats.OllamaCost, stats.OllamaTasks)
	fmt.Printf("  Savings:  $%.4f (%.1f%%)\n", stats.EstimatedSavings, stats.SavingsPercent)
	fmt.Println("═══════════════════════════════════════")

	return nil
}
