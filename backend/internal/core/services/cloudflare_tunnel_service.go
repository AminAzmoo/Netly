package services

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/netly/backend/internal/infrastructure/logger"
)

type CloudflareTunnelService struct {
	logger *logger.Logger
}

func NewCloudflareTunnelService(logger *logger.Logger) *CloudflareTunnelService {
	return &CloudflareTunnelService{logger: logger}
}

func (s *CloudflareTunnelService) SetupAndStart(localPort int) (string, error) {
	if err := s.installCloudflared(); err != nil {
		return "", fmt.Errorf("failed to install cloudflared: %w", err)
	}

	publicURL, err := s.startQuickTunnel(localPort)
	if err != nil {
		return "", fmt.Errorf("failed to start tunnel: %w", err)
	}

	return publicURL, nil
}

func (s *CloudflareTunnelService) installCloudflared() error {
	if _, err := exec.LookPath("cloudflared"); err == nil {
		return nil
	}

	s.logger.Info("Installing cloudflared...")
	
	cmd := exec.Command("bash", "-c", "sudo curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o /usr/local/bin/cloudflared && sudo chmod +x /usr/local/bin/cloudflared")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (s *CloudflareTunnelService) startQuickTunnel(port int) (string, error) {
	cmd := exec.Command("cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var publicURL string
	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			break
		}
		line := string(buf[:n])
		if strings.Contains(line, "trycloudflare.com") {
			parts := strings.Split(line, "https://")
			if len(parts) > 1 {
				url := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				publicURL = "https://" + url
				break
			}
		}
	}

	if publicURL == "" {
		return "", fmt.Errorf("failed to detect public URL")
	}

	s.logger.Info("Cloudflare Quick Tunnel started", "url", publicURL)
	return publicURL, nil
}
