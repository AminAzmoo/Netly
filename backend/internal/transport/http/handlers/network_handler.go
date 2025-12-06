package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type NetworkHandler struct {
	tunnelRepo  ports.TunnelRepository
	serviceRepo ports.ServiceRepository
	nodeRepo    ports.NodeRepository
	logger      *logger.Logger
	ipamConfig  IPAMConfigInfo
	portamConfig PortAMConfigInfo
}

type IPAMConfigInfo struct {
	IPv4CIDR string
	IPv6CIDR string
}

type PortAMConfigInfo struct {
	MinPort int
	MaxPort int
}

type NetworkHandlerConfig struct {
	TunnelRepo   ports.TunnelRepository
	ServiceRepo  ports.ServiceRepository
	NodeRepo     ports.NodeRepository
	Logger       *logger.Logger
	IPAMConfig   IPAMConfigInfo
	PortAMConfig PortAMConfigInfo
}

func NewNetworkHandler(cfg NetworkHandlerConfig) *NetworkHandler {
	return &NetworkHandler{
		tunnelRepo:   cfg.TunnelRepo,
		serviceRepo:  cfg.ServiceRepo,
		nodeRepo:     cfg.NodeRepo,
		logger:       cfg.Logger,
		ipamConfig:   cfg.IPAMConfig,
		portamConfig: cfg.PortAMConfig,
	}
}

// GetNetworkStats returns IPAM and PortAM statistics
func (h *NetworkHandler) GetNetworkStats(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get all tunnels for IP allocation info
	tunnels, err := h.tunnelRepo.GetAll(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tunnels",
		})
	}

	// Get all services for port allocation info
	services, err := h.serviceRepo.GetAll(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch services",
		})
	}

	// Get all nodes
	nodes, err := h.nodeRepo.GetAll(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch nodes",
		})
	}

	// Build IP allocations list
	ipAllocations := make([]fiber.Map, 0)
	for _, t := range tunnels {
		if t.InternalIPv4 != "" {
			ipAllocations = append(ipAllocations, fiber.Map{
				"ip":            t.InternalIPv4,
				"ipv6":          t.InternalIPv6,
				"type":          "tunnel",
				"resource_id":   t.ID,
				"resource_name": t.Name,
				"allocated_at":  t.CreatedAt,
			})
		}
	}

	// Build port allocations list per node
	portAllocations := make([]fiber.Map, 0)
	nodeMap := make(map[uint]string)
	for _, n := range nodes {
		nodeMap[n.ID] = n.Name
	}

	for _, t := range tunnels {
		if t.SourcePort > 0 {
			portAllocations = append(portAllocations, fiber.Map{
				"port":          t.SourcePort,
				"node_id":       t.SourceNodeID,
				"node_name":     nodeMap[t.SourceNodeID],
				"protocol":      t.Protocol,
				"type":          "tunnel",
				"resource_id":   t.ID,
				"resource_name": t.Name,
			})
		}
		if t.DestPort > 0 {
			portAllocations = append(portAllocations, fiber.Map{
				"port":          t.DestPort,
				"node_id":       t.DestNodeID,
				"node_name":     nodeMap[t.DestNodeID],
				"protocol":      t.Protocol,
				"type":          "tunnel",
				"resource_id":   t.ID,
				"resource_name": t.Name,
			})
		}
	}

	for _, s := range services {
		portAllocations = append(portAllocations, fiber.Map{
			"port":          s.ListenPort,
			"node_id":       s.NodeID,
			"node_name":     nodeMap[s.NodeID],
			"protocol":      s.Protocol,
			"type":          "service",
			"resource_id":   s.ID,
			"resource_name": s.Name,
		})
	}

	// Calculate stats
	totalPortRange := h.portamConfig.MaxPort - h.portamConfig.MinPort
	usedPorts := len(portAllocations)

	return c.JSON(fiber.Map{
		"ipam": fiber.Map{
			"ipv4_cidr":        h.ipamConfig.IPv4CIDR,
			"ipv6_cidr":        h.ipamConfig.IPv6CIDR,
			"allocated_count":  len(ipAllocations),
			"allocations":      ipAllocations,
		},
		"portam": fiber.Map{
			"min_port":        h.portamConfig.MinPort,
			"max_port":        h.portamConfig.MaxPort,
			"total_range":     totalPortRange,
			"used_count":      usedPorts,
			"available_count": totalPortRange - usedPorts,
			"allocations":     portAllocations,
		},
		"summary": fiber.Map{
			"total_nodes":    len(nodes),
			"total_tunnels":  len(tunnels),
			"total_services": len(services),
		},
	})
}
