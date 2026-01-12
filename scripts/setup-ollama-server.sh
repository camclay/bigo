#!/bin/bash
#
# BigO Ollama Server Setup Script
#
# This script configures a fresh Ubuntu server to run Ollama
# as a remote inference backend for BigO.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/yourusername/bigo/main/scripts/setup-ollama-server.sh | bash
#
# Or download and run:
#   chmod +x setup-ollama-server.sh
#   ./setup-ollama-server.sh

set -e

echo "╔════════════════════════════════════════════╗"
echo "║     BigO Ollama Server Setup               ║"
echo "╚════════════════════════════════════════════╝"
echo

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "Please run as a regular user (not root)"
    exit 1
fi

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "Cannot detect OS. This script supports Ubuntu/Debian."
    exit 1
fi

echo "Detected OS: $OS"
echo

# Step 1: Install Ollama
echo "Step 1: Installing Ollama..."
if command -v ollama &> /dev/null; then
    echo "  Ollama already installed: $(ollama --version)"
else
    curl -fsSL https://ollama.ai/install.sh | sh
    echo "  Ollama installed successfully"
fi
echo

# Step 2: Configure for network access
echo "Step 2: Configuring network access..."
sudo mkdir -p /etc/systemd/system/ollama.service.d/

cat << 'EOF' | sudo tee /etc/systemd/system/ollama.service.d/override.conf
[Service]
Environment="OLLAMA_HOST=0.0.0.0"
EOF

sudo systemctl daemon-reload
sudo systemctl restart ollama
echo "  Ollama configured to listen on 0.0.0.0:11434"
echo

# Step 3: Pull recommended models
echo "Step 3: Pulling recommended models..."
echo "  This may take a while depending on your connection..."
echo

echo "  Pulling phi3:mini-16k (fast model, ~2GB)..."
ollama pull phi3:mini-16k

echo "  Pulling qwen3:8b (default model, ~5GB)..."
ollama pull qwen3:8b

# Optional larger models - uncomment if you have the VRAM
# echo "  Pulling qwen3:8b-8k (extended context)..."
# ollama pull qwen3:8b-8k

# echo "  Pulling deepseek-coder-v2:16b (coding specialist, ~10GB)..."
# ollama pull deepseek-coder-v2:16b

echo

# Step 4: Configure firewall (optional)
echo "Step 4: Firewall configuration..."
if command -v ufw &> /dev/null; then
    echo "  UFW detected. Allowing port 11434 from private networks..."
    sudo ufw allow from 10.0.0.0/8 to any port 11434
    sudo ufw allow from 172.16.0.0/12 to any port 11434
    sudo ufw allow from 192.168.0.0/16 to any port 11434
    echo "  Firewall rules added"
else
    echo "  UFW not installed. Please configure your firewall manually."
    echo "  Allow TCP port 11434 from your trusted networks."
fi
echo

# Step 5: Verify installation
echo "Step 5: Verifying installation..."
echo

echo "Available models:"
ollama list
echo

echo "Testing API..."
curl -s http://localhost:11434/api/tags | head -c 200
echo
echo

# Get IP address
IP=$(hostname -I | awk '{print $1}')

echo "╔════════════════════════════════════════════╗"
echo "║     Setup Complete!                        ║"
echo "╚════════════════════════════════════════════╝"
echo
echo "Ollama is running at: http://$IP:11434"
echo
echo "To use with BigO, update your .bigo/config.yaml:"
echo
echo "  workers:"
echo "    ollama:"
echo "      endpoint: http://$IP:11434"
echo
echo "Test from your primary machine:"
echo "  curl http://$IP:11434/api/tags"
echo
echo "To pull additional models:"
echo "  ollama pull <model-name>"
echo
echo "View logs:"
echo "  journalctl -u ollama -f"
