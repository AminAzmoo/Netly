package handlers

import (
    "encoding/json"
    "strconv"
    "strings"

    "github.com/gofiber/fiber/v2"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/core/services"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
)

type AgentHandler struct {
    nodeService ports.NodeService
    logger      *logger.Logger
    keyManager  *services.KeyManager
}

func NewAgentHandler(nodeService ports.NodeService, logger *logger.Logger, keyManager *services.KeyManager) *AgentHandler {
    return &AgentHandler{nodeService: nodeService, logger: logger, keyManager: keyManager}
}

type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	RAMUsage    float64 `json:"ram_usage"`
	RAMTotal    uint64  `json:"ram_total"`
	RAMUsed     uint64  `json:"ram_used"`
	Uptime      uint64  `json:"uptime"`
	NetworkRx   uint64  `json:"network_rx"`
	NetworkTx   uint64  `json:"network_tx"`
	Hostname    string  `json:"hostname"`
	OS          string  `json:"os"`
	Platform    string  `json:"platform"`
	CollectedAt int64   `json:"collected_at"`
}

type HeartbeatRequest struct {
	Stats        *SystemStats `json:"stats"`
	AgentVersion string       `json:"agent_version"`
	Timestamp    int64        `json:"timestamp"`
}

type RegisterNodeRequest struct {
	Token string `json:"token"`
}

func (h *AgentHandler) RegisterNode(c *fiber.Ctx) error {
	var req RegisterNodeRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warnw("register_node_body_parse_failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Token == "" {
		h.logger.Warnw("register_node_missing_token")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token is required"})
	}

	// Validate token format: node-token-{id}
	if !strings.HasPrefix(req.Token, "node-token-") {
		h.logger.Warnw("register_node_invalid_token_format", "token", req.Token)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
	}

	nodeIDStr := strings.TrimPrefix(req.Token, "node-token-")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		h.logger.Warnw("register_node_invalid_node_id", "value", nodeIDStr)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid node ID in token"})
	}

	// Verify node exists
	node, err := h.nodeService.GetNodeByID(c.Context(), uint(nodeID))
	if err != nil {
		h.logger.Warnw("register_node_not_found", "node_id", nodeID)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	// Get SSH public key from KeyManager (will be injected)
	pubKey := h.keyManager.GetPublicKey()
	if pubKey == "" {
		h.logger.Errorw("register_node_key_not_available", "node_id", nodeID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "SSH key not available"})
	}

	h.logger.Infow("register_node_success", "node_id", nodeID, "node_name", node.Name)
	c.Set("Content-Type", "text/plain")
	return c.SendString(pubKey)
}

func (h *AgentHandler) Heartbeat(c *fiber.Ctx) error {
    authHeader := c.Get("Authorization")
    if authHeader == "" {
        h.logger.Warnw("agent_heartbeat_missing_auth")
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization header"})
    }

    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        h.logger.Warnw("agent_heartbeat_invalid_auth_header")
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization header"})
    }

    token := parts[1]
    if !strings.HasPrefix(token, "node-token-") {
        h.logger.Warnw("agent_heartbeat_invalid_token_format")
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
    }

    nodeIDStr := strings.TrimPrefix(token, "node-token-")
    nodeID, err := strconv.Atoi(nodeIDStr)
    if err != nil {
        h.logger.Warnw("agent_heartbeat_invalid_node_id", "value", nodeIDStr)
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid node ID in token"})
    }
	
    var req HeartbeatRequest
    if err := c.BodyParser(&req); err != nil {
        h.logger.Warnw("agent_heartbeat_body_parse_failed", "error", err)
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
    }

    if req.Stats == nil {
        h.logger.Warnw("agent_heartbeat_missing_stats", "node_id", nodeID)
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing stats"})
    }

    statsJSON, _ := json.Marshal(req.Stats)
    var statsMap domain.JSONB
    json.Unmarshal(statsJSON, &statsMap)

    h.logger.Infow("agent_heartbeat_received", 
        "node_id", nodeID, 
        "cpu", req.Stats.CPUUsage, 
        "ram", req.Stats.RAMUsage,
        "ram_total_mb", req.Stats.RAMTotal/1024/1024,
        "ram_used_mb", req.Stats.RAMUsed/1024/1024,
        "uptime", req.Stats.Uptime,
        "hostname", req.Stats.Hostname,
    )

    if err := h.nodeService.UpdateNodeStats(c.Context(), uint(nodeID), statsMap); err != nil {
        h.logger.Errorw("agent_heartbeat_update_failed", "node_id", nodeID, "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    h.logger.Infow("agent_heartbeat_ok", "node_id", nodeID)
    return c.SendStatus(fiber.StatusOK)
}
