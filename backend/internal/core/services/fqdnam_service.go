package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/domain"
	"github.com/netly/backend/internal/infrastructure/logger"
)

// FQDNAMService manages FQDN (Fully Qualified Domain Name) allocations for services
type fqdnamService struct {
	serviceRepo  ports.ServiceRepository
	settingRepo  ports.SystemSettingRepository
	logger       *logger.Logger
	baseDomain   string
	mu           sync.Mutex
}

type FQDNAMServiceConfig struct {
	ServiceRepo ports.ServiceRepository
	SettingRepo ports.SystemSettingRepository
	Logger      *logger.Logger
	BaseDomain  string // e.g., "vpn.example.com"
}

func NewFQDNAMService(cfg FQDNAMServiceConfig) ports.FQDNAMService {
	return &fqdnamService{
		serviceRepo: cfg.ServiceRepo,
		settingRepo: cfg.SettingRepo,
		logger:      cfg.Logger,
		baseDomain:  cfg.BaseDomain,
	}
}

// AllocateFQDN generates a unique FQDN for a service
func (s *fqdnamService) AllocateFQDN(ctx context.Context, serviceName string, nodeID uint) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get base domain from settings if not configured
	baseDomain := s.baseDomain
	if baseDomain == "" {
		if setting, err := s.settingRepo.Get(ctx, "fqdn_base_domain"); err == nil && setting.Value != "" {
			baseDomain = setting.Value
		} else {
			baseDomain = "local.netly"
		}
	}

	// Generate subdomain from service name
	subdomain := s.sanitizeSubdomain(serviceName)

	// Check if already exists, append random suffix if needed
	existingFQDNs, err := s.getAllocatedFQDNs(ctx)
	if err != nil {
		s.logger.Warnw("failed to get existing FQDNs", "error", err)
	}

	fqdn := fmt.Sprintf("%s.%s", subdomain, baseDomain)
	
	// If FQDN exists, append node ID and random suffix
	if existingFQDNs[fqdn] {
		suffix := s.generateShortID()
		fqdn = fmt.Sprintf("%s-%d-%s.%s", subdomain, nodeID, suffix, baseDomain)
	}

	s.logger.Infow("allocated FQDN", "fqdn", fqdn, "service", serviceName, "node_id", nodeID)
	return fqdn, nil
}

// ReleaseFQDN releases an FQDN allocation
func (s *fqdnamService) ReleaseFQDN(ctx context.Context, fqdn string) error {
	s.logger.Infow("released FQDN", "fqdn", fqdn)
	return nil
}

// GetAllocations returns all current FQDN allocations
func (s *fqdnamService) GetAllocations(ctx context.Context) ([]domain.FQDNAllocation, error) {
	services, err := s.serviceRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	allocations := make([]domain.FQDNAllocation, 0)
	for _, svc := range services {
		// Check if service has FQDN in config
		if svc.Config != nil {
			if fqdn, ok := svc.Config["fqdn"].(string); ok && fqdn != "" {
				allocations = append(allocations, domain.FQDNAllocation{
					FQDN:         fqdn,
					ServiceID:    svc.ID,
					ServiceName:  svc.Name,
					NodeID:       svc.NodeID,
					Protocol:     string(svc.Protocol),
					Port:         svc.ListenPort,
					CreatedAt:    svc.CreatedAt,
				})
			}
		}
	}

	return allocations, nil
}

// ValidateFQDN checks if an FQDN is valid and available
func (s *fqdnamService) ValidateFQDN(ctx context.Context, fqdn string) (bool, error) {
	if fqdn == "" {
		return false, fmt.Errorf("FQDN cannot be empty")
	}

	// Basic validation
	if len(fqdn) > 253 {
		return false, fmt.Errorf("FQDN too long (max 253 characters)")
	}

	parts := strings.Split(fqdn, ".")
	for _, part := range parts {
		if len(part) > 63 {
			return false, fmt.Errorf("label too long (max 63 characters)")
		}
		if part == "" {
			return false, fmt.Errorf("empty label in FQDN")
		}
	}

	// Check if already allocated
	existingFQDNs, err := s.getAllocatedFQDNs(ctx)
	if err != nil {
		return false, err
	}

	if existingFQDNs[fqdn] {
		return false, fmt.Errorf("FQDN already allocated")
	}

	return true, nil
}

// GetBaseDomain returns the configured base domain
func (s *fqdnamService) GetBaseDomain(ctx context.Context) string {
	if s.baseDomain != "" {
		return s.baseDomain
	}
	
	if setting, err := s.settingRepo.Get(ctx, "fqdn_base_domain"); err == nil && setting.Value != "" {
		return setting.Value
	}
	
	return "local.netly"
}

// SetBaseDomain updates the base domain setting
func (s *fqdnamService) SetBaseDomain(ctx context.Context, baseDomain string) error {
	setting := &domain.SystemSetting{
		Key:      "fqdn_base_domain",
		Value:    baseDomain,
		Type:     "string",
		Category: "network",
	}
	return s.settingRepo.Set(ctx, setting)
}

// Helper functions

func (s *fqdnamService) sanitizeSubdomain(name string) string {
	// Convert to lowercase
	subdomain := strings.ToLower(name)
	
	// Replace spaces and underscores with hyphens
	subdomain = strings.ReplaceAll(subdomain, " ", "-")
	subdomain = strings.ReplaceAll(subdomain, "_", "-")
	
	// Remove invalid characters (keep only alphanumeric and hyphens)
	var result strings.Builder
	for _, r := range subdomain {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	
	subdomain = result.String()
	
	// Remove leading/trailing hyphens
	subdomain = strings.Trim(subdomain, "-")
	
	// Limit length
	if len(subdomain) > 32 {
		subdomain = subdomain[:32]
	}
	
	// Default if empty
	if subdomain == "" {
		subdomain = "svc"
	}
	
	return subdomain
}

func (s *fqdnamService) generateShortID() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *fqdnamService) getAllocatedFQDNs(ctx context.Context) (map[string]bool, error) {
	services, err := s.serviceRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	fqdns := make(map[string]bool)
	for _, svc := range services {
		if svc.Config != nil {
			if fqdn, ok := svc.Config["fqdn"].(string); ok && fqdn != "" {
				fqdns[fqdn] = true
			}
		}
	}

	return fqdns, nil
}
