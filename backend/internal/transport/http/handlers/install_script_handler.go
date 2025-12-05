package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type InstallScriptHandler struct {
	logger *logger.Logger
}

func NewInstallScriptHandler(logger *logger.Logger) *InstallScriptHandler {
	return &InstallScriptHandler{
		logger: logger,
	}
}

func (h *InstallScriptHandler) GetInstallScript(c *fiber.Ctx) error {
	publicURL := c.Get("X-Public-URL")
	// Fallback logic if header is missing or is a tunnel placeholder
	if publicURL == "" || publicURL == "https://YOUR-TUNNEL-URL.trycloudflare.com" {
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s", scheme, c.Hostname())
	}

	// Retrieve token from URL query parameter
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing token parameter")
	}

	script := h.generateInstallScript(publicURL, token)
	return c.Type("text/plain").SendString(script)
}

func (h *InstallScriptHandler) generateInstallScript(publicURL string, token string) string {
	// Properly escape the values for shell safety to prevent injection
	escapedPublicURL := strings.ReplaceAll(publicURL, "'", "'\"'\"'")
	escapedToken := strings.ReplaceAll(token, "'", "'\"'\"'")

	// We use Sprintf to inject Go variables into the Bash script.
	// 1st %s -> BACKEND_URL='%s' (escapedPublicURL)
	// 2nd %s -> token: '%s' (escapedToken)
	return fmt.Sprintf(`#!/bin/bash
set -e

# 1. Define Backend URL globally for the script
BACKEND_URL='%s'
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
        echo "⚠️  Unknown OS, skipping dependencies check"
        ;;
esac

# Create config directory and file
echo "📝 Creating config file..."
sudo mkdir -p /etc/netly
sudo tee /etc/netly/config.yaml > /dev/null <<EOF
server_url: "${BACKEND_URL}"
token: '%s'
EOF

# Download agent binary
echo "⬇️  Downloading agent..."
# Use the bash variable defined at the top
AGENT_URL="${BACKEND_URL}/downloads/netly-agent"
sudo curl -fsSL "$AGENT_URL" -o /usr/local/bin/netly-agent || {
    echo "❌ Failed to download agent from $AGENT_URL"
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
`, escapedPublicURL, escapedToken)
}
