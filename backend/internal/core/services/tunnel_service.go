package services

import (
    "context"
    "encoding/json"
    "net"
    "time"
    "fmt"
    "sort"
    "sync"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/core/services/factory"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
)

type tunnelService struct {
    tunnelRepo ports.TunnelRepository
    nodeRepo   ports.NodeRepository
    ipam       ports.IPAMService
    portam     ports.PortAMService
    factory    *factory.FactoryService
    logger     *logger.Logger
    timelineRepo ports.TimelineRepository
    mu        sync.Mutex
    locks     map[string]*sync.Mutex
}

type TunnelServiceConfig struct {
    TunnelRepo ports.TunnelRepository
    NodeRepo   ports.NodeRepository
    IPAM       ports.IPAMService
    PortAM     ports.PortAMService
    Factory    *factory.FactoryService
    Logger     *logger.Logger
    TimelineRepo ports.TimelineRepository
}

func NewTunnelService(cfg TunnelServiceConfig) ports.TunnelService {
    return &tunnelService{
        tunnelRepo: cfg.TunnelRepo,
        nodeRepo:   cfg.NodeRepo,
        ipam:       cfg.IPAM,
        portam:     cfg.PortAM,
        factory:    cfg.Factory,
        logger:     cfg.Logger,
        timelineRepo: cfg.TimelineRepo,
        locks:      make(map[string]*sync.Mutex),
    }
}

func (s *tunnelService) CreateTunnel(ctx context.Context, input ports.CreateTunnelInput) (*domain.Tunnel, error) {
    start := time.Now()
    step := time.Now()
    s.logger.Infow("tunnel_create_start", "source_node_id", input.SourceNodeID, "dest_node_id", input.DestNodeID, "protocol", input.Protocol)
    s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelInit, domain.EventStatusPending, "Initializing Direct Tunnel", map[string]interface{}{
        "source_node_id": input.SourceNodeID,
        "dest_node_id": input.DestNodeID,
        "protocol":      input.Protocol,
        "topology":      "direct",
    })
    // Validate nodes exist and are different
    if input.SourceNodeID == input.DestNodeID {
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Source and destination are the same", map[string]interface{}{
            "source_node_id": input.SourceNodeID,
            "dest_node_id":   input.DestNodeID,
            "step":           "validation",
        })
        return nil, ErrTunnelSameNode
    }

    unlock := s.lockKeys(
        fmt.Sprintf("node:%d", input.SourceNodeID),
        fmt.Sprintf("node:%d", input.DestNodeID),
        fmt.Sprintf("tunnel:%d:%d", input.SourceNodeID, input.DestNodeID),
    )
    defer unlock()

    sourceNode, err := s.nodeRepo.GetByID(ctx, input.SourceNodeID)
    if err != nil {
        s.logger.Errorw("source node not found", "node_id", input.SourceNodeID)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Source node not found", map[string]interface{}{
            "source_node_id": input.SourceNodeID,
            "step":           "get_nodes",
        })
        return nil, ErrNodeNotFound
    }

    destNode, err := s.nodeRepo.GetByID(ctx, input.DestNodeID)
    if err != nil {
        s.logger.Errorw("dest node not found", "node_id", input.DestNodeID)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Destination node not found", map[string]interface{}{
            "dest_node_id": input.DestNodeID,
            "step":         "get_nodes",
        })
        return nil, ErrNodeNotFound
    }
    s.logger.Infow("tunnel_create_step", "step", "get_nodes", "duration_ms", time.Since(step).Milliseconds(), "elapsed_ms", time.Since(start).Milliseconds())
    step = time.Now()

    // Allocate internal IPs
    ipv4Subnet, ipv6ULA, err := s.ipam.AllocateTunnelIPs(ctx)
    if err != nil {
        s.logger.Errorw("failed to allocate IPs", "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "IPAM allocation failed", map[string]interface{}{
            "error": err.Error(),
            "step":  "allocate_ips",
        })
        return nil, err
    }
    s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelIPAM, domain.EventStatusPending, "Allocated IPs for direct tunnel", map[string]interface{}{
        "ipv4": ipv4Subnet,
        "ipv6": ipv6ULA,
    })
    s.logger.Infow("tunnel_create_step", "step", "allocate_ips", "duration_ms", time.Since(step).Milliseconds(), "elapsed_ms", time.Since(start).Milliseconds())
    step = time.Now()

    // Reserve ports on both nodes
    sourcePort, err := s.portam.ReservePort(ctx, sourceNode.ID, string(input.Protocol))
    if err != nil {
        s.logger.Errorw("failed to reserve source port", "node_id", sourceNode.ID, "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Port reservation failed (source)", map[string]interface{}{
            "node_id": sourceNode.ID,
            "error":   err.Error(),
            "step":    "reserve_ports",
        })
        return nil, err
    }

    destPort, err := s.portam.ReservePort(ctx, destNode.ID, string(input.Protocol))
    if err != nil {
        s.logger.Errorw("failed to reserve dest port", "node_id", destNode.ID, "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Port reservation failed (dest)", map[string]interface{}{
            "node_id": destNode.ID,
            "error":   err.Error(),
            "step":    "reserve_ports",
        })
        return nil, err
    }
    s.logger.Infow("tunnel_create_step", "step", "reserve_ports", "duration_ms", time.Since(step).Milliseconds(), "elapsed_ms", time.Since(start).Milliseconds(), "source_port", sourcePort, "dest_port", destPort)
    step = time.Now()

    // Derive /30 host IPs (server <-> client)
    serverWGIP, clientWGIP, err := deriveWGIPs(ipv4Subnet)
    if err != nil {
        s.logger.Errorw("failed to derive wg ips", "error", err, "subnet", ipv4Subnet)
        return nil, err
    }

    // Generate Protocol Configuration
    configParams := factory.ConfigParams{
        Protocol:   string(input.Protocol),
        Port:       destPort,
        ServerIP:   destNode.IP,
        SNI:        "yahoo.com", // Should come from input or domain setting
        ClientIP:   clientWGIP,
        ServerWGIP: serverWGIP,
    }

    configResult, err := s.factory.GenerateConfig(configParams)
    if err != nil {
        s.logger.Errorw("failed to generate config", "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "ProtocolFactory config generation failed", map[string]interface{}{
            "error": err.Error(),
            "step":  "generate_config",
        })
        return nil, err
    }
    s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelConfig, domain.EventStatusPending, "Configs generated via ProtocolFactory", map[string]interface{}{
        "protocol": input.Protocol,
    })
    s.logger.Infow("tunnel_create_step", "step", "generate_config", "duration_ms", time.Since(step).Milliseconds(), "elapsed_ms", time.Since(start).Milliseconds())
    step = time.Now()

	// Map ConfigResult to JSONB
	configData := domain.JSONB{
		"inbound":       configResult.Inbound,
		"client_config": configResult.ClientConfig,
		"metadata":      configResult.Metadata,
	}

    tunnel := &domain.Tunnel{
        Name:         input.Name,
        Protocol:     input.Protocol,
        SourceNodeID: input.SourceNodeID,
        DestNodeID:   input.DestNodeID,
        SourcePort:   sourcePort,
        DestPort:     destPort,
        InternalIPv4: ipv4Subnet,
        InternalIPv6: ipv6ULA,
        Config:       configData,
        Status:       domain.TunnelStatusPending,
        Type:         domain.TunnelTypeDirect,
        Hops:         domain.JSONB{"nodes": []uint{input.SourceNodeID, input.DestNodeID}},
        Nodes:        domain.JSONB{"nodes": []uint{input.SourceNodeID, input.DestNodeID}},
    }

    if err := s.tunnelRepo.Create(ctx, tunnel); err != nil {
        s.logger.Errorw("failed to create tunnel", "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Persist tunnel failed", map[string]interface{}{
            "error": err.Error(),
            "step":  "persist",
        })
        return nil, err
    }
    s.logTunnelEvent(ctx, &tunnel.ID, domain.EventTypeTunnelDispatch, domain.EventStatusPending, "Commands queued for Agents", map[string]interface{}{
        "source_node_id": input.SourceNodeID,
        "dest_node_id":   input.DestNodeID,
    })
    tunnel.Status = domain.TunnelStatusActive
    if err := s.tunnelRepo.Update(ctx, tunnel); err != nil {
        s.logger.Errorw("failed to update tunnel status", "id", tunnel.ID, "error", err)
    }
    s.logTunnelEvent(ctx, &tunnel.ID, domain.EventTypeTunnelReady, domain.EventStatusSuccess, "Tunnel is active", map[string]interface{}{
        "source_node_id": input.SourceNodeID,
        "dest_node_id":   input.DestNodeID,
    })
    s.logger.Infow("tunnel_create_step", "step", "persist", "duration_ms", time.Since(step).Milliseconds(), "elapsed_ms", time.Since(start).Milliseconds())
    s.logger.Infow("tunnel_create_done", "tunnel_id", tunnel.ID, "total_ms", time.Since(start).Milliseconds())

    return tunnel, nil
}

func (s *tunnelService) CreateChain(ctx context.Context, entryID, relayID, exitID uint, protocol domain.TunnelProtocol) (*domain.Tunnel, error) {
    s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelInit, domain.EventStatusPending, "Initializing Multi-Hop Tunnel", map[string]interface{}{
        "entry_id": entryID,
        "relay_id": relayID,
        "exit_id":  exitID,
        "protocol": protocol,
        "topology": "chain",
    })
    // Validate inputs
    if entryID == relayID || relayID == exitID || entryID == exitID {
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Invalid chain: duplicate nodes", map[string]interface{}{
            "entry_id": entryID,
            "relay_id": relayID,
            "exit_id":  exitID,
            "step":     "validation",
        })
        return nil, ErrTunnelSameNode
    }

    unlock := s.lockKeys(
        fmt.Sprintf("node:%d", entryID),
        fmt.Sprintf("node:%d", relayID),
        fmt.Sprintf("node:%d", exitID),
        fmt.Sprintf("tunnel:%d:%d", entryID, relayID),
        fmt.Sprintf("tunnel:%d:%d", relayID, exitID),
    )
    defer unlock()

    entryNode, err := s.nodeRepo.GetByID(ctx, entryID)
	if err != nil {
		return nil, ErrNodeNotFound
	}
	relayNode, err := s.nodeRepo.GetByID(ctx, relayID)
	if err != nil {
		return nil, ErrNodeNotFound
	}
	exitNode, err := s.nodeRepo.GetByID(ctx, exitID)
	if err != nil {
		return nil, ErrNodeNotFound
	}

    // Allocate IPs for Segment A (Entry -> Relay)
    ipv4A, ipv6A, err := s.ipam.AllocateTunnelIPs(ctx)
    if err != nil {
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "IPAM allocation failed (segment A)", map[string]interface{}{
            "error": err.Error(),
            "step":  "allocate_ips",
            "segment": "A",
        })
        return nil, err
    }

    // Allocate IPs for Segment B (Relay -> Exit)
    ipv4B, ipv6B, err := s.ipam.AllocateTunnelIPs(ctx)
    if err != nil {
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "IPAM allocation failed (segment B)", map[string]interface{}{
            "error": err.Error(),
            "step":  "allocate_ips",
            "segment": "B",
        })
        return nil, err
    }
    s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelIPAM, domain.EventStatusPending, "Allocated IPs for chain tunnel", map[string]interface{}{
        "ipv4_a": ipv4A,
        "ipv6_a": ipv6A,
        "ipv4_b": ipv4B,
        "ipv6_b": ipv6B,
    })

	// Reserve Ports
	// Entry -> Relay (Relay listens)
    relayPortIn, err := s.portam.ReservePort(ctx, relayID, string(protocol))
    if err != nil {
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Port reservation failed (relay)", map[string]interface{}{
            "node_id": relayID,
            "error":   err.Error(),
            "step":    "reserve_ports",
        })
        return nil, err
    }
	
	// Relay -> Exit (Exit listens)
    exitPort, err := s.portam.ReservePort(ctx, exitID, string(protocol))
    if err != nil {
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Port reservation failed (exit)", map[string]interface{}{
            "node_id": exitID,
            "error":   err.Error(),
            "step":    "reserve_ports",
        })
        return nil, err
    }

    // Derive WG host IPs per segment
    entryA, relayA, err := derivePair(ipv4A)
    if err != nil {
        return nil, err
    }
    relayB, exitB, err := derivePair(ipv4B)
    if err != nil {
        return nil, err
    }

    chainParams := factory.ChainConfigParams{
        Protocol: string(protocol),
    }
    chainParams.SegmentA.EntryIP = entryA
    chainParams.SegmentA.RelayIP = relayA
    chainParams.SegmentA.RelayPort = relayPortIn
    chainParams.SegmentA.RelayPublicIP = relayNode.IP
    chainParams.SegmentB.RelayIP = relayB
    chainParams.SegmentB.ExitIP = exitB
    chainParams.SegmentB.ExitPort = exitPort
    chainParams.SegmentB.ExitPublicIP = exitNode.IP

    chainConfig, err := s.factory.GenerateChainConfig(chainParams)
    if err != nil {
        s.logger.Errorw("failed to generate chain config", "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "ProtocolFactory chain config generation failed", map[string]interface{}{
            "error": err.Error(),
            "step":  "generate_config",
        })
        return nil, err
    }
    s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelConfig, domain.EventStatusPending, "Configs generated via ProtocolFactory", map[string]interface{}{
        "protocol": protocol,
    })

	// Construct Segments Data
    segments := domain.JSONB{
        "segment_a": map[string]interface{}{
            "source_id": entryID,
            "dest_id":   relayID,
            "ipv4":      ipv4A,
            "ipv6":      ipv6A,
            "dest_port": relayPortIn,
            "entry_ip":  entryA,
            "relay_ip":  relayA,
        },
        "segment_b": map[string]interface{}{
            "source_id": relayID,
            "dest_id":   exitID,
            "ipv4":      ipv4B,
            "ipv6":      ipv6B,
            "dest_port": exitPort,
            "relay_ip":  relayB,
            "exit_ip":   exitB,
        },
    }

	hops := domain.JSONB{
		"nodes": []uint{entryID, relayID, exitID},
	}

    tunnel := &domain.Tunnel{
        Name:         "Chain: " + entryNode.Name + " -> " + relayNode.Name + " -> " + exitNode.Name,
        Protocol:     protocol,
        SourceNodeID: entryID,
        DestNodeID:   exitID,
        SourcePort:   0, // N/A for chain master record
        DestPort:     exitPort,
        Status:       domain.TunnelStatusPending,
        Type:         domain.TunnelTypeChain,
        Hops:         hops,
        Segments:     segments,
        Config:       domain.JSONB{
            "entry_config": chainConfig.EntryConfig,
            "relay_config": chainConfig.RelayConfig,
            "exit_config":  chainConfig.ExitConfig,
            "metadata":     chainConfig.Metadata,
        },
        Nodes:        domain.JSONB{"nodes": []uint{entryID, relayID, exitID}},
    }

    if err := s.tunnelRepo.Create(ctx, tunnel); err != nil {
        s.logger.Errorw("failed to create chain tunnel", "error", err)
        s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Persist chain tunnel failed", map[string]interface{}{
            "error": err.Error(),
            "step":  "persist",
        })
        return nil, err
    }

    s.logTunnelEvent(ctx, &tunnel.ID, domain.EventTypeTunnelDispatch, domain.EventStatusPending, "Commands queued for Agents", map[string]interface{}{
        "entry_id": entryID,
        "relay_id": relayID,
        "exit_id":  exitID,
    })
    tunnel.Status = domain.TunnelStatusActive
    if err := s.tunnelRepo.Update(ctx, tunnel); err != nil {
        s.logger.Errorw("failed to update chain tunnel status", "id", tunnel.ID, "error", err)
    }
    s.logTunnelEvent(ctx, &tunnel.ID, domain.EventTypeTunnelReady, domain.EventStatusSuccess, "Tunnel is active", map[string]interface{}{
        "entry_id": entryID,
        "relay_id": relayID,
        "exit_id":  exitID,
    })

	return tunnel, nil
}

func (s *tunnelService) GetTunnels(ctx context.Context) ([]domain.Tunnel, error) {
	return s.tunnelRepo.GetAll(ctx)
}

func (s *tunnelService) GetTunnelByID(ctx context.Context, id uint) (*domain.Tunnel, error) {
	return s.tunnelRepo.GetByID(ctx, id)
}

func (s *tunnelService) DeleteTunnel(ctx context.Context, id uint) error {
    // Get tunnel to release resources
    tunnel, err := s.tunnelRepo.GetByID(ctx, id)
    if err != nil {
        return err
    }

    unlock := s.lockKeys(
        fmt.Sprintf("tunnel:%d:%d", tunnel.SourceNodeID, tunnel.DestNodeID),
        fmt.Sprintf("node:%d", tunnel.SourceNodeID),
        fmt.Sprintf("node:%d", tunnel.DestNodeID),
    )
    defer unlock()

	// Release IPs
	if err := s.ipam.ReleaseIPs(ctx, tunnel.InternalIPv4, tunnel.InternalIPv6); err != nil {
		s.logger.Warnw("failed to release ips", "error", err)
	}

	// Release Ports
	if err := s.portam.ReleasePort(ctx, tunnel.SourceNodeID, tunnel.SourcePort, string(tunnel.Protocol)); err != nil {
		s.logger.Warnw("failed to release source port", "error", err)
	}
	if err := s.portam.ReleasePort(ctx, tunnel.DestNodeID, tunnel.DestPort, string(tunnel.Protocol)); err != nil {
		s.logger.Warnw("failed to release dest port", "error", err)
	}

    return s.tunnelRepo.Delete(ctx, id)
}

// Helpers
func deriveWGIPs(cidr string) (string, string, error) {
    ip, ipnet, err := net.ParseCIDR(cidr)
    if err != nil {
        return "", "", err
    }
    base := ip.Mask(ipnet.Mask)
    baseInt := ipToUint32(base)
    host1 := uint32ToIP(baseInt + 1)
    host2 := uint32ToIP(baseInt + 2)
    return host1.String() + "/30", host2.String() + "/30", nil
}

func derivePair(cidr string) (string, string, error) {
    server, client, err := deriveWGIPs(cidr)
    if err != nil {
        return "", "", err
    }
    // return client first (Entry/Relay client), then server (Relay/Exit server)
    return client, server, nil
}

func (s *tunnelService) logTunnelEvent(ctx context.Context, tunnelID *uint, etype string, status domain.EventStatus, msg string, meta map[string]interface{}) {
    if s.timelineRepo == nil {
        return
    }
    var metadata domain.JSONB
    if meta != nil {
        b, _ := json.Marshal(meta)
        _ = json.Unmarshal(b, &metadata)
    }
    if v := ctx.Value("request_id"); v != nil {
        metadata["request_id"] = v
    }
    event := &domain.TimelineEvent{
        Type:         etype,
        Status:       status,
        Message:      msg,
        ResourceType: "tunnel",
        ResourceID:   tunnelID,
        Meta:         metadata,
        CreatedAt:    time.Now(),
    }
    _ = s.timelineRepo.Create(ctx, event)
}

func (s *tunnelService) lockKeys(keys ...string) func() {
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
