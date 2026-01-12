package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cammy/bigo/internal/config"
	"github.com/cammy/bigo/internal/ledger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize BigO in the current directory",
	Long:  `Creates a .bigo directory with the task ledger and default configuration.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	bigoDir := filepath.Join(cwd, ".bigo")

	// Check if already initialized
	if _, err = os.Stat(bigoDir); err == nil {
		return fmt.Errorf("BigO already initialized in this directory")
	}

	// Create .bigo directory
	if err = os.MkdirAll(bigoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .bigo directory: %w", err)
	}

	// Initialize ledger
	ledgerPath := filepath.Join(bigoDir, "ledger.db")
	db, err := ledger.Init(ledgerPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ledger: %w", err)
	}
	defer db.Close()

	// Create default config
	configPath := filepath.Join(bigoDir, "config.yaml")
	if err := config.WriteDefault(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("✓ BigO initialized successfully!")
	fmt.Printf("  Ledger: %s\n", ledgerPath)
	fmt.Printf("  Config: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  bigo run <task>    # Execute a task")
	fmt.Println("  bigo status        # View task status")
	fmt.Println("  bigo config        # View/edit configuration")

	return nil
}
