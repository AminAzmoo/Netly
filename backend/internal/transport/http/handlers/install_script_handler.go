package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type InstallScriptHandler struct {
	logger *logger.Logger
	config *fiber.Config
}

func NewInstallScriptHandler(logger *logger.Logger) *InstallScriptHandler {
	return &InstallScriptHandler{
		logger: logger,
	}
}

func (h *InstallScriptHandler) GetInstallScript(c *fiber.Ctx) error {
	publicURL := c.Get("X-Public-URL")
	if publicURL == "" || publicURL == "https://YOUR-TUNNEL-URL.trycloudflare.com" {
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s", scheme, c.Hostname())
	}
	script := h.generateInstallScript(publicURL)
	return c.Type("text/plain").SendString(script)
}

func (h *InstallScriptHandler) generateInstallScript(publicURL string) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

BACKEND_URL="%s"
AGENT_VERSION="1.0.0"

echo "🚀 Installing Netly Agent..."

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "❌ Unsupported OS"
    exit 1
fi

# Install dependencies
echo "📦 Installing dependencies..."
case "$OS" in
    ubuntu|debian)
        export DEBIAN_FRONTEND=noninteractive
        sudo apt-get update -qq
        sudo apt-get install -y curl wget
        ;;
    centos|rhel|fedora)
        sudo yum install -y curl wget
        ;;
    *)
        echo "⚠️  Unknown OS, skipping dependencies"
        ;;
esac

# Download agent binary
echo "⬇️  Downloading agent..."
AGENT_URL="${BACKEND_URL}/downloads/netly-agent"
sudo curl -fsSL "$AGENT_URL" -o /usr/local/bin/netly-agent || {
    echo "❌ Failed to download agent"
    exit 1
}

sudo chmod +x /usr/local/bin/netly-agent

# Create systemd service
echo "⚙️  Creating systemd service..."
sudo tee /etc/systemd/system/netly-agent.service > /dev/null <<EOF
[Unit]
Description=Netly Agent
After=network.target

[Service]
Type=simple
Environment="BACKEND_URL=${BACKEND_URL}"
ExecStart=/usr/local/bin/netly-agent
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Start service
echo "▶️  Starting agent..."
sudo systemctl daemon-reload
sudo systemctl enable netly-agent
sudo systemctl start netly-agent

# Check status
sleep 2
if sudo systemctl is-active --quiet netly-agent; then
    echo "✅ Netly Agent installed successfully!"
    echo "📊 Status: sudo systemctl status netly-agent"
else
    echo "❌ Agent failed to start. Check logs: sudo journalctl -u netly-agent -f"
    exit 1
fi
`, publicURL)
}
