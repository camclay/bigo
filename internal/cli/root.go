package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "bigo",
	Short: "BigO - Unified Claude + Ollama Agent Orchestrator",
	Long: `BigO intelligently tiers work across Claude models (haiku/sonnet/opus)
and local Ollama models, optimizing for cost and enabling parallel execution.

It classifies tasks by complexity and routes them to the most cost-effective
backend while maintaining quality through blind validation.`,
	Version: version,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("BigO %s\n", version)
	},
}

// SetVersion sets the version string (called from main)
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}
