package handlers

import (
	"fmt"
	"text/template"

	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/core/services"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type InstallHandler struct {
	settingService *services.SystemSettingService
	logger         *logger.Logger
}

func NewInstallHandler(settingService *services.SystemSettingService, logger *logger.Logger) *InstallHandler {
	return &InstallHandler{
		settingService: settingService,
		logger:         logger,
	}
}

const installScriptTemplate = `#!/bin/bash
set -e

API_URL="{{.APIURL}}"
NODE_TOKEN="{{.NodeToken}}"

echo "🚀 Netly Agent Installer"
echo "========================"

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
   echo "❌ Please run as root (use sudo)"
   exit 1
fi

# Detect OS/Arch
echo "🔍 Detecting system architecture..."
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac
echo "✓ Detected: Linux $ARCH"

# Download agent binary
echo "📥 Downloading netly-agent..."
BINARY_URL="${API_URL}/downloads/netly-agent-${ARCH}"
curl -sfL "$BINARY_URL" -o /tmp/netly-agent
if [ $? -ne 0 ]; then
    echo "❌ Failed to download agent binary"
    exit 1
fi

# Make executable
chmod +x /tmp/netly-agent

# Run agent install command
echo "⚙️  Installing agent..."
/tmp/netly-agent install --server="${API_URL}" --token="${NODE_TOKEN}"

if [ $? -eq 0 ]; then
    echo "✅ Installation complete!"
    rm -f /tmp/netly-agent
else
    echo "❌ Installation failed"
    rm -f /tmp/netly-agent
    exit 1
fi
`

func (h *InstallHandler) GetInstallScript(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing token parameter")
	}

	settings, err := h.settingService.GetSettingsStruct()
	if err != nil {
		h.logger.Errorw("failed to get settings", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	if settings.PublicURL == "" {
		return c.Status(fiber.StatusServiceUnavailable).SendString("System Public URL not ready")
	}

	tmpl, err := template.New("install").Parse(installScriptTemplate)
	if err != nil {
		h.logger.Errorw("failed to parse template", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	data := map[string]string{
		"APIURL":    settings.PublicURL,
		"NodeToken": token,
	}

	c.Set("Content-Type", "text/x-shellscript")
	c.Set("Content-Disposition", "inline; filename=install.sh")

	return tmpl.Execute(c.Response().BodyWriter(), data)
}

func (h *InstallHandler) GetNodeCommand(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	if nodeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing node ID"})
	}

	// Resolve public URL with priority: DB > Config > Request Host
	apiURL := ""
	if settings, err := h.settingService.GetSettingsStruct(); err == nil && settings.PublicURL != "" {
		apiURL = settings.PublicURL
	}
	if apiURL == "" {
		if configURL := c.Get("X-Public-URL"); configURL != "" {
			apiURL = configURL
		} else {
			apiURL = c.BaseURL()
		}
	}

	nodeToken := fmt.Sprintf("node-token-%s", nodeID)
	command := fmt.Sprintf("curl -sfL %s/install.sh?token=%s | sudo bash", apiURL, nodeToken)

	return c.JSON(fiber.Map{
		"command": command,
		"api_url": apiURL,
		"token":   nodeToken,
	})
}
