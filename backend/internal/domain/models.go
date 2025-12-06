package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ==================== ENUMS ====================

type NodeRole string

const (
	NodeRoleEntry  NodeRole = "entry"
	NodeRoleExit   NodeRole = "exit"
	NodeRoleHybrid NodeRole = "hybrid"
)

type NodeStatus string

const (
	NodeStatusPending    NodeStatus = "pending"
	NodeStatusInstalling NodeStatus = "installing"
	NodeStatusOnline     NodeStatus = "online"
	NodeStatusOffline    NodeStatus = "offline"
	NodeStatusError      NodeStatus = "error"
)

type TunnelProtocol string

const (
	TunnelProtocolWireGuard TunnelProtocol = "wireguard"
	TunnelProtocolHysteria2 TunnelProtocol = "hysteria2"
	TunnelProtocolReality   TunnelProtocol = "vless_reality"
)

type TunnelType string

const (
	TunnelTypeDirect TunnelType = "direct"
	TunnelTypeChain  TunnelType = "chain"
)

type TunnelStatus string

const (
	TunnelStatusPending TunnelStatus = "pending"
	TunnelStatusActive  TunnelStatus = "active"
	TunnelStatusFailed  TunnelStatus = "failed"
)

type ServiceProtocol string

const (
	ServiceProtocolVLESS    ServiceProtocol = "vless"
	ServiceProtocolHysteria ServiceProtocol = "hysteria2"
	ServiceProtocolTUIC     ServiceProtocol = "tuic"
)

type RoutingMode string

const (
	RoutingModeDirect RoutingMode = "direct"
	RoutingModeTunnel RoutingMode = "tunnel"
	RoutingModeWARP   RoutingMode = "warp"
)

type EventStatus string

const (
	EventStatusPending EventStatus = "pending"
	EventStatusSuccess EventStatus = "success"
	EventStatusFailed  EventStatus = "failed"
)

// ==================== JSONB TYPES ====================

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONB: invalid type")
	}
	return json.Unmarshal(bytes, j)
}

// ==================== ENTITIES ====================

type Node struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Name     string     `gorm:"size:255;not null" json:"name"`
	IP       string     `gorm:"size:45;uniqueIndex;not null" json:"ip"`
	SSHPort  int        `gorm:"default:22" json:"ssh_port"`
	Role     NodeRole   `gorm:"size:20;not null;default:'entry'" json:"role"`
	Status   NodeStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	AuthData string     `gorm:"type:text" json:"-"`
	GeoData  JSONB      `gorm:"type:jsonb" json:"geo_data"`
	Stats    JSONB      `gorm:"type:jsonb" json:"stats"` // Added Stats field
	IsActive bool       `gorm:"default:true" json:"is_active"`

	// WireGuard Keys
	WireGuardPrivateKey string `gorm:"type:text" json:"-"`
	WireGuardPublicKey  string `gorm:"size:255" json:"wireguard_public_key,omitempty"`

	// Private IP for internal communication (Hyper-V/VPC)
	PrivateIP string `gorm:"size:45" json:"private_ip,omitempty"`

	// Last error log for debugging
	LastLog string `gorm:"type:text" json:"last_log,omitempty"`

	// Relationships
	SourceTunnels []Tunnel  `gorm:"foreignKey:SourceNodeID" json:"source_tunnels,omitempty"`
	DestTunnels   []Tunnel  `gorm:"foreignKey:DestNodeID" json:"dest_tunnels,omitempty"`
	Services      []Service `gorm:"foreignKey:NodeID" json:"services,omitempty"`
}

type Tunnel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Name         string         `gorm:"size:255;not null" json:"name"`
	Protocol     TunnelProtocol `gorm:"size:20;not null" json:"protocol"`
	InternalIPv4 string         `gorm:"size:15" json:"internal_ipv4"`
	InternalIPv6 string         `gorm:"size:45" json:"internal_ipv6"`
	SourcePort   int            `gorm:"not null" json:"source_port"`
	DestPort     int            `gorm:"not null" json:"dest_port"`
	Config       JSONB          `gorm:"type:jsonb" json:"config"`
	Status       TunnelStatus   `gorm:"size:20;not null;default:'pending'" json:"status"`
	Type         TunnelType     `gorm:"size:20;default:'direct'" json:"type"`
	Hops         JSONB          `gorm:"type:jsonb" json:"hops"`
	Nodes        JSONB          `gorm:"type:jsonb" json:"nodes"`
	Segments     JSONB          `gorm:"type:jsonb" json:"segments"` // Details for each hop

	// Relationships
	SourceNodeID uint  `gorm:"not null;index" json:"source_node_id"`
	SourceNode   *Node `gorm:"constraint:OnDelete:CASCADE" json:"source_node,omitempty"`
	DestNodeID   uint  `gorm:"not null;index" json:"dest_node_id"`
	DestNode     *Node `gorm:"constraint:OnDelete:CASCADE" json:"dest_node,omitempty"`
}

type Service struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Name         string          `gorm:"size:255;not null" json:"name"`
	Protocol     ServiceProtocol `gorm:"size:20;not null" json:"protocol"`
	ListenPort   int             `gorm:"not null" json:"listen_port"`
	RoutingMode  RoutingMode     `gorm:"size:20;not null;default:'direct'" json:"routing_mode"`
	Config       JSONB           `gorm:"type:jsonb" json:"config"`
	TotalTraffic int64           `gorm:"default:0" json:"total_traffic"`

	// Relationships
	NodeID uint  `gorm:"not null;index" json:"node_id"`
	Node   *Node `gorm:"constraint:OnDelete:CASCADE" json:"node,omitempty"`
}

type TimelineEvent struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Type         string      `gorm:"size:100;not null;index" json:"type"`
	Status       EventStatus `gorm:"size:20;not null;default:'pending';index" json:"status"`
	Message      string      `gorm:"type:text" json:"message"`
	Meta         JSONB       `gorm:"type:jsonb" json:"meta"`
	ResourceID   *uint       `gorm:"index" json:"resource_id,omitempty"`
	ResourceType string      `gorm:"size:100;index" json:"resource_type"`
}

type SystemSettings struct {
	SSHPrivateKey          string
	SSHPublicKey           string
	CloudflareToken        string
	CloudflareEmail        string
	CloudflareGlobalKey    string
	CloudflareAccountID    string
	CloudflareTunnelID     string
	CloudflareTunnelName   string
	CloudflareTunnelSecret string
	PublicURL              string
}

type SystemSetting struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Key      string `gorm:"size:255;uniqueIndex;not null" json:"key"`
	Value    string `gorm:"type:text" json:"value"`
	Type     string `gorm:"size:50;default:'string'" json:"type"`
	Category string `gorm:"size:100;index" json:"category"`
}

// ==================== RESOURCE MANAGEMENT ====================

type IPAllocation struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	NodeID      uint   `gorm:"not null;index" json:"node_id"`
	Node        *Node  `gorm:"constraint:OnDelete:CASCADE" json:"node,omitempty"`
	IPAddress   string `gorm:"size:45;not null;index" json:"ip_address"`
	IPVersion   int    `gorm:"default:4" json:"ip_version"`
	InUse       bool   `gorm:"default:true" json:"in_use"`
	AllocatedTo string `gorm:"size:100" json:"allocated_to"`
}

type PortAllocation struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	NodeID      uint   `gorm:"not null;index" json:"node_id"`
	Node        *Node  `gorm:"constraint:OnDelete:CASCADE" json:"node,omitempty"`
	Port        int    `gorm:"not null" json:"port"`
	Protocol    string `gorm:"size:10;default:'tcp'" json:"protocol"`
	InUse       bool   `gorm:"default:true" json:"in_use"`
	AllocatedTo string `gorm:"size:100" json:"allocated_to"`
}

// Composite unique index for port allocation
func (PortAllocation) TableName() string {
	return "port_allocations"
}

func (IPAllocation) TableName() string {
	return "ip_allocations"
}
