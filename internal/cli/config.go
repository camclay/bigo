package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cammy/bigo/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or edit BigO configuration",
	Long:  `Shows the current configuration. Use subcommands to modify settings.`,
	RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(cwd, ".bigo", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Pretty print the config
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Printf("Configuration (%s):\n", configPath)
	fmt.Println("─────────────────────────────────────")
	fmt.Println(string(out))

	return nil
}
