package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all BigO configuration
type Config struct {
	Conductor  ConductorConfig  `yaml:"conductor"`
	Workers    WorkersConfig    `yaml:"workers"`
	Validators ValidatorsConfig `yaml:"validators"`
	Ledger     LedgerConfig     `yaml:"ledger"`
	Bus        BusConfig        `yaml:"bus"`
}

// ConductorConfig configures the main orchestrator
type ConductorConfig struct {
	ClassifierModel   string `yaml:"classifier_model"`
	MaxRetries        int    `yaml:"max_retries"`
	ValidationTimeout string `yaml:"validation_timeout"`
}

// WorkersConfig configures all worker backends
type WorkersConfig struct {
	Claude ClaudeConfig `yaml:"claude"`
	Ollama OllamaConfig `yaml:"ollama"`
	Gemini GeminiConfig `yaml:"gemini"`
}

// ClaudeConfig configures the Claude backend
type ClaudeConfig struct {
	Enabled       bool              `yaml:"enabled"`
	MaxConcurrent int               `yaml:"max_concurrent"`
	Models        map[string]string `yaml:"models"`
	CostLimits    CostLimits        `yaml:"cost_limits"`
}

// CostLimits sets spending limits for Claude
type CostLimits struct {
	DailyUSD   float64 `yaml:"daily_usd"`
	PerTaskUSD float64 `yaml:"per_task_usd"`
}

// OllamaConfig configures the Ollama backend
type OllamaConfig struct {
	Enabled       bool              `yaml:"enabled"`
	Endpoint      string            `yaml:"endpoint"`
	MaxConcurrent int               `yaml:"max_concurrent"`
	Models        map[string]string `yaml:"models"`
	OpenCodePath  string            `yaml:"opencode_path"`
}

// GeminiConfig configures the Gemini backend
type GeminiConfig struct {
	Enabled       bool              `yaml:"enabled"`
	APIKey        string            `yaml:"api_key"`
	MaxConcurrent int               `yaml:"max_concurrent"`
	Models        map[string]string `yaml:"models"`
}

// ValidatorsConfig configures the validation system
type ValidatorsConfig struct {
	PoolSize int      `yaml:"pool_size"`
	Timeout  string   `yaml:"timeout"`
	Backends []string `yaml:"backends"`
}

// LedgerConfig configures the SQLite ledger
type LedgerConfig struct {
	Path string `yaml:"path"`
}

// BusConfig configures the message bus
type BusConfig struct {
	BufferSize int `yaml:"buffer_size"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Conductor: ConductorConfig{
			ClassifierModel:   "claude:sonnet",
			MaxRetries:        3,
			ValidationTimeout: "300s",
		},
		Workers: WorkersConfig{
			Claude: ClaudeConfig{
				Enabled:       true,
				MaxConcurrent: 2,
				Models: map[string]string{
					"opus":   "claude-opus-4-5-20251101",
					"sonnet": "claude-sonnet-4-20250514",
					"haiku":  "claude-haiku-3-5-20241022",
				},
				CostLimits: CostLimits{
					DailyUSD:   50.0,
					PerTaskUSD: 5.0,
				},
			},
			Ollama: OllamaConfig{
				Enabled:       true,
				Endpoint:      "http://cammy-custom2020:11434",
				MaxConcurrent: 4,
				Models: map[string]string{
					"default":   "qwen3:8b",
					"fast":      "phi3:mini-16k",
					"reasoning": "qwen3:8b-8k",
				},
				OpenCodePath: "opencode",
			},
			Gemini: GeminiConfig{
				Enabled:       true,
				APIKey:        "", // User must provide
				MaxConcurrent: 4,
				Models: map[string]string{
					"flash": "gemini-1.5-flash",
					"pro":   "gemini-1.5-pro",
				},
			},
		},
		Validators: ValidatorsConfig{
			PoolSize: 5,
			Timeout:  "120s",
			Backends: []string{
				"claude:sonnet",
				"ollama:qwen3:8b",
			},
		},
		Ledger: LedgerConfig{
			Path: ".bigo/ledger.db",
		},
		Bus: BusConfig{
			BufferSize: 1000,
		},
	}
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// WriteDefault writes the default configuration to a file
func WriteDefault(path string) error {
	cfg := Default()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := []byte(`# BigO Configuration
# Unified Claude + Ollama Agent Orchestrator
#
# Tier routing:
#   T0 (TRIVIAL)  → Ollama (fast model)
#   T1 (SIMPLE)   → Ollama (default) + 1 validator
#   T2 (STANDARD) → Claude Sonnet + 2 validators
#   T3 (COMPLEX)  → Claude Sonnet + Opus planning + 3 validators
#   T4 (CRITICAL) → Claude Opus + 5 validators

`)

	return os.WriteFile(path, append(header, data...), 0600)
}
