package dto

import "github.com/netly/backend/internal/domain"

type UpdateNodeRequest struct {
	Name       *string           `json:"name,omitempty"`
	SSHPort    *int              `json:"ssh_port,omitempty"`
	Role       *domain.NodeRole  `json:"role,omitempty"`
	Username   *string           `json:"username,omitempty"`
	Password   *string           `json:"password,omitempty"`
	PrivateKey *string           `json:"private_key,omitempty"`
}