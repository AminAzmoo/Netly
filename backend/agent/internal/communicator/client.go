package communicator

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/netly/agent/internal/stats"
    "go.uber.org/zap"
)

type Command struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
	Priority  int    `json:"priority"`
	CreatedAt int64  `json:"created_at"`
}

type HeartbeatRequest struct {
	Stats     *stats.SystemStats `json:"stats"`
	AgentVersion string          `json:"agent_version"`
	Timestamp    int64           `json:"timestamp"`
}

type HeartbeatResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message,omitempty"`
	Commands []Command `json:"commands,omitempty"`
	Config   *RemoteConfig `json:"config,omitempty"`
}

type RemoteConfig struct {
	HeartbeatInterval int `json:"heartbeat_interval,omitempty"`
}

type Client struct {
    backendURL string
    nodeToken  string
    httpClient *http.Client
    version    string
    logger     *zap.Logger
}

type ClientConfig struct {
    BackendURL string
    NodeToken  string
    Timeout    time.Duration
    Version    string
    Logger     *zap.Logger
}

func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

    return &Client{
        backendURL: cfg.BackendURL,
        nodeToken:  cfg.NodeToken,
        version:    cfg.Version,
        httpClient: &http.Client{
            Timeout: timeout,
        },
        logger:     cfg.Logger,
    }
}

func (c *Client) SendHeartbeat(systemStats *stats.SystemStats) (*HeartbeatResponse, error) {
    start := time.Now()
    req := HeartbeatRequest{
        Stats:        systemStats,
        AgentVersion: c.version,
        Timestamp:    time.Now().Unix(),
    }

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

    url := fmt.Sprintf("%s/api/v1/agent/heartbeat", c.backendURL)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.nodeToken))
	httpReq.Header.Set("User-Agent", fmt.Sprintf("NetlyAgent/%s", c.version))

    if c.logger != nil {
        c.logger.Info("agent_heartbeat_request",
            zap.String("url", url),
            zap.Int("payload_bytes", len(body)),
            zap.String("ua", fmt.Sprintf("NetlyAgent/%s", c.version)),
        )
    }
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        if c.logger != nil {
            c.logger.Warn("agent_heartbeat_network_error", zap.Error(err))
        }
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

    if c.logger != nil {
        c.logger.Info("agent_heartbeat_response",
            zap.Int("status", resp.StatusCode),
            zap.Int64("duration_ms", time.Since(start).Milliseconds()),
            zap.Int("resp_bytes", len(respBody)),
        )
    }
    if resp.StatusCode != http.StatusOK {
        if c.logger != nil {
            c.logger.Warn("agent_heartbeat_bad_status", zap.Int("status", resp.StatusCode))
        }
        return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(respBody))
    }

	var heartbeatResp HeartbeatResponse
    if err := json.Unmarshal(respBody, &heartbeatResp); err != nil {
        if c.logger != nil {
            c.logger.Warn("agent_heartbeat_parse_error", zap.Error(err))
        }
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    if c.logger != nil {
        c.logger.Info("agent_heartbeat_parsed", zap.Int("commands", len(heartbeatResp.Commands)))
    }

	return &heartbeatResp, nil
}

// ReportCommandResult reports the result of a command execution back to the backend
func (c *Client) ReportCommandResult(commandID uint, success bool, output string) error {
    start := time.Now()
    payload := map[string]interface{}{
        "command_id": commandID,
        "success":    success,
        "output":     output,
        "timestamp":  time.Now().Unix(),
    }

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agent/command/result", c.backendURL)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.nodeToken))

    if c.logger != nil {
        c.logger.Info("agent_command_report_request",
            zap.String("url", url),
            zap.Uint("command_id", commandID),
            zap.Int("payload_bytes", len(body)),
        )
    }
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        if c.logger != nil {
            c.logger.Warn("agent_command_report_network_error", zap.Error(err), zap.Uint("command_id", commandID))
        }
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if c.logger != nil {
        c.logger.Info("agent_command_report_response",
            zap.Int("status", resp.StatusCode),
            zap.Int64("duration_ms", time.Since(start).Milliseconds()),
            zap.Uint("command_id", commandID),
        )
    }
    if resp.StatusCode != http.StatusOK {
        if c.logger != nil {
            c.logger.Warn("agent_command_report_bad_status", zap.Int("status", resp.StatusCode), zap.Uint("command_id", commandID))
        }
        return fmt.Errorf("server returned status %d", resp.StatusCode)
    }

	return nil
}
