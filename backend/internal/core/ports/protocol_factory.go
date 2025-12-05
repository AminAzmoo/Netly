package ports

import (
	"github.com/netly/backend/internal/domain"
)

// ProtocolFactory generates protocol-specific configurations
type ProtocolFactory interface {
	// GenerateTunnelConfig generates configurations for both ends of a tunnel
	// Returns sourceConfig (for source node) and destConfig (for destination node)
	GenerateTunnelConfig(tunnel *domain.Tunnel) (sourceConfig, destConfig map[string]interface{}, err error)

	// GenerateServiceConfig generates configuration for a service endpoint
	GenerateServiceConfig(service *domain.Service) (config map[string]interface{}, err error)
}

// TunnelConfigOutput holds the generated configurations
type TunnelConfigOutput struct {
	SourceConfig map[string]interface{} `json:"source_config"`
	DestConfig   map[string]interface{} `json:"dest_config"`
	Credentials  map[string]interface{} `json:"credentials"`
}
