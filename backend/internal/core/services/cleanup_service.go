package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/domain"
	"github.com/netly/backend/internal/infrastructure/logger"
	"github.com/netly/backend/internal/infrastructure/remote"
	"github.com/netly/backend/pkg/utils/crypto"
	"golang.org/x/crypto/ssh"
)

type CleanupMode string

const (
	CleanupModeSoft CleanupMode = "soft"
	CleanupModeHard CleanupMode = "hard"
)

type CleanupRequest struct {
	NodeID      uint
	Mode        CleanupMode
	Force       bool
	ConfirmText string
	RequestedBy string
}

type CleanupService struct {
	logger        *logger.Logger
	timelineRepo  ports.TimelineRepository
	encryptionKey string
}

func NewCleanupService(logger *logger.Logger) *CleanupService {
	return &CleanupService{logger: logger}
}

func (s *CleanupService) SetTimelineRepo(repo ports.TimelineRepository) {
	s.timelineRepo = repo
}

func (s *CleanupService) SetEncryptionKey(key string) {
	s.encryptionKey = key
}

func (s *CleanupService) CleanupNode(ctx context.Context, req CleanupRequest, node *domain.Node) error {
	if req.Mode == CleanupModeHard {
		if !req.Force || req.ConfirmText != "DELETE NODE" {
			return ErrCleanupValidationFailed
		}
	}

	s.logCleanupEvent(ctx, req.NodeID, req.Mode, "started", fmt.Sprintf("%s operation initiated", req.Mode), nil)

	authData, err := crypto.Decrypt(node.AuthData, s.encryptionKey)
	if err != nil {
		s.logCleanupEvent(ctx, req.NodeID, req.Mode, "failed", "failed to decrypt auth data", map[string]any{"error": err.Error()})
		return fmt.Errorf("failed to decrypt auth data: %w", err)
	}

	var auth authDataPayload
	if err := json.Unmarshal([]byte(authData), &auth); err != nil {
		s.logCleanupEvent(ctx, req.NodeID, req.Mode, "failed", "failed to parse auth data", map[string]any{"error": err.Error()})
		return fmt.Errorf("failed to parse auth data: %w", err)
	}

	client := remote.NewSSHClient(remote.SSHConfig{
		Host:       node.IP,
		Port:       node.SSHPort,
		User:       auth.User,
		Password:   auth.Password,
		PrivateKey: auth.SSHKey,
		Timeout:    30 * time.Second,
	})

	conn, err := client.ConnectWithRetry()
	if err != nil {
		s.logCleanupEvent(ctx, req.NodeID, req.Mode, "failed", "SSH connection failed", map[string]any{"error": err.Error()})
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer conn.Close()

	if req.Mode == CleanupModeSoft {
		err = s.softCleanup(ctx, client, conn)
	} else {
		err = s.hardCleanup(ctx, client, conn, req.NodeID)
	}

	if err != nil {
		s.logCleanupEvent(ctx, req.NodeID, req.Mode, "failed", err.Error(), map[string]any{"error": err.Error()})
		return err
	}

	eventType := "cleanup"
	if req.Mode == CleanupModeHard {
		eventType = "uninstall"
	}
	s.logCleanupEvent(ctx, req.NodeID, req.Mode, "success", fmt.Sprintf("%s completed successfully", eventType), nil)
	return nil
}

func (s *CleanupService) softCleanup(ctx context.Context, client *remote.SSHClient, conn *ssh.Client) error {
	script := `
sudo systemctl stop netly-agent 2>/dev/null || true
sudo systemctl stop sing-box 2>/dev/null || true
sudo rm -rf /tmp/netly-* 2>/dev/null || true
sudo rm -rf /var/log/netly/* 2>/dev/null || true
echo "Soft cleanup completed"
`
	_, err := client.Execute(ctx, conn, script)
	if err != nil {
		return fmt.Errorf("soft cleanup failed: %w", err)
	}

	s.logger.Infow("soft_cleanup_success")
	return nil
}

func (s *CleanupService) hardCleanup(ctx context.Context, client *remote.SSHClient, conn *ssh.Client, nodeID uint) error {
	script := `
sudo systemctl stop netly-agent 2>/dev/null || true
sudo systemctl disable netly-agent 2>/dev/null || true
sudo systemctl stop sing-box 2>/dev/null || true
sudo rm -f /etc/systemd/system/netly-agent.service
sudo rm -f /usr/local/bin/netly-agent
sudo rm -rf /etc/netly
sudo systemctl daemon-reload
echo "Hard uninstall completed"
`
	_, err := client.Execute(ctx, conn, script)
	if err != nil {
		return fmt.Errorf("hard uninstall failed on node %d: %w", nodeID, err)
	}

	s.logger.Infow("hard_cleanup_success", "node_id", nodeID)
	return nil
}

func (s *CleanupService) SoftUninstall(ctx context.Context, node *domain.Node) error {
	req := CleanupRequest{
		NodeID: node.ID,
		Mode:   CleanupModeSoft,
	}
	return s.CleanupNode(ctx, req, node)
}

func (s *CleanupService) HardUninstall(ctx context.Context, client *remote.SSHClient, conn *ssh.Client) error {
	return ErrCleanupDeprecated
}

func (s *CleanupService) logCleanupEvent(ctx context.Context, nodeID uint, mode CleanupMode, status string, msg string, meta map[string]any) {
	if s.timelineRepo == nil {
		return
	}

	eventType := "cleanup"
	if mode == CleanupModeHard {
		eventType = "uninstall"
	}

	if meta == nil {
		meta = make(map[string]any)
	}
	meta["mode"] = string(mode)
	if reqID := ctx.Value("request_id"); reqID != nil {
		meta["request_id"] = reqID
	}

	eventStatus := domain.EventStatusPending
	if status == "success" {
		eventStatus = domain.EventStatusSuccess
	} else if status == "failed" {
		eventStatus = domain.EventStatusFailed
	}

	nid := nodeID
	var jsonbMeta domain.JSONB
	bytes, _ := json.Marshal(meta)
	_ = json.Unmarshal(bytes, &jsonbMeta)

	event := &domain.TimelineEvent{
		Type:         eventType,
		Status:       eventStatus,
		Message:      msg,
		Meta:         jsonbMeta,
		ResourceID:   &nid,
		ResourceType: "node",
		CreatedAt:    time.Now(),
	}

	if err := s.timelineRepo.Create(ctx, event); err != nil {
		s.logger.Errorw("cleanup_timeline_event_failed", "error", err)
	}
}

type authDataPayload struct {
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	SSHKey   string `json:"ssh_key,omitempty"`
}
