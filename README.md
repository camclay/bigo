# BigO

**Unified Claude + Gemini + Ollama Agent Orchestrator**

BigO intelligently routes coding tasks to the most cost-effective AI backend. Simple tasks go to free local Ollama models, while complex work uses Claude's advanced reasoning or Gemini's large context window. Save money without sacrificing quality.

```
┌─────────────────────────────────────────────────────────────────┐
│                         BigO Conductor                          │
│  ┌───────────┐  ┌──────────────┐  ┌─────────────────────────┐  │
│  │ Classifier │  │ Task Ledger  │  │     Message Bus         │  │
│  │            │  │ (SQLite)     │  │ (coordination)          │  │
│  └───────────┘  └──────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Claude/Gemini  │  │   Ollama Tier   │  │  Validator Pool │
│                 │  │                 │  │                 │
│ • Opus/Pro (P0) │  │ • phi3 (fast)   │  │ • Blind review  │
│ • Sonnet (P1-2) │  │ • qwen3 (default│  │ • Multi-model   │
│ • Haiku/Flash   │  │ • Remote server │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

## Features

- **Smart Task Classification**: Automatically categorizes tasks by complexity (TRIVIAL → CRITICAL)
- **Cost Optimization**: Routes simple tasks to free Ollama, complex ones to Claude or Gemini
- **Remote Ollama Support**: Run Ollama on a dedicated GPU server
- **Task Persistence**: SQLite ledger for crash recovery and analytics
- **Cost Tracking**: See exactly how much you're saving

## Quick Start

### Prerequisites

- Go 1.21+
- [Ollama](https://ollama.ai) (local or remote)
- [Claude CLI](https://docs.anthropic.com/claude-code) (for Claude backends)
- Gemini API Key (optional, for Gemini backends)

### Installation

#### Homebrew (macOS/Linux)

```bash
brew tap yourusername/tap
brew install bigo
```

#### Download Binary

Download from [Releases](https://github.com/yourusername/bigo/releases):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/yourusername/bigo/releases/latest/download/bigo-darwin-arm64
chmod +x bigo-darwin-arm64
sudo mv bigo-darwin-arm64 /usr/local/bin/bigo

# macOS (Intel)
curl -LO https://github.com/yourusername/bigo/releases/latest/download/bigo-darwin-amd64
chmod +x bigo-darwin-amd64
sudo mv bigo-darwin-amd64 /usr/local/bin/bigo

# Linux (amd64)
curl -LO https://github.com/yourusername/bigo/releases/latest/download/bigo-linux-amd64
chmod +x bigo-linux-amd64
sudo mv bigo-linux-amd64 /usr/local/bin/bigo
```

#### Build from Source

```bash
git clone https://github.com/yourusername/bigo.git
cd bigo
go build ./cmd/bigo
sudo mv bigo /usr/local/bin/
```

### Initialize a Project

```bash
cd your-project
bigo init
```

This creates a `.bigo/` directory with:
- `ledger.db` - SQLite database for task tracking
- `config.yaml` - Configuration file

### Run Your First Task

```bash
# Dry run (classify without executing)
bigo run -n "fix the typo in README.md"

# Execute
bigo run "fix the typo in README.md"

# Check status
bigo status
```

## Architecture

### Two-Machine Setup

BigO is designed for a common setup where you have:

1. **Primary Machine** (laptop/workstation): Runs the `bigo` CLI and Claude
2. **GPU Server**: Runs Ollama with larger models

```
┌──────────────────────┐         ┌──────────────────────┐
│   Primary Machine    │         │     GPU Server       │
│                      │         │                      │
│  ┌────────────────┐  │  HTTP   │  ┌────────────────┐  │
│  │   bigo CLI     │──┼─────────┼─▶│    Ollama      │  │
│  └────────────────┘  │  :11434 │  │                │  │
│          │           │         │  │  • phi3        │  │
│          ▼           │         │  │  • qwen3       │  │
│  ┌────────────────┐  │         │  │  • deepseek    │  │
│  │  Claude/Gemini │  │         │  └────────────────┘  │
│  └────────────────┘  │         │                      │
└──────────────────────┘         └──────────────────────┘
```

### Task Classification

| Tier | Complexity | Backend | Use Case |
|------|------------|---------|----------|
| T0 | TRIVIAL | Ollama (fast) | Typos, formatting, comments |
| T1 | SIMPLE | Ollama (default) | Add function, fix obvious bug |
| T2 | STANDARD | Claude Sonnet / Gemini Flash | New feature, refactoring |
| T3 | COMPLEX | Claude Sonnet / Gemini Pro | Architecture, multi-file |
| T4 | CRITICAL | Claude Opus / Gemini Pro | Security, auth, payments |

## Configuration

### Primary Machine

Edit `.bigo/config.yaml`:

```yaml
conductor:
  classifier_model: claude:sonnet
  max_retries: 3
  validation_timeout: 300s

workers:
  claude:
    enabled: true
    max_concurrent: 2
    models:
      opus: claude-opus-4-5-20251101
      sonnet: claude-sonnet-4-20250514
      haiku: claude-haiku-3-5-20241022
    cost_limits:
      daily_usd: 50.0
      per_task_usd: 5.0

  gemini:
    enabled: true
    api_key: "YOUR_GEMINI_API_KEY"
    models:
      flash: gemini-1.5-flash
      pro: gemini-1.5-pro

  ollama:
    enabled: true
    endpoint: http://your-gpu-server:11434  # Remote Ollama
    max_concurrent: 4
    models:
      fast: phi3:mini-16k      # 3.8B - trivial tasks
      default: qwen3:8b        # 8.2B - simple tasks
      reasoning: qwen3:8b-8k   # Extended context

validators:
  pool_size: 5
  timeout: 120s
  backends:
    - claude:sonnet
    - ollama:qwen3:8b

ledger:
  path: .bigo/ledger.db
```

### GPU Server (Ollama)

See [docs/ollama-server-setup.md](docs/ollama-server-setup.md) for detailed setup.

Quick setup:

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull models
ollama pull phi3:mini-16k
ollama pull qwen3:8b

# Configure for network access
sudo systemctl edit ollama.service
# Add: Environment="OLLAMA_HOST=0.0.0.0"

sudo systemctl restart ollama

# Verify
curl http://localhost:11434/api/tags
```

## Commands

```bash
bigo init              # Initialize in current directory
bigo run "task"        # Execute a task
bigo run -n "task"     # Dry run (classify only)
bigo classify "task"   # Test classifier
bigo status            # View stats and cost savings
bigo config            # View configuration
```

## Cost Savings Example

```
BigO Status
═══════════════════════════════════════
Tasks:      47 total (2 pending, 45 completed)
Executions: 52 total
───────────────────────────────────────
Cost Breakdown:
  Claude/Gemini: $1.2340 (12 tasks)
  Ollama:        $0.0000 (35 tasks)
  Savings:       $1.7500 (58.6%)
═══════════════════════════════════════
```

## Roadmap

- [ ] **Validation System**: Blind validators that review worker output
- [ ] **Parallel Workers**: Multiple concurrent Ollama instances
- [ ] **Kubernetes Support**: Scale Ollama across a cluster
- [ ] **OpenCode Integration**: Use OpenCode for tool-enabled local execution
- [ ] **Web Dashboard**: Visual task management and analytics

## Project Structure

```
bigo/
├── cmd/bigo/              # CLI entry point
├── internal/
│   ├── cli/               # Command implementations
│   ├── conductor/         # Orchestrator and classifier
│   ├── config/            # Configuration management
│   ├── ledger/            # SQLite state management
│   ├── workers/           # Ollama and Claude workers
│   ├── validators/        # Validation system (planned)
│   └── bus/               # Message bus (planned)
├── pkg/types/             # Shared types
├── docs/                  # Documentation
├── examples/              # Example configurations
└── scripts/               # Setup and utility scripts
```

## Inspiration

BigO draws inspiration from:

- [Zeroshot](https://github.com/covibes/zeroshot) - Multi-agent validation architecture
- [Oh-My-Claude-Sisyphus](https://github.com/Yeachan-Heo/oh-my-claude-sisyphus) - Agent orchestration patterns

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
