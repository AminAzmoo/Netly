package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/domain"
	"github.com/netly/backend/internal/infrastructure/logger"
	"github.com/netly/backend/internal/infrastructure/remote"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	EventTypeInstallation = "AGENT_INSTALLATION"
	AgentBinaryPath       = "/usr/local/bin/netly-agent"
	AgentServicePath      = "/etc/systemd/system/netly-agent.service"
)

type installerService struct {
    timelineRepo ports.TimelineRepository
    logger       *logger.Logger
    enableTaskCorrelation bool
    publicURL    string
}

func NewInstallerService(timelineRepo ports.TimelineRepository, log *logger.Logger, enableTaskCorrelation bool, publicURL string) ports.InstallerService {
    return &installerService{
        timelineRepo: timelineRepo,
        logger:       log,
        enableTaskCorrelation: enableTaskCorrelation,
        publicURL:    publicURL,
    }
}

func (s *installerService) ValidateBinaryExistence() error {
	binaryPaths := []string{
		"bin/netly-agent",
		"agent/netly-agent",
		"../agent/netly-agent",
	}

	for _, path := range binaryPaths {
		if _, err := os.Stat(path); err == nil {
			s.logger.Infow("agent_binary_found", "path", path)
			return nil
		}
	}

	return fmt.Errorf("agent binary not found in any expected location: %v", binaryPaths)
}

func (s *installerService) InstallAgent(ctx context.Context, node *domain.Node, authData string) error {
	var auth authDataPayload
	if err := json.Unmarshal([]byte(authData), &auth); err != nil {
		return ErrInstallationFailed
	}

	// Log start
	s.logEvent(ctx, node.ID, domain.EventStatusPending, "Starting agent installation", nil)

	sshClient := remote.NewSSHClient(remote.SSHConfig{
		Host:       node.IP,
		Port:       node.SSHPort,
		User:       auth.User,
		Password:   auth.Password,
		PrivateKey: auth.SSHKey,
		Timeout:    30 * time.Second,
		MaxRetries: 5,
	})

	// Establish a persistent SSH connection for the entire installation process
	// This prevents rate-limiting/firewall issues caused by opening multiple connections
	conn, err := sshClient.ConnectWithRetry()
	if err != nil {
		s.logger.Errorw("failed to establish ssh connection", "node_id", node.ID, "error", err)
		s.logEvent(ctx, node.ID, domain.EventStatusFailed, "SSH connection failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("ssh connection failed: %w", err)
	}
	// We use a pointer to the client pointer so we can update it if reconnection happens
	currentConn := conn
	defer func() {
		if currentConn != nil {
			currentConn.Close()
		}
	}()

	// Step 1: System check
	s.logger.Infow("checking system", "node_id", node.ID, "host", node.IP, "port", node.SSHPort)
	cmdCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	systemInfo, err := s.executeWithRetry(cmdCtx, sshClient, &currentConn, "uname -a")
	if err != nil {
		s.logger.Errorw("system check failed", "node_id", node.ID, "error", err)
		s.logEvent(ctx, node.ID, domain.EventStatusFailed, "System check failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("%w: %v", ErrSystemCheckFailed, err)
	}
	s.logger.Infow("system info", "node_id", node.ID, "info", strings.TrimSpace(systemInfo))

	// Architecture check
	archCtx, archCancel := context.WithTimeout(ctx, 30*time.Second)
	defer archCancel()
	arch, err := s.executeWithRetry(archCtx, sshClient, &currentConn, "uname -m")
	if err != nil {
		s.logger.Warnw("architecture check failed", "node_id", node.ID, "error", err)
	} else {
		arch = strings.TrimSpace(arch)
		s.logger.Infow("remote architecture", "node_id", node.ID, "arch", arch)
		if arch != "x86_64" && arch != "amd64" {
			s.logger.Warnw("architecture mismatch", "node_id", node.ID, "expected", "amd64", "got", arch)
		}
	}

	// Step 2: Detect OS and install dependencies
	s.logger.Infow("installing dependencies", "node_id", node.ID)
	// Increase timeout for dependency installation as apt update can be slow
	depCtx, depCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer depCancel()

	if err := s.installDependencies(depCtx, sshClient, &currentConn, systemInfo); err != nil {
		s.logEvent(ctx, node.ID, domain.EventStatusFailed, "Dependency installation failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("%w: %v", ErrDependencyInstall, err)
	}

	// Step 3: Deploy agent binary
	s.logger.Infow("deploying agent", "node_id", node.ID)
	if err := s.deployAgent(ctx, sshClient, &currentConn, node.ID); err != nil {
		s.logEvent(ctx, node.ID, domain.EventStatusFailed, "Agent deployment failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("%w: %v", ErrAgentDeployFailed, err)
	}

	// Step 3.5: Ensure SSH persistence (Anti-Lockout)
	s.logger.Infow("ensuring ssh access", "node_id", node.ID)
	if err := s.EnsureSSHAccess(ctx, sshClient, &currentConn); err != nil {
		// We log this but don't fail the installation, as it's a safety measure
		s.logger.Warnw("failed to ensure ssh persistence", "node_id", node.ID, "error", err)
	}

	// Step 4: Create and start systemd service
	s.logger.Infow("starting service", "node_id", node.ID)
	if err := s.startService(ctx, sshClient, &currentConn); err != nil {
		s.logEvent(ctx, node.ID, domain.EventStatusFailed, "Service start failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("%w: %v", ErrServiceStartFailed, err)
	}

	// Log success
	s.logEvent(ctx, node.ID, domain.EventStatusSuccess, "Agent installation completed", map[string]interface{}{
		"system_info": strings.TrimSpace(systemInfo),
	})

	s.logger.Infow("agent installation completed", "node_id", node.ID)
	return nil
}

func (s *installerService) installDependencies(ctx context.Context, client *remote.SSHClient, conn **ssh.Client, systemInfo string) error {
	var installCmd string

	// Ensure non-interactive mode for apt/yum
	// Use sudo for all commands

	systemInfo = strings.ToLower(systemInfo)
	// Helper for waiting for locks
	waitLockCmd := "while fuser /var/lib/dpkg/lock >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do echo 'Waiting for apt lock...'; sleep 3; done"

	// Apt options to be non-interactive and robust
	aptOpts := "-y -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' -o Acquire::Retries=3"

	switch {
	case strings.Contains(systemInfo, "ubuntu"), strings.Contains(systemInfo, "debian"):
		installCmd = fmt.Sprintf("export DEBIAN_FRONTEND=noninteractive && "+
			"%s && "+
			"(sudo -E apt-get update -o Acquire::Retries=3 -o Acquire::http::Timeout=20 || true) && "+
			"%s && "+
			"sudo -E apt-get install %s --fix-missing wireguard-tools iptables curl",
			waitLockCmd, waitLockCmd, aptOpts)

	case strings.Contains(systemInfo, "centos"), strings.Contains(systemInfo, "rhel"), strings.Contains(systemInfo, "fedora"):
		installCmd = "sudo yum install -y wireguard-tools iptables curl"
	case strings.Contains(systemInfo, "arch"):
		installCmd = "sudo pacman -Sy --noconfirm wireguard-tools iptables curl"
	default:
		// Fallback to debian-style
		installCmd = fmt.Sprintf("export DEBIAN_FRONTEND=noninteractive && "+
			"%s && "+
			"(sudo -E apt-get update -o Acquire::Retries=3 || true) && "+
			"%s && "+
			"sudo -E apt-get install %s --fix-missing wireguard-tools iptables curl",
			waitLockCmd, waitLockCmd, aptOpts)
	}

	// Execute installation
	output, err := s.executeWithRetry(ctx, client, conn, installCmd)
	if err != nil {
		// Log the actual output to understand WHY it failed
		s.logger.Errorw("dependency installation command failed",
			"error", err,
			"output", output,
			"command", installCmd,
		)
		// Return output as part of error for better debugging in frontend
		return fmt.Errorf("%w: %s (Output: %s)", err, "command execution failed", output)
	}

	s.logger.Infow("dependencies installed successfully", "output_summary", "check logs for full output")
	return nil
}

func (s *installerService) deployAgent(ctx context.Context, client *remote.SSHClient, conn **ssh.Client, nodeID uint) error {
	binaryPaths := []string{
		"bin/netly-agent",
		"agent/netly-agent",
		"../agent/netly-agent",
	}

	var localFile *os.File
	var err error
	var localPath string
	for _, path := range binaryPaths {
		localFile, err = os.Open(path)
		if err == nil {
			localPath = path
			break
		}
	}
	if err != nil {
		return fmt.Errorf("agent binary not found. compile it first: GOOS=linux GOARCH=amd64 go build -o bin/netly-agent ./agent/cmd/agent")
	}
	defer localFile.Close()

	stat, _ := localFile.Stat()
	localSize := stat.Size()
	s.logger.Infow("uploading agent binary", "node_id", nodeID, "path", localPath, "size_bytes", localSize)

	sftpClient, err := sftp.NewClient(*conn)
	if err != nil {
		return fmt.Errorf("failed to create sftp client: %w", err)
	}
	defer sftpClient.Close()

	tempPath := "/tmp/netly-agent"
	remoteFile, err := sftpClient.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}

	written, err := remoteFile.ReadFrom(localFile)
	if err != nil {
		remoteFile.Close()
		return fmt.Errorf("failed to upload binary: %w", err)
	}
	remoteFile.Close()
	s.logger.Infow("binary uploaded", "node_id", nodeID, "bytes_written", written)

	if written != localSize {
		s.logger.Errorw("upload size mismatch", "node_id", nodeID, "expected", localSize, "got", written)
		return fmt.Errorf("upload incomplete: expected %d bytes, got %d", localSize, written)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	moveCmd := fmt.Sprintf("sudo mv %s %s && sudo chmod +x %s && ls -lh %s", tempPath, AgentBinaryPath, AgentBinaryPath, AgentBinaryPath)
	output, err := s.executeWithRetry(cmdCtx, client, conn, moveCmd)
	if err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}
	s.logger.Infow("binary installed", "node_id", nodeID, "ls_output", strings.TrimSpace(output))

	return nil
}

func (s *installerService) EnsureSSHAccess(ctx context.Context, client *remote.SSHClient, conn **ssh.Client) error {
	// Commands to ensure port 22 is ALWAYS open and persisted
	commands := []string{
		// 1. IPTables: Insert rule at TOP of chain with highest priority
		"sudo iptables -C INPUT -p tcp --dport 22 -j ACCEPT 2>/dev/null || sudo iptables -I INPUT 1 -p tcp --dport 22 -j ACCEPT",
		
		// 2. Also add to OUTPUT chain to ensure responses can go out
		"sudo iptables -C OUTPUT -p tcp --sport 22 -j ACCEPT 2>/dev/null || sudo iptables -I OUTPUT 1 -p tcp --sport 22 -j ACCEPT",
		
		// 3. Save iptables rules permanently
		"sudo sh -c 'iptables-save > /etc/iptables/rules.v4' || true",
		"sudo sh -c 'mkdir -p /etc/iptables && iptables-save > /etc/iptables/rules.v4' || true",
		
		// 4. Install and configure iptables-persistent
		"export DEBIAN_FRONTEND=noninteractive && sudo -E apt-get install -y iptables-persistent netfilter-persistent 2>/dev/null || true",
		"sudo netfilter-persistent save 2>/dev/null || true",
		
		// 5. UFW: If active, allow SSH explicitly and reload
		"sudo ufw status 2>/dev/null | grep -q 'Status: active' && (sudo ufw allow 22/tcp && sudo ufw reload) || true",
		
		// 6. Create a systemd service to ensure SSH is always allowed on boot
		"sudo sh -c 'cat > /etc/systemd/system/ensure-ssh-access.service << EOF\n[Unit]\nDescription=Ensure SSH Access\nBefore=network.target\n\n[Service]\nType=oneshot\nExecStart=/sbin/iptables -I INPUT 1 -p tcp --dport 22 -j ACCEPT\nRemainAfterExit=yes\n\n[Install]\nWantedBy=multi-user.target\nEOF' || true",
		"sudo systemctl daemon-reload || true",
		"sudo systemctl enable ensure-ssh-access.service || true",
		"sudo systemctl start ensure-ssh-access.service || true",
	}

	for _, cmd := range commands {
		cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, _ = s.executeWithRetry(cmdCtx, client, conn, cmd)
		cancel()
	}

	return nil
}

func (s *installerService) startService(ctx context.Context, client *remote.SSHClient, conn **ssh.Client) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Netly Agent
After=network.target

[Service]
Type=simple
ExecStart=%s start
Restart=always
RestartSec=5
Environment="NETLY_BACKEND_URL=%s"
Environment="NETLY_AGENT_TOKEN=%s"
Environment="LOG_LEVEL=info"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target`, AgentBinaryPath, s.publicURL, "change-me-agent")

	createServiceCmd := fmt.Sprintf("sudo bash -c 'cat > %s <<EOF\n%s\nEOF'", AgentServicePath, serviceContent)
	
	// 30s timeout for service creation
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	if _, err := s.executeWithRetry(cmdCtx, client, conn, createServiceCmd); err != nil {
		return err
	}

	// Reload systemd and enable service
	prepCommands := []string{
		"sudo systemctl daemon-reload",
		"sudo systemctl enable netly-agent",
	}

	for _, cmd := range prepCommands {
		stepCtx, stepCancel := context.WithTimeout(ctx, 30*time.Second)
		if _, err := s.executeWithRetry(stepCtx, client, conn, cmd); err != nil {
			stepCancel()
			return err
		}
		stepCancel()
	}

	// Start service
	startCmd := "sudo systemctl start netly-agent"
	
	startCtx, startCancel := context.WithTimeout(ctx, 20*time.Second)
	defer startCancel()

	if _, err := s.executeWithRetry(startCtx, client, conn, startCmd); err != nil {
		s.logger.Warnw("service start command returned error, but may still be running", "error", err)
	}
	
	return nil
}

func (s *installerService) logEvent(ctx context.Context, resourceID uint, status domain.EventStatus, msg string, meta map[string]interface{}) {
    if s.timelineRepo == nil {
        return
    }

    // Convert meta to JSONB
    var metadata domain.JSONB
    if meta != nil {
        bytes, _ := json.Marshal(meta)
        _ = json.Unmarshal(bytes, &metadata)
    } else {
        metadata = make(domain.JSONB)
    }

    if s.enableTaskCorrelation {
        if v := ctx.Value("task_id"); v != nil {
            metadata["task_id"] = v
        }
    }
    if v := ctx.Value("request_id"); v != nil {
        metadata["request_id"] = v
    }

    event := &domain.TimelineEvent{
        Type:         EventTypeInstallation,
        Status:       status,
        Message:      msg,
        ResourceType: "node",
        ResourceID:   &resourceID,
        Meta:         metadata,
        CreatedAt:    time.Now(),
    }

	if err := s.timelineRepo.Create(ctx, event); err != nil {
		s.logger.Errorw("failed to log timeline event", "error", err)
	}
}

// executeWithRetry attempts to execute a command, and if it fails due to network error,
// it reconnects and retries. It updates the connection pointer if reconnected.
func (s *installerService) executeWithRetry(ctx context.Context, client *remote.SSHClient, conn **ssh.Client, cmd string) (string, error) {
	// Try execution with current connection
	output, err := client.Execute(ctx, *conn, cmd)
	if err == nil {
		return output, nil
	}

	// Check if error is retryable (network related)
	// "broken pipe", "EOF", "connection reset", "shutdown" are common network errors
	errStr := err.Error()
	isNetworkError := strings.Contains(errStr, "broken pipe") || 
		strings.Contains(errStr, "EOF") || 
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "shutdown") ||
		strings.Contains(errStr, "client is closed")

	if !isNetworkError {
		return output, err
	}

	s.logger.Warnw("ssh connection lost during command execution, attempting to reconnect", "error", err, "command", cmd)

	// Close old connection explicitly (just in case)
	if *conn != nil {
		(*conn).Close()
	}

	// Reconnect
	newConn, reconnectErr := client.ConnectWithRetry()
	if reconnectErr != nil {
		return "", fmt.Errorf("failed to reconnect after network error: %w (original error: %v)", reconnectErr, err)
	}

	// Update the connection pointer so caller uses the new one
	*conn = newConn
	s.logger.Infow("ssh reconnected successfully, retrying command")

	// Retry command with new connection
	return client.Execute(ctx, *conn, cmd)
}
