# Ollama Server Setup Guide

This guide covers setting up a dedicated Ollama server for BigO. This is the recommended configuration for teams or users with a GPU-equipped server.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Your Network                                │
│                                                                  │
│  ┌─────────────────┐           ┌─────────────────────────────┐  │
│  │ Primary Machine │           │      GPU Server             │  │
│  │ (Laptop/Desktop)│           │  (e.g., cammy-custom2020)   │  │
│  │                 │   HTTP    │                             │  │
│  │  bigo CLI ──────┼───────────┼──▶ Ollama API (:11434)      │  │
│  │                 │           │       │                     │  │
│  │  Claude CLI     │           │       ▼                     │  │
│  │                 │           │  ┌─────────────────────┐    │  │
│  └─────────────────┘           │  │ GPU (CUDA/ROCm)     │    │  │
│                                │  │ • phi3:mini-16k     │    │  │
│                                │  │ • qwen3:8b          │    │  │
│                                │  │ • deepseek-coder    │    │  │
│                                │  └─────────────────────┘    │  │
│                                └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Server Requirements

### Minimum
- 16GB RAM
- NVIDIA GPU with 8GB+ VRAM (for 7B models)
- Ubuntu 22.04+ or similar Linux

### Recommended
- 32GB+ RAM
- NVIDIA GPU with 16GB+ VRAM (for 14B+ models)
- NVMe SSD for model storage
- Stable network connection to primary machine

## Installation

### 1. Install Ollama

```bash
# Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Verify installation
ollama --version
```

### 2. Install NVIDIA Drivers (if using NVIDIA GPU)

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install nvidia-driver-535 nvidia-cuda-toolkit

# Verify
nvidia-smi
```

### 3. Pull Required Models

```bash
# Fast model for trivial tasks (3.8B, ~2GB)
ollama pull phi3:mini-16k

# Default model for simple tasks (8.2B, ~5GB)
ollama pull qwen3:8b

# Extended context for reasoning (8.2B, ~5GB)
ollama pull qwen3:8b-8k

# Optional: Larger coding models
ollama pull deepseek-coder-v2:16b
ollama pull qwen2.5-coder:14b
```

### 4. Configure Network Access

By default, Ollama only listens on localhost. To allow remote connections:

#### Option A: Systemd (Recommended for servers)

```bash
# Create override file
sudo systemctl edit ollama.service
```

Add:
```ini
[Service]
Environment="OLLAMA_HOST=0.0.0.0"
```

Then:
```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

#### Option B: Environment Variable

```bash
# Add to ~/.bashrc or /etc/environment
export OLLAMA_HOST=0.0.0.0

# Restart Ollama
ollama serve
```

### 5. Verify Remote Access

From your primary machine:

```bash
# Replace with your server's hostname/IP
curl http://your-gpu-server:11434/api/tags
```

You should see a JSON list of available models.

## Firewall Configuration

### UFW (Ubuntu)

```bash
# Allow Ollama port from your network
sudo ufw allow from 192.168.1.0/24 to any port 11434

# Or allow from specific machine
sudo ufw allow from 192.168.1.100 to any port 11434
```

### iptables

```bash
# Allow from subnet
sudo iptables -A INPUT -p tcp -s 192.168.1.0/24 --dport 11434 -j ACCEPT
```

## Security Considerations

### Network Segmentation

Ollama has no built-in authentication. Recommendations:

1. **Private Network Only**: Never expose port 11434 to the internet
2. **VPN**: Use WireGuard or similar for remote access
3. **SSH Tunnel**: Forward the port over SSH

```bash
# SSH tunnel from primary machine
ssh -L 11434:localhost:11434 user@gpu-server

# Then configure bigo to use localhost
# endpoint: http://localhost:11434
```

### Reverse Proxy with Auth (Advanced)

Use nginx with basic auth:

```nginx
server {
    listen 11434 ssl;

    auth_basic "Ollama API";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / {
        proxy_pass http://127.0.0.1:11435;  # Ollama on different port
    }
}
```

## Performance Tuning

### GPU Memory

```bash
# Set max VRAM usage (in MB)
export OLLAMA_GPU_MEMORY=12000

# Or in systemd override
Environment="OLLAMA_GPU_MEMORY=12000"
```

### Concurrent Requests

```bash
# Allow more parallel requests (default: 1)
export OLLAMA_NUM_PARALLEL=4
```

### Model Preloading

Keep frequently used models in memory:

```bash
# Preload models at startup
ollama run phi3:mini-16k "" &
ollama run qwen3:8b "" &
```

## Monitoring

### Check GPU Usage

```bash
# Real-time GPU monitoring
watch -n 1 nvidia-smi
```

### Ollama Logs

```bash
# Systemd logs
journalctl -u ollama -f

# Or if running manually
ollama serve 2>&1 | tee /var/log/ollama.log
```

### Health Check Script

```bash
#!/bin/bash
# save as /usr/local/bin/ollama-health

response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:11434/api/tags)
if [ "$response" = "200" ]; then
    echo "Ollama OK"
    exit 0
else
    echo "Ollama FAILED (HTTP $response)"
    exit 1
fi
```

## Troubleshooting

### "connection refused"

1. Check Ollama is running: `systemctl status ollama`
2. Check binding: `ss -tlnp | grep 11434`
3. Check firewall: `sudo ufw status`

### "model not found"

```bash
# List available models
ollama list

# Pull missing model
ollama pull model-name
```

### Slow inference

1. Check GPU is being used: `nvidia-smi` during inference
2. Ensure model fits in VRAM
3. Try a smaller quantization: `ollama pull model:q4_0`

### Out of memory

```bash
# Use smaller models
ollama pull phi3:mini  # Instead of larger variants

# Or reduce context
ollama pull qwen3:8b-4k  # Smaller context window
```

## Model Recommendations by VRAM

| VRAM | Recommended Models |
|------|-------------------|
| 8GB  | phi3:mini, qwen3:8b (q4) |
| 12GB | qwen3:8b, deepseek-coder:6.7b |
| 16GB | qwen2.5-coder:14b, deepseek-coder-v2:16b |
| 24GB | qwen3:32b, deepseek-r1:32b |

## Next Steps

- [Kubernetes Deployment](kubernetes-setup.md) (coming soon)
- [Load Balancing Multiple GPUs](multi-gpu-setup.md) (coming soon)
