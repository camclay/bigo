package cli

import (
	"fmt"
	"strings"

	"github.com/cammy/bigo/internal/conductor"
	"github.com/spf13/cobra"
)

var classifyCmd = &cobra.Command{
	Use:   "classify [task description]",
	Short: "Classify a task without executing it",
	Long: `Analyzes the task description and shows the complexity tier,
recommended backend, and reasoning behind the classification.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runClassify,
}

func init() {
	rootCmd.AddCommand(classifyCmd)
}

func runClassify(cmd *cobra.Command, args []string) error {
	task := strings.Join(args, " ")

	classifier := conductor.NewClassifier()
	result := classifier.Classify(task, "")

	fmt.Println("Task Classification")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Task: %s\n", task)
	fmt.Println("───────────────────────────────────────")
	fmt.Printf("Tier:       %s (T%d)\n", result.Tier.String(), result.Tier)
	fmt.Printf("Confidence: %.0f%%\n", result.Confidence*100)
	fmt.Printf("Backend:    %s\n", result.RecommendedBackend)
	fmt.Println("───────────────────────────────────────")
	fmt.Printf("Reasoning:  %s\n", result.Reasoning)

	if len(result.Patterns) > 0 {
		fmt.Printf("Patterns:   %s\n", strings.Join(result.Patterns, ", "))
	}

	fmt.Println("───────────────────────────────────────")

	// Show tier routing info
	fmt.Println("\nRouting for this tier:")
	switch result.Tier {
	case 0:
		fmt.Println("  → Ollama (fast) - no validation")
		fmt.Println("  → Cost: $0.00")
	case 1:
		fmt.Println("  → Ollama (default) + 1 validator")
		fmt.Println("  → Cost: $0.00")
	case 2:
		fmt.Println("  → Claude Sonnet + 2 validators")
		fmt.Println("  → Est. cost: ~$0.02-0.10")
	case 3:
		fmt.Println("  → Claude Sonnet + Opus planning + 3 validators")
		fmt.Println("  → Est. cost: ~$0.10-0.50")
	case 4:
		fmt.Println("  → Claude Opus + 5 validators")
		fmt.Println("  → Est. cost: ~$0.50-2.00")
	}

	return nil
}
