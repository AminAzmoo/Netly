package dto

import (
	"net"
	"time"

	"github.com/netly/backend/internal/domain"
)

type CreateNodeRequest struct {
	Name       string       `json:"name" validate:"required"`
	IP         string       `json:"ip" validate:"required"`
	SSHPort    int          `json:"ssh_port"`
	Username   string       `json:"username" validate:"required"`
	Password   string       `json:"password,omitempty"`
	PrivateKey string       `json:"private_key,omitempty"`
	Role       string       `json:"role" validate:"required,oneof=entry exit hybrid"`
	GeoData    domain.JSONB `json:"geo_data,omitempty"`
}

func (r *CreateNodeRequest) Validate() []string {
	var errors []string

	if r.Name == "" {
		errors = append(errors, "name is required")
	}

	if r.IP == "" {
		errors = append(errors, "ip is required")
	} else if net.ParseIP(r.IP) == nil {
		errors = append(errors, "ip is not a valid IP address")
	}

	if r.Username == "" {
		errors = append(errors, "username is required")
	}

	if r.Password == "" && r.PrivateKey == "" {
		errors = append(errors, "either password or private_key is required")
	}

	if r.Role == "" {
		errors = append(errors, "role is required")
	} else if r.Role != "entry" && r.Role != "exit" && r.Role != "hybrid" {
		errors = append(errors, "role must be one of: entry, exit, hybrid")
	}

	return errors
}

func (r *CreateNodeRequest) GetSSHPort() int {
	if r.SSHPort == 0 {
		return 22
	}
	return r.SSHPort
}

func (r *CreateNodeRequest) GetRole() domain.NodeRole {
	switch r.Role {
	case "exit":
		return domain.NodeRoleExit
	case "hybrid":
		return domain.NodeRoleHybrid
	default:
		return domain.NodeRoleEntry
	}
}


type NodeResponse struct {
    ID        uint              `json:"id"`
    Name      string            `json:"name"`
    IP        string            `json:"ip"`
    SSHPort   int               `json:"ssh_port"`
    Role      domain.NodeRole   `json:"role"`
    Status    domain.NodeStatus `json:"status"`
    GeoData   domain.JSONB      `json:"geo_data,omitempty"`
    Stats     domain.JSONB      `json:"stats,omitempty"`
    IsActive  bool              `json:"is_active"`
    CreatedAt time.Time         `json:"created_at"`
    UpdatedAt time.Time         `json:"updated_at"`
}

func NodeToResponse(node *domain.Node) NodeResponse {
    return NodeResponse{
        ID:        node.ID,
        Name:      node.Name,
        IP:        node.IP,
        SSHPort:   node.SSHPort,
        Role:      node.Role,
        Status:    node.Status,
        GeoData:   node.GeoData,
        Stats:     node.Stats,
        IsActive:  node.IsActive,
        CreatedAt: node.CreatedAt,
        UpdatedAt: node.UpdatedAt,
    }
}

func NodesToResponse(nodes []domain.Node) []NodeResponse {
	responses := make([]NodeResponse, len(nodes))
	for i, node := range nodes {
		responses[i] = NodeToResponse(&node)
	}
	return responses
}

type ErrorResponse struct {
	Error   string   `json:"error"`
	Details []string `json:"details,omitempty"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}
