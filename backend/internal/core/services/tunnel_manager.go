package services

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"

	"github.com/netly/backend/internal/infrastructure/logger"
)

type TunnelManager struct {
	settingService *SystemSettingService
	logger         *logger.Logger
	cmd            *exec.Cmd
	cancel         context.CancelFunc
	mu             sync.Mutex
	publicURL      string
}

func NewTunnelManager(settingService *SystemSettingService, logger *logger.Logger) *TunnelManager {
	return &TunnelManager{
		settingService: settingService,
		logger:         logger,
	}
}

func (tm *TunnelManager) StartTunnel(token string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.cmd != nil && tm.cancel != nil {
		tm.logger.Info("Stopping existing tunnel...")
		tm.cancel()
		tm.cmd.Wait()
	}

	ctx, cancel := context.WithCancel(context.Background())
	tm.cancel = cancel

	tm.cmd = exec.CommandContext(ctx, "cloudflared", "tunnel", "run", "--token", token)
	
	stdout, err := tm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := tm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := tm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}

	tm.logger.Info("Cloudflare tunnel started, detecting public URL...")

	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			tm.logger.Infow("cloudflared_stdout", "line", line)
			
			if matches := urlRegex.FindString(line); matches != "" {
				tm.publicURL = matches
				tm.logger.Infow("tunnel_url_detected", "url", matches)
				
				ctx := context.Background()
				settings := map[string]interface{}{"public_url": matches}
				if err := tm.settingService.UpdateSettings(ctx, settings); err != nil {
					tm.logger.Errorw("failed to update public url", "error", err)
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			tm.logger.Warnw("cloudflared_stderr", "line", line)
			
			if matches := urlRegex.FindString(line); matches != "" {
				tm.publicURL = matches
				tm.logger.Infow("tunnel_url_detected", "url", matches)
				
				ctx := context.Background()
				settings := map[string]interface{}{"public_url": matches}
				if err := tm.settingService.UpdateSettings(ctx, settings); err != nil {
					tm.logger.Errorw("failed to update public url", "error", err)
				}
			}
		}
	}()

	return nil
}

func (tm *TunnelManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.cancel != nil {
		tm.cancel()
		if tm.cmd != nil {
			tm.cmd.Wait()
		}
	}
}

func (tm *TunnelManager) GetPublicURL() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.publicURL
}
