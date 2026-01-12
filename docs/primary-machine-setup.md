# Primary Machine Setup Guide

This guide covers setting up your primary development machine to use BigO with a remote Ollama server and local Claude CLI.

## Overview

The primary machine runs:
- **BigO CLI**: The orchestrator that classifies and routes tasks
- **Claude CLI**: For executing Claude-tier tasks (STANDARD, COMPLEX, CRITICAL)

It connects to:
- **Remote Ollama Server**: For executing Ollama-tier tasks (TRIVIAL, SIMPLE)

## Prerequisites

### Required
- Go 1.21+ ([install](https://go.dev/dl/))
- Network access to your Ollama server

### For Claude Integration
- Claude CLI ([install](https://docs.anthropic.com/claude-code))
- Anthropic API key or Claude subscription

## Installation

### 1. Install Go

```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### 2. Install BigO

```bash
# Clone and build
git clone https://github.com/yourusername/bigo.git
cd bigo
go build ./cmd/bigo

# Install globally (optional)
go install ./cmd/bigo

# Or copy to PATH
sudo cp bigo /usr/local/bin/
```

### 3. Install Claude CLI (Optional but Recommended)

```bash
# macOS/Linux
npm install -g @anthropic-ai/claude-code

# Or via pip
pip install claude-cli

# Authenticate
claude auth
```

### 4. Verify Ollama Connection

```bash
# Test connection to your Ollama server
curl http://your-gpu-server:11434/api/tags

# You should see available models listed
```

## Configuration

### Initialize BigO in Your Project

```bash
cd your-project
bigo init
```

### Edit Configuration

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
      daily_usd: 50.0      # Daily spending limit
      per_task_usd: 5.0    # Per-task limit

  ollama:
    enabled: true
    endpoint: http://your-gpu-server:11434  # ← Your Ollama server
    max_concurrent: 4
    models:
      fast: phi3:mini-16k      # For TRIVIAL tasks
      default: qwen3:8b        # For SIMPLE tasks
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

### Global Configuration (Optional)

Create `~/.bigo/config.yaml` for defaults across all projects:

```yaml
workers:
  ollama:
    endpoint: http://your-gpu-server:11434
```

## Usage

### Basic Commands

```bash
# Classify a task (see what tier it would be)
bigo classify "fix typo in README"
# → TRIVIAL (T0) → ollama:fast

bigo classify "implement user authentication"
# → CRITICAL (T4) → claude:opus

# Dry run (classify without executing)
bigo run -n "add input validation to the form"

# Execute a task
bigo run "add a helper function to parse dates"

# Check status and cost savings
bigo status
```

### Example Session

```bash
$ bigo run "fix the typo in the word 'recieve'"
BigO Task Execution
═══════════════════════════════════════
Task: fix the typo in the word 'recieve'
───────────────────────────────────────
Executing...

Status:   done
Backend:  ollama:fast
Duration: 2.1s
Tokens:   156
Cost:     $0.0000
───────────────────────────────────────
Output:
The word 'recieve' should be spelled 'receive'.
The correct spelling follows the "i before e except after c" rule.
```

## Network Configuration

### If Ollama Server is on Same Network

```yaml
ollama:
  endpoint: http://192.168.1.50:11434
  # or
  endpoint: http://gpu-server.local:11434
```

### If Using SSH Tunnel

```bash
# On primary machine, create tunnel
ssh -L 11434:localhost:11434 user@gpu-server -N &
```

```yaml
ollama:
  endpoint: http://localhost:11434
```

### If Using VPN

```yaml
ollama:
  endpoint: http://10.0.0.50:11434  # VPN IP of GPU server
```

## Troubleshooting

### "no available worker for this task tier"

The required backend isn't registered. Check:
1. Ollama endpoint is reachable
2. Required models are pulled on the server
3. Claude CLI is installed (for STANDARD+ tasks)

```bash
# Test Ollama
curl http://your-server:11434/api/tags

# Test Claude
claude --version
```

### "connection refused" to Ollama

1. Verify server is running: `ssh user@server "systemctl status ollama"`
2. Check server is listening on 0.0.0.0: `ssh user@server "ss -tlnp | grep 11434"`
3. Check firewall allows your IP

### Claude tasks failing

```bash
# Re-authenticate
claude auth

# Test manually
echo "Hello" | claude --print
```

### Slow classification

Classification is local (no API calls). If slow:
1. Check if disk is slow (ledger.db access)
2. First run may be slower due to compilation

## Tips

### Cost Control

Set limits in config to prevent runaway spending:

```yaml
workers:
  claude:
    cost_limits:
      daily_usd: 20.0     # Stop after $20/day
      per_task_usd: 2.0   # Warn on expensive tasks
```

### Force a Specific Tier

```bash
# Force to use Ollama even for complex tasks
bigo run --tier trivial "complex task description"
```

### View Task History

```bash
# SQLite database stores all tasks
sqlite3 .bigo/ledger.db "SELECT * FROM tasks ORDER BY created_at DESC LIMIT 10"
```

## Next Steps

- Set up your [Ollama Server](ollama-server-setup.md) if not done
- Configure [CI/CD integration](ci-cd-setup.md) (coming soon)
- Explore [advanced patterns](advanced-patterns.md) (coming soon)
