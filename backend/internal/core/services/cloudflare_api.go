package services

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/netly/backend/internal/infrastructure/logger"
)

// CloudflareAPIClient handles Cloudflare API interactions and token generation
type CloudflareAPIClient struct {
	logger *logger.Logger
}

// tunnelTokenPayload represents the token structure for cloudflared
// IMPORTANT: Keys MUST be lowercase 'a', 't', 's' for cloudflared to accept the token
type tunnelTokenPayload struct {
	AccountTag   string `json:"a"` // Account ID
	TunnelID     string `json:"t"` // Tunnel UUID
	TunnelSecret string `json:"s"` // Base64-encoded tunnel secret
}

// CloudflareCredentials contains the credentials needed to interact with Cloudflare API
type CloudflareCredentials struct {
	Email     string
	GlobalKey string
	AccountID string
}

// TunnelInfo contains information about a created/fetched tunnel
type TunnelInfo struct {
	ID           string
	Name         string
	TunnelSecret string
	Token        string // The final base64-encoded token for cloudflared
}

func NewCloudflareAPIClient(logger *logger.Logger) *CloudflareAPIClient {
	return &CloudflareAPIClient{logger: logger}
}

// CreateOrGetTunnel creates a new tunnel or fetches an existing one and returns a valid token
func (c *CloudflareAPIClient) CreateOrGetTunnel(creds CloudflareCredentials, tunnelName string) (*TunnelInfo, error) {
	if creds.Email == "" || creds.GlobalKey == "" || creds.AccountID == "" {
		return nil, fmt.Errorf("missing required cloudflare credentials (email, global_key, or account_id)")
	}

	if tunnelName == "" {
		tunnelName = "netly-tunnel"
	}

	// Try to create a new tunnel
	tunnelInfo, err := c.createTunnel(creds, tunnelName)
	if err != nil {
		// If tunnel already exists (409), try to fetch it
		c.logger.Warnw("failed to create tunnel, trying to fetch existing", "error", err)

		tunnelInfo, err = c.getTunnelByName(creds, tunnelName)
		if err != nil {
			return nil, fmt.Errorf("failed to create or fetch tunnel: %w", err)
		}
	}

	// Generate the token with correct lowercase keys
	token, err := c.generateToken(creds.AccountID, tunnelInfo.ID, tunnelInfo.TunnelSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	tunnelInfo.Token = token
	return tunnelInfo, nil
}

// createTunnel creates a new Cloudflare tunnel using the CORRECT /tunnels endpoint
// This matches the proven logic from cloudflare_tunnel.go
func (c *CloudflareAPIClient) createTunnel(creds CloudflareCredentials, tunnelName string) (*TunnelInfo, error) {
	// Create tunnel request payload - just the name, Cloudflare generates the secret
	payload := map[string]string{"name": tunnelName}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tunnel payload: %w", err)
	}

	// CRITICAL: Use /tunnels endpoint, NOT /cfd_tunnel
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/tunnels", creds.AccountID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", creds.Email)
	req.Header.Set("X-Auth-Key", creds.GlobalKey)
	req.Header.Set("Content-Type", "application/json")

	c.logger.Infow("creating cloudflare tunnel", "name", tunnelName, "account_id", creds.AccountID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Handle 409 conflict - tunnel already exists, try with unique name
	if resp.StatusCode == 409 {
		c.logger.Warnw("tunnel name conflict, retrying with unique name", "original_name", tunnelName)
		randBytes := make([]byte, 4)
		rand.Read(randBytes)
		newName := fmt.Sprintf("%s-%x", tunnelName, randBytes)

		payload["name"] = newName
		jsonPayload, _ = json.Marshal(payload)

		req, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		req.Header.Set("X-Auth-Email", creds.Email)
		req.Header.Set("X-Auth-Key", creds.GlobalKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send retry request: %w", err)
		}
		defer resp.Body.Close()
		body, _ = io.ReadAll(resp.Body)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("cloudflare API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	// CRITICAL: Parse the tunnel_secret from Cloudflare's response
	// Cloudflare generates and returns the secret - we must use it!
	var result struct {
		Success bool `json:"success"`
		Result  struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			TunnelSecret string `json:"tunnel_secret"` // CRITICAL: Get secret from API response
		} `json:"result"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare error: %s", result.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare API returned unsuccessful response")
	}

	c.logger.Infow("tunnel created successfully",
		"tunnel_id", result.Result.ID,
		"name", result.Result.Name,
		"has_secret", result.Result.TunnelSecret != "")

	return &TunnelInfo{
		ID:           result.Result.ID,
		Name:         result.Result.Name,
		TunnelSecret: result.Result.TunnelSecret, // Use Cloudflare's secret!
	}, nil
}

// getTunnelByName fetches an existing tunnel by name
// NOTE: For existing tunnels, we cannot retrieve the secret - must delete and recreate
func (c *CloudflareAPIClient) getTunnelByName(creds CloudflareCredentials, tunnelName string) (*TunnelInfo, error) {
	// CRITICAL: Use /tunnels endpoint, NOT /cfd_tunnel
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/tunnels?name=%s", creds.AccountID, tunnelName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", creds.Email)
	req.Header.Set("X-Auth-Key", creds.GlobalKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cloudflare API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Result) == 0 {
		return nil, fmt.Errorf("tunnel not found: %s", tunnelName)
	}

	tunnel := result.Result[0]
	c.logger.Infow("found existing tunnel, deleting to recreate with new secret", "tunnel_id", tunnel.ID, "name", tunnel.Name)

	// Delete the existing tunnel so we can recreate with a fresh secret
	if err := c.deleteTunnel(creds, tunnel.ID); err != nil {
		c.logger.Warnw("failed to delete existing tunnel", "error", err)
		// Try creating with a unique name instead
		randBytes := make([]byte, 4)
		rand.Read(randBytes)
		newName := fmt.Sprintf("%s-%x", tunnelName, randBytes)
		return c.createTunnel(creds, newName)
	}

	// Recreate the tunnel to get a fresh secret
	return c.createTunnel(creds, tunnelName)
}

// deleteTunnel deletes a Cloudflare tunnel by ID
func (c *CloudflareAPIClient) deleteTunnel(creds CloudflareCredentials, tunnelID string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/tunnels/%s", creds.AccountID, tunnelID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", creds.Email)
	req.Header.Set("X-Auth-Key", creds.GlobalKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	c.logger.Infow("tunnel deleted successfully", "tunnel_id", tunnelID)
	return nil
}

// generateToken creates a valid cloudflared token with lowercase JSON keys
// Token format: base64({"a":"<AccountID>", "t":"<TunnelID>", "s":"<TunnelSecret>"})
func (c *CloudflareAPIClient) generateToken(accountID, tunnelID, tunnelSecret string) (string, error) {
	// CRITICAL: Use the tunnelTokenPayload struct which has lowercase json tags
	// This ensures the JSON is serialized as {"a":..., "t":..., "s":...}
	// NOT {"A":..., "T":..., "S":...} which would cause cloudflared to reject the token
	payload := tunnelTokenPayload{
		AccountTag:   accountID,
		TunnelID:     tunnelID,
		TunnelSecret: tunnelSecret,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token payload: %w", err)
	}

	c.logger.Debugw("generated token payload", "json", string(jsonBytes))

	// Base64 encode the JSON
	token := base64.StdEncoding.EncodeToString(jsonBytes)

	return token, nil
}

// GenerateTokenFromCredentials generates a token directly from stored credentials
// This is used when we already have the tunnel info stored
func (c *CloudflareAPIClient) GenerateTokenFromCredentials(accountID, tunnelID, tunnelSecret string) (string, error) {
	return c.generateToken(accountID, tunnelID, tunnelSecret)
}
