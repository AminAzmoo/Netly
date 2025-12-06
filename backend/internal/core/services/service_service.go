package services

import (
    "context"
    "fmt"
    "sort"
    "sync"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
)

type ServiceServiceConfig struct {
    ServiceRepo ports.ServiceRepository
    NodeRepo    ports.NodeRepository
    TunnelRepo  ports.TunnelRepository
    FQDNAMSvc   ports.FQDNAMService
    Logger      *logger.Logger
    EnableLocks bool
}

type serviceService struct {
    repo        ports.ServiceRepository
    nodeRepo    ports.NodeRepository
    tunnelRepo  ports.TunnelRepository
    fqdnamSvc   ports.FQDNAMService
    logger      *logger.Logger
    mu          sync.Mutex
    locks       map[string]*sync.Mutex
    enableLocks bool
}

func NewServiceService(cfg ServiceServiceConfig) ports.ServiceService {
    return &serviceService{
        repo:        cfg.ServiceRepo,
        nodeRepo:    cfg.NodeRepo,
        tunnelRepo:  cfg.TunnelRepo,
        fqdnamSvc:   cfg.FQDNAMSvc,
        logger:      cfg.Logger,
        locks:       make(map[string]*sync.Mutex),
        enableLocks: cfg.EnableLocks,
    }
}

func (s *serviceService) lockKeys(keys ...string) func() {
    if !s.enableLocks {
        return func() {}
    }
    if len(keys) == 0 {
        return func() {}
    }
    sort.Strings(keys)
    s.mu.Lock()
    acquired := make([]*sync.Mutex, 0, len(keys))
    for _, k := range keys {
        m := s.locks[k]
        if m == nil {
            m = &sync.Mutex{}
            s.locks[k] = m
        }
        acquired = append(acquired, m)
    }
    s.mu.Unlock()
    for _, m := range acquired {
        m.Lock()
    }
    return func() {
        for i := len(acquired) - 1; i >= 0; i-- {
            acquired[i].Unlock()
        }
    }
}

func (s *serviceService) CreateService(ctx context.Context, input ports.CreateServiceInput) (*domain.Service, error) {
    unlock := s.lockKeys(
        fmt.Sprintf("node:%d", input.NodeID),
        fmt.Sprintf("service:%s:%d", input.Name, input.NodeID),
    )
    defer unlock()
    
    // Validate Node exists
    if _, err := s.nodeRepo.GetByID(ctx, input.NodeID); err != nil {
        s.logger.Error("Node not found", map[string]interface{}{"node_id": input.NodeID, "error": err.Error()})
        return nil, err
    }

    // Initialize config if nil
    config := input.Config
    if config == nil {
        config = make(domain.JSONB)
    }

    // Auto-allocate FQDN if FQDNAM service is available
    if s.fqdnamSvc != nil {
        fqdn, err := s.fqdnamSvc.AllocateFQDN(ctx, input.Name, input.NodeID)
        if err != nil {
            s.logger.Warnw("Failed to allocate FQDN", "error", err, "service_name", input.Name)
        } else {
            config["fqdn"] = fqdn
            s.logger.Infow("FQDN allocated for service", "fqdn", fqdn, "service_name", input.Name)
        }
    }

    service := &domain.Service{
        Name:        input.Name,
        Protocol:    input.Protocol,
        NodeID:      input.NodeID,
        ListenPort:  input.ListenPort,
        RoutingMode: input.RoutingMode,
        Config:      config,
    }

    if err := s.repo.Create(ctx, service); err != nil {
        s.logger.Error("Failed to create service", map[string]interface{}{"error": err.Error()})
        return nil, err
    }

    return service, nil
}

func (s *serviceService) GetServices(ctx context.Context) ([]domain.Service, error) {
	return s.repo.GetAll(ctx)
}

func (s *serviceService) GetServiceByID(ctx context.Context, id uint) (*domain.Service, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *serviceService) DeleteService(ctx context.Context, id uint) error {
    svc, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    unlock := s.lockKeys(
        fmt.Sprintf("service:%d", id),
        fmt.Sprintf("node:%d", svc.NodeID),
    )
    defer unlock()
    return s.repo.Delete(ctx, id)
}
