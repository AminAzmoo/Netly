package services

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/netly/backend/internal/config"
	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/domain"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type ipamService struct {
	tunnelRepo ports.TunnelRepository
	logger     *logger.Logger
	ipv4Base   net.IP
	ipv4Mask   net.IPMask
	ipv6Base   net.IP
	ipv6Mask   net.IPMask
	mu         sync.Mutex
}

type IPAMServiceConfig struct {
	TunnelRepo ports.TunnelRepository
	Logger     *logger.Logger
	Config     config.IPAMConfig
}

func NewIPAMService(cfg IPAMServiceConfig) (ports.IPAMService, error) {
	ipv4IP, ipv4Net, err := net.ParseCIDR(cfg.Config.IPv4CIDR)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCIDR, err)
	}

	ipv6IP, ipv6Net, err := net.ParseCIDR(cfg.Config.IPv6CIDR)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCIDR, err)
	}

	return &ipamService{
		tunnelRepo: cfg.TunnelRepo,
		logger:     cfg.Logger,
		ipv4Base:   ipv4IP.Mask(ipv4Net.Mask),
		ipv4Mask:   ipv4Net.Mask,
		ipv6Base:   ipv6IP.Mask(ipv6Net.Mask),
		ipv6Mask:   ipv6Net.Mask,
	}, nil
}

func (s *ipamService) AllocateTunnelIPs(ctx context.Context) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all existing tunnels to find used IPs
	tunnels, err := s.tunnelRepo.GetAll(ctx)
	if err != nil {
		return "", "", ErrIPAllocationFailed
	}

	// Find highest used IPv4 /30 subnet
	ipv4Subnet, err := s.allocateIPv4Subnet(tunnels)
	if err != nil {
		return "", "", err
	}

	// Generate IPv6 ULA based on tunnel count
	ipv6Addr := s.allocateIPv6(len(tunnels))

	s.logger.Infow("allocated tunnel IPs", "ipv4", ipv4Subnet, "ipv6", ipv6Addr)
	return ipv4Subnet, ipv6Addr, nil
}

func (s *ipamService) allocateIPv4Subnet(tunnels []domain.Tunnel) (string, error) {
	// Start from base + 4 (skip network address)
	baseInt := ipToUint32(s.ipv4Base)
	nextSubnet := baseInt + 4 // First usable /30 subnet

	// Find highest used subnet
	for _, t := range tunnels {
		if t.InternalIPv4 == "" {
			continue
		}
		ip, _, err := net.ParseCIDR(t.InternalIPv4)
		if err != nil {
			continue
		}
		tunnelInt := ipToUint32(ip)
		// Move to next /30 block after this one
		if tunnelInt >= nextSubnet {
			nextSubnet = ((tunnelInt / 4) + 1) * 4
		}
	}

	// Check if within range
	maskSize, _ := s.ipv4Mask.Size()
	maxIP := baseInt + (1 << (32 - maskSize)) - 4
	if nextSubnet > maxIP {
		return "", ErrIPRangeExhausted
	}

	// Return as /30 CIDR
	newIP := uint32ToIP(nextSubnet)
	return fmt.Sprintf("%s/30", newIP.String()), nil
}

func (s *ipamService) allocateIPv6(tunnelCount int) string {
	// Generate unique IPv6 ULA based on tunnel index
	// Format: fd00::tunnel_index:1/64
	return fmt.Sprintf("fd00::%d:1/64", tunnelCount+1)
}

func (s *ipamService) ReleaseIPs(ctx context.Context, ipv4, ipv6 string) error {
	// IPs are implicitly released when tunnel is deleted
	// No explicit tracking needed with current implementation
	s.logger.Infow("released tunnel IPs", "ipv4", ipv4, "ipv6", ipv6)
	return nil
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
