package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/netly/backend/internal/infrastructure/logger"
)

type TunnelManager struct {
	settingService *SystemSettingService
	logger         *logger.Logger
	cmd            *exec.Cmd
	cancel         context.CancelFunc
	mu             sync.Mutex
	publicURL      string
	isRunning      bool
	fixedURL       string // Manual URL that overrides log parsing
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

	if tm.isRunning {
		return fmt.Errorf("tunnel is already running")
	}

	// Stop existing tunnel if any
	if tm.cmd != nil {
		tm.stopTunnelUnsafe()
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

	tm.isRunning = true
	tm.logger.Info("Cloudflare tunnel started, detecting public URL...")

	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

	// Monitor process and handle output
	go tm.monitorProcess(stdout, stderr, urlRegex)

	return nil
}

func (tm *TunnelManager) StartTunnelWithURL(token, fixedURL string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.isRunning {
		return fmt.Errorf("tunnel is already running")
	}

	// Stop existing tunnel if any
	if tm.cmd != nil {
		tm.stopTunnelUnsafe()
	}

	// Set fixed URL if provided
	tm.fixedURL = fixedURL
	if fixedURL != "" {
		tm.publicURL = fixedURL
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

	tm.isRunning = true
	if fixedURL != "" {
		tm.logger.Info("Cloudflare tunnel started with fixed URL", "url", fixedURL)
	} else {
		tm.logger.Info("Cloudflare tunnel started, detecting public URL...")
	}

	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

	// Monitor process and handle output
	go tm.monitorProcess(stdout, stderr, urlRegex)

	return nil
}

func (tm *TunnelManager) monitorProcess(stdout, stderr io.ReadCloser, urlRegex *regexp.Regexp) {
	defer func() {
		tm.mu.Lock()
		tm.isRunning = false
		tm.mu.Unlock()
	}()

	// Capture stderr for error logging
	var stderrOutput []string

	// Monitor stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			tm.logger.Debugw("cloudflared stdout", "line", line)
			tm.extractURL(line, urlRegex)
		}
	}()

	// Monitor stderr and capture errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrOutput = append(stderrOutput, line)
			tm.logger.Warnw("cloudflared stderr", "line", line)
			tm.extractURL(line, urlRegex)
		}
	}()

	// Wait for process to finish
	err := tm.cmd.Wait()
	if err != nil {
		tm.logger.Errorw("cloudflared process terminated with error", 
			"error", err,
			"stderr_lines", stderrOutput,
			"exit_code", tm.cmd.ProcessState.ExitCode())
	} else {
		tm.logger.Info("cloudflared process terminated normally")
	}
}

func (tm *TunnelManager) extractURL(line string, urlRegex *regexp.Regexp) {
	// Skip URL extraction if we have a fixed URL
	tm.mu.Lock()
	if tm.fixedURL != "" {
		tm.mu.Unlock()
		return
	}
	tm.mu.Unlock()

	if matches := urlRegex.FindString(line); matches != "" {
		tm.mu.Lock()
		if tm.publicURL != matches {
			tm.publicURL = matches
			tm.mu.Unlock()
			
			tm.logger.Infow("tunnel_url_detected", "url", matches)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			settings := map[string]interface{}{"public_url": matches}
			if err := tm.settingService.UpdateSettings(ctx, settings); err != nil {
				tm.logger.Errorw("failed to update public url", "error", err)
			}
		} else {
			tm.mu.Unlock()
		}
	}
}

func (tm *TunnelManager) StopTunnel() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.stopTunnelUnsafe()
}

func (tm *TunnelManager) stopTunnelUnsafe() error {
	if tm.cmd == nil {
		return nil
	}

	tm.logger.Info("Stopping cloudflare tunnel...")

	// Cancel context first
	if tm.cancel != nil {
		tm.cancel()
	}

	// Try graceful shutdown first
	if tm.cmd.Process != nil {
		if err := tm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			tm.logger.Warnw("failed to send SIGTERM", "error", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- tm.cmd.Wait()
		}()

		select {
		case <-time.After(5 * time.Second):
			// Force kill if graceful shutdown fails
			tm.logger.Warn("Graceful shutdown timeout, force killing process")
			if err := tm.cmd.Process.Kill(); err != nil {
				tm.logger.Errorw("failed to kill process", "error", err)
				return err
			}
			<-done // Wait for process to actually exit
		case err := <-done:
			if err != nil && err.Error() != "signal: terminated" {
				tm.logger.Warnw("process exited with error", "error", err)
			}
		}
	}

	tm.cmd = nil
	tm.cancel = nil
	tm.isRunning = false
	tm.fixedURL = ""
	tm.logger.Info("Cloudflare tunnel stopped")
	return nil
}

func (tm *TunnelManager) IsRunning() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.isRunning && tm.cmd != nil && tm.cmd.Process != nil
}

func (tm *TunnelManager) GetPublicURL() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.publicURL
}
