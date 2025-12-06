package handlers

import (
	"fmt"
	"text/template"

	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/core/services"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type InstallHandler struct {
	settingService    *services.SystemSettingService
	logger            *logger.Logger
	fallbackPublicURL string
}

func NewInstallHandler(settingService *services.SystemSettingService, logger *logger.Logger, fallbackPublicURL string) *InstallHandler {
	return &InstallHandler{
		settingService:    settingService,
		logger:            logger,
		fallbackPublicURL: fallbackPublicURL,
	}
}

// FIX:
// 1. Filename changed from config.yaml -> agent.yaml
// 2. YAML keys changed: server_url -> backend_url, token -> node_token
const installScriptTemplate = `#!/bin/bash
set -e

API_URL="{{.APIURL}}"
NODE_TOKEN="{{.NodeToken}}"

echo "🚀 Netly Agent Installer (Fixed)"
echo "=============================="

if [ "$EUID" -ne 0 ]; then 
   echo "❌ Please run as root (use sudo)"
   exit 1
fi

ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

# 1. ساخت فایل کانفیگ با نام و محتوای صحیح
echo "📝 Creating configuration..."
mkdir -p /etc/netly

# اصلاح مهم: تغییر نام فایل به agent.yaml و کلیدها طبق کد Go
cat > /etc/netly/agent.yaml <<EOF
backend_url: "${API_URL}"
node_token: "${NODE_TOKEN}"
log_path: "/var/log/netly-agent.log"
heartbeat_interval: 10s
EOF

# 2. دانلود ایجنت
echo "📥 Downloading netly-agent..."
BINARY_URL="${API_URL}/downloads/netly-agent-${ARCH}"
curl -sfL "$BINARY_URL" -o /usr/local/bin/netly-agent
chmod +x /usr/local/bin/netly-agent

# 3. ساخت سرویس
echo "⚙️  Configuring systemd service..."
cat > /etc/systemd/system/netly-agent.service <<EOF
[Unit]
Description=Netly Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/netly-agent --config /etc/netly/agent.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 4. اجرا
echo "▶️  Starting agent..."
systemctl daemon-reload
systemctl enable netly-agent
systemctl restart netly-agent

sleep 2
if systemctl is-active --quiet netly-agent; then
    echo "✅ Installation complete! Agent is running."
else
    echo "❌ Agent failed to start. Check logs: journalctl -u netly-agent -f"
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

	apiURL := settings.PublicURL
	if apiURL == "" {
		apiURL = h.fallbackPublicURL
	}
	if apiURL == "" {
		return c.Status(fiber.StatusServiceUnavailable).SendString("System Public URL not ready")
	}

	tmpl, err := template.New("install").Parse(installScriptTemplate)
	if err != nil {
		h.logger.Errorw("failed to parse template", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	data := map[string]string{
		"APIURL":    apiURL,
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

	apiURL := ""
	if settings, err := h.settingService.GetSettingsStruct(); err == nil && settings.PublicURL != "" {
		apiURL = settings.PublicURL
	} else if h.fallbackPublicURL != "" {
		apiURL = h.fallbackPublicURL
	} else {
		apiURL = c.BaseURL()
	}

	nodeToken := fmt.Sprintf("node-token-%s", nodeID)
	command := fmt.Sprintf("curl -fL %s/install.sh?token=%s | sudo bash", apiURL, nodeToken)

	return c.JSON(fiber.Map{
		"command": command,
		"api_url": apiURL,
		"token":   nodeToken,
	})
}
