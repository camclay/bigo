package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cammy/bigo/internal/conductor"
	"github.com/cammy/bigo/internal/config"
	"github.com/cammy/bigo/internal/ledger"
	"github.com/cammy/bigo/internal/workers"
	"github.com/cammy/bigo/pkg/types"
	"github.com/spf13/cobra"
)

var (
	runTier   string
	runDryRun bool
)

var runCmd = &cobra.Command{
	Use:   "run [task description]",
	Short: "Execute a task through the orchestrator",
	Long: `Classifies the task, routes it to the appropriate backend
(Ollama for simple tasks, Claude for complex ones), and executes it.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTask,
}

func init() {
	runCmd.Flags().StringVarP(&runTier, "tier", "t", "", "Force a specific tier (trivial, simple, standard, complex, critical)")
	runCmd.Flags().BoolVarP(&runDryRun, "dry-run", "n", false, "Classify and show routing without executing")
}

func runTask(cmd *cobra.Command, args []string) error {
	task := strings.Join(args, " ")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load config and ledger
	configPath := filepath.Join(cwd, ".bigo", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		// Use default config if not initialized
		cfg = config.Default()
	}

	ledgerPath := filepath.Join(cwd, ".bigo", "ledger.db")
	var l *ledger.Ledger
	if _, err = os.Stat(ledgerPath); err == nil {
		l, err = ledger.Open(ledgerPath)
		if err != nil {
			return fmt.Errorf("failed to open ledger: %w", err)
		}
		defer l.Close()
	}

	// Check quotas before initializing
	ctx := cmd.Context()

	// Check Claude Quota
	if cfg.Workers.Claude.Enabled {
		// Use first available model for check
		var model string
		for _, m := range cfg.Workers.Claude.Models {
			model = m
			break
		}

		if model != "" {
			w := workers.NewClaudeWorker("quota-check", workers.ClaudeConfig{
				Model:   model,
				Backend: types.BackendClaudeSonnet, // Dummy backend for check
			})
							fmt.Printf("Checking Claude quota (%s)...\n", model)
							if err = w.CheckQuota(ctx); err != nil {
								fmt.Printf("⚠ Claude quota check failed: %v\n  Disabling Claude backend.\n", err)
			
				cfg.Workers.Claude.Enabled = false
			}
		}
	}

	// Check Gemini Quota
	if cfg.Workers.Gemini.Enabled && cfg.Workers.Gemini.APIKey != "" {
		// Use first available model for check
		var model string
		for _, m := range cfg.Workers.Gemini.Models {
			model = m
			break
		}

		if model != "" {
			w := workers.NewGeminiWorker("quota-check", workers.GeminiConfig{
				APIKey:  cfg.Workers.Gemini.APIKey,
				Model:   model,
				Backend: types.BackendGeminiFlash, // Dummy backend for check
			})
							fmt.Printf("Checking Gemini quota (%s)...\n", model)
							if err = w.CheckQuota(ctx); err != nil {
								fmt.Printf("⚠ Gemini quota check failed: %v\n  Disabling Gemini backend.\n", err)
			
				cfg.Workers.Gemini.Enabled = false
			}
		}
	}

	// Create conductor
	cond := conductor.NewConductor(cfg, l)

	fmt.Println("BigO Task Execution")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Task: %s\n", task)
	fmt.Println("───────────────────────────────────────")

	if runDryRun {
		result := cond.DryRun(task, "")

		fmt.Printf("Tier:       %s (T%d)\n", result.Classification.Tier.String(), result.Classification.Tier)
		fmt.Printf("Confidence: %.0f%%\n", result.Classification.Confidence*100)
		fmt.Printf("Backend:    %s\n", result.ActualBackend)

		if !result.WorkerAvailable {
			fmt.Println("⚠ Primary worker not available")
			if result.FallbackBackend != "" {
				fmt.Printf("  Fallback: %s\n", result.FallbackBackend)
			} else {
				fmt.Println("  No fallback available!")
			}
		}

		if result.ValidationRequired {
			tierCfg := result.Classification.Tier
			fmt.Printf("Validation: %d validator(s) required\n", tierCfg+1)
		} else {
			fmt.Println("Validation: none (trivial tier)")
		}

		fmt.Println("───────────────────────────────────────")
		fmt.Println("[DRY RUN] No execution performed")
		return nil
	}

	// Register Ollama workers
	if cfg.Workers.Ollama.Enabled {
		for name, model := range cfg.Workers.Ollama.Models {
			var backend types.Backend
			switch name {
			case "fast":
				backend = types.BackendOllamaFast
			case "reasoning":
				backend = types.BackendOllamaReason
			default:
				backend = types.BackendOllama
			}

			worker := workers.NewOllamaWorker(name, workers.OllamaConfig{
				Endpoint: cfg.Workers.Ollama.Endpoint,
				Model:    model,
				Backend:  backend,
			})
			cond.RegisterWorker(worker)
		}
	}

	// Register Claude workers
	if cfg.Workers.Claude.Enabled {
		for name, model := range cfg.Workers.Claude.Models {
			var backend types.Backend
			switch name {
			case "opus":
				backend = types.BackendClaudeOpus
			case "haiku":
				backend = types.BackendClaudeHaiku
			default:
				backend = types.BackendClaudeSonnet
			}

			worker := workers.NewClaudeWorker(name, workers.ClaudeConfig{
				Model:   model,
				Backend: backend,
			})
			cond.RegisterWorker(worker)
		}
	}

	// Register Gemini workers
	if cfg.Workers.Gemini.Enabled && cfg.Workers.Gemini.APIKey != "" {
		for name, model := range cfg.Workers.Gemini.Models {
			var backend types.Backend
			switch name {
			case "pro":
				backend = types.BackendGeminiPro
			default:
				backend = types.BackendGeminiFlash
			}

			worker := workers.NewGeminiWorker(name, workers.GeminiConfig{
				APIKey:  cfg.Workers.Gemini.APIKey,
				Model:   model,
				Backend: backend,
			})
			cond.RegisterWorker(worker)
		}
	}

	// Execute the task
	fmt.Println("Executing...")
	fmt.Println()

	result, err := cond.Run(ctx, task, "")
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Display results
	fmt.Printf("Status:   %s\n", result.Status)
	fmt.Printf("Backend:  %s\n", result.ActualBackend)
	fmt.Printf("Duration: %s\n", result.Duration.Round(time.Millisecond))

	if result.Execution != nil {
		fmt.Printf("Tokens:   %d\n", result.Execution.TokensUsed)
		fmt.Printf("Cost:     $%.4f\n", result.Execution.CostUSD)
		fmt.Println("───────────────────────────────────────")
		fmt.Println("Output:")
		fmt.Println(result.Execution.Output)
	}

	if result.Error != "" {
		fmt.Println("───────────────────────────────────────")
		fmt.Printf("Error: %s\n", result.Error)
	}

	return nil
}
