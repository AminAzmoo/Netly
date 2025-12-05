package services

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/netly/backend/internal/config"
	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type portamService struct {
	tunnelRepo  ports.TunnelRepository
	serviceRepo ports.ServiceRepository
	logger      *logger.Logger
	minPort     int
	maxPort     int
	mu          sync.Mutex
	rng         *rand.Rand
}

type PortAMServiceConfig struct {
	TunnelRepo  ports.TunnelRepository
	ServiceRepo ports.ServiceRepository
	Logger      *logger.Logger
	Config      config.PortAMConfig
}

func NewPortAMService(cfg PortAMServiceConfig) (ports.PortAMService, error) {
	if cfg.Config.MinPort >= cfg.Config.MaxPort {
		return nil, ErrInvalidPortRange
	}

	return &portamService{
		tunnelRepo:  cfg.TunnelRepo,
		serviceRepo: cfg.ServiceRepo,
		logger:      cfg.Logger,
		minPort:     cfg.Config.MinPort,
		maxPort:     cfg.Config.MaxPort,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (s *portamService) ReservePort(ctx context.Context, nodeID uint, protocol string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usedPorts, err := s.getUsedPorts(ctx, nodeID)
	if err != nil {
		return 0, err
	}

	// Try random ports first for better distribution
	portRange := s.maxPort - s.minPort
	for attempts := 0; attempts < 100; attempts++ {
		port := s.minPort + s.rng.Intn(portRange)
		if !usedPorts[port] {
			s.logger.Infow("reserved port", "node_id", nodeID, "port", port, "protocol", protocol)
			return port, nil
		}
	}

	// Fallback to sequential search
	for port := s.minPort; port <= s.maxPort; port++ {
		if !usedPorts[port] {
			s.logger.Infow("reserved port", "node_id", nodeID, "port", port, "protocol", protocol)
			return port, nil
		}
	}

	return 0, ErrNoPortsAvailable
}

func (s *portamService) ReleasePort(ctx context.Context, nodeID uint, port int, protocol string) error {
	// Ports are implicitly released when tunnel/service is deleted
	s.logger.Infow("released port", "node_id", nodeID, "port", port, "protocol", protocol)
	return nil
}

func (s *portamService) IsPortAvailable(ctx context.Context, nodeID uint, port int, protocol string) (bool, error) {
	usedPorts, err := s.getUsedPorts(ctx, nodeID)
	if err != nil {
		return false, err
	}
	return !usedPorts[port], nil
}

func (s *portamService) getUsedPorts(ctx context.Context, nodeID uint) (map[int]bool, error) {
	usedPorts := make(map[int]bool)

	// Get ports from tunnels (both source and dest)
    tunnels, err := s.tunnelRepo.GetByNodeID(ctx, nodeID)
    if err != nil {
        return nil, err
    }

    for _, t := range tunnels {
        if t.SourceNodeID == nodeID {
            usedPorts[t.SourcePort] = true
        }
        if t.DestNodeID == nodeID {
            usedPorts[t.DestPort] = true
        }
    }

	// Get ports from services
	if s.serviceRepo != nil {
		services, err := s.serviceRepo.GetByNodeID(ctx, nodeID)
		if err == nil {
			for _, svc := range services {
				usedPorts[svc.ListenPort] = true
			}
		}
	}

	return usedPorts, nil
}
