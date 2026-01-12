package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bigo",
	Short: "BigO - Unified Claude + Ollama Agent Orchestrator",
	Long: `BigO intelligently tiers work across Claude models (haiku/sonnet/opus)
and local Ollama models, optimizing for cost and enabling parallel execution.

It classifies tasks by complexity and routes them to the most cost-effective
backend while maintaining quality through blind validation.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
}
