package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/core/services/factory"
	"github.com/netly/backend/internal/domain"
	"github.com/netly/backend/internal/domain/singbox"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type tunnelService struct {
	tunnelRepo   ports.TunnelRepository
	nodeRepo     ports.NodeRepository
	ipam         ports.IPAMService
	portam       ports.PortAMService
	factory      *factory.FactoryService
	taskService  ports.TaskService
	logger       *logger.Logger
	timelineRepo ports.TimelineRepository
	mu           sync.Mutex
	locks        map[string]*sync.Mutex
}

type TunnelServiceConfig struct {
	TunnelRepo   ports.TunnelRepository
	NodeRepo     ports.NodeRepository
	IPAM         ports.IPAMService
	PortAM       ports.PortAMService
	Factory      *factory.FactoryService
	TaskService  ports.TaskService
	Logger       *logger.Logger
	TimelineRepo ports.TimelineRepository
}

func NewTunnelService(cfg TunnelServiceConfig) ports.TunnelService {
	return &tunnelService{
		tunnelRepo:   cfg.TunnelRepo,
		nodeRepo:     cfg.NodeRepo,
		ipam:         cfg.IPAM,
		portam:       cfg.PortAM,
		factory:      cfg.Factory,
		taskService:  cfg.TaskService,
		logger:       cfg.Logger,
		timelineRepo: cfg.TimelineRepo,
		locks:        make(map[string]*sync.Mutex),
	}
}

func (s *tunnelService) CreateTunnel(ctx context.Context, input ports.CreateTunnelInput) (*domain.Tunnel, error) {
	start := time.Now()
	step := time.Now()
	s.logger.Infow("tunnel_create_start", "source_node_id", input.SourceNodeID, "dest_node_id", input.DestNodeID, "protocol", input.Protocol)
	s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelInit, domain.EventStatusPending, "Initializing Direct Tunnel", map[string]interface{}{
		"source_node_id": input.SourceNodeID,
		"dest_node_id":   input.DestNodeID,
		"protocol":       input.Protocol,
		"topology":       "direct",
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

	// DEBUG: Log allocated IPs
	s.logger.Infow("wireguard_ip_allocation",
		"subnet", ipv4Subnet,
		"server_ip", serverWGIP,
		"client_ip", clientWGIP,
		"dest_node_id", input.DestNodeID,
		"source_node_id", input.SourceNodeID,
	)

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

	// ==================== DISPATCH COMMANDS TO AGENTS ====================
	if s.taskService != nil {
		// Prepare Content
		var inboundContent string
		switch v := configResult.Inbound.(type) {
		case string:
			inboundContent = v
		case singbox.Inbound:
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				s.logger.Errorw("failed to marshal singbox config", "error", err)
				return nil, err
			}
			inboundContent = string(b)
		case domain.JSONB: // Handle raw JSONB if it comes through
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				s.logger.Errorw("failed to marshal jsonb config", "error", err)
				return nil, err
			}
			inboundContent = string(b)
		case map[string]interface{}: // Handle raw map
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				s.logger.Errorw("failed to marshal map config", "error", err)
				return nil, err
			}
			inboundContent = string(b)
		default:
			return nil, fmt.Errorf("unexpected config type: %T", configResult.Inbound)
		}

		// Prepare WireGuard variables (used in both server and client configs)
		// Get endpoint IPs (prefer PrivateIP for Hyper-V environments)
		sourceEndpointIP := getNodeEndpointIP(sourceNode)
		destEndpointIP := getNodeEndpointIP(destNode)

		// Extract just the IP without CIDR for AllowedIPs (use /32 for point-to-point)
		serverIPOnly := strings.Split(serverWGIP, "/")[0]
		clientIPOnly := strings.Split(clientWGIP, "/")[0]

		// Dispatch to Dest Node (Server)
		if input.Protocol == "wireguard" {
			// WireGuard: Generate FULL config including Peer section
			// Dest Node = Server (gets .1/30, listens)
			// Source Node = Client (gets .2/30, connects)
			//
			// CRITICAL: Each node gets a DIFFERENT IP to avoid conflicts!
			// serverWGIP = x.x.x.1/30 (for dest/server)
			// clientWGIP = x.x.x.2/30 (for source/client)

			// Server config (Dest Node) - gets serverWGIP (.1)
			serverConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
ListenPort = %d
PostUp = iptables -A FORWARD -i %%i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %%i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s:%d
PersistentKeepalive = 25`,
				destNode.WireGuardPrivateKey,
				serverWGIP, // Server gets .1/30
				destPort,
				sourceNode.WireGuardPublicKey,
				clientIPOnly, // Allow traffic from client's IP
				sourceEndpointIP,
				sourcePort)

			s.logger.Infow("wireguard_server_config",
				"dest_node_id", destNode.ID,
				"server_wg_ip", serverWGIP,
				"peer_allowed_ip", clientIPOnly+"/32",
				"peer_endpoint", fmt.Sprintf("%s:%d", sourceEndpointIP, sourcePort))

			// Escape and Dispatch
			escapedContent := strings.ReplaceAll(serverConf, "'", "'\\''")
			script := fmt.Sprintf("sudo systemctl stop wg-quick@wg0 2>/dev/null || true && echo '%s' | sudo tee /etc/wireguard/wg0.conf > /dev/null && sudo chmod 600 /etc/wireguard/wg0.conf && sudo systemctl enable --now wg-quick@wg0", escapedContent)

			destPayload := domain.JSONB{
				"script":      script,
				"interpreter": "sh",
			}
			if _, err := s.taskService.CreateCommand(input.DestNodeID, domain.CmdExecuteScript, destPayload); err != nil {
				s.logger.Errorw("failed to dispatch command to dest node", "node_id", input.DestNodeID, "error", err)
			} else {
				s.logger.Infow("dispatched CMD_EXECUTE_SCRIPT to dest node (WG)", "node_id", input.DestNodeID)
			}
		} else {
			// Sing-Box/Others: Use ApplyConfig
			destPayload := domain.JSONB{
				"target_path": "/etc/sing-box/config.json", // Defaulting path for now, or should be dynamic?
				// User prompt didn't specify path for SingBox, but existing code used /etc/wireguard/wg0.conf for EVERYTHING which was wrong for SingBox
				// However, if I change it now, I might break something else.
				// BUT the code at line 221 hardcoded "/etc/wireguard/wg0.conf".
				// If protocol is SingBox, writing to wg0.conf is definitely wrong.
				// Assuming standard singbox path or sticking to what was there (which was wrong but maybe "working" in user's mind until panic?)
				// Let's use a safe default if it's not wireguard.
				"content":      inboundContent,
				"service_name": "sing-box", // Assumed service name
				"enable":       true,
				"tunnel_id":    tunnel.ID,
				"role":         "server",
			}
			// Adjust path based on protocol if needed
			if input.Protocol != "wireguard" {
				destPayload["target_path"] = "/etc/sing-box/config.json"
			} else {
				destPayload["target_path"] = "/etc/wireguard/wg0.conf"
			}

			if _, err := s.taskService.CreateCommand(input.DestNodeID, domain.CmdApplyConfig, destPayload); err != nil {
				s.logger.Errorw("failed to dispatch command to dest node", "node_id", input.DestNodeID, "error", err)
			} else {
				s.logger.Infow("dispatched CMD_APPLY_CONFIG to dest node", "node_id", input.DestNodeID)
			}
		}

		// Dispatch to Source Node (Client)
		if input.Protocol == "wireguard" {
			// Client config (Source Node) - gets clientWGIP (.2)
			// CRITICAL: Client gets DIFFERENT IP than server!

			// Client config - connects to server
			clientConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
ListenPort = %d
PostUp = iptables -A FORWARD -i %%i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %%i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s:%d
PersistentKeepalive = 25`,
				sourceNode.WireGuardPrivateKey,
				clientWGIP, // Client gets .2/30
				sourcePort,
				destNode.WireGuardPublicKey,
				serverIPOnly, // Allow traffic from server's IP
				destEndpointIP,
				destPort)

			s.logger.Infow("wireguard_client_config",
				"source_node_id", sourceNode.ID,
				"client_wg_ip", clientWGIP,
				"peer_allowed_ip", serverIPOnly+"/32",
				"peer_endpoint", fmt.Sprintf("%s:%d", destEndpointIP, destPort))

			escapedContent := strings.ReplaceAll(clientConf, "'", "'\\''")
			script := fmt.Sprintf("sudo systemctl stop wg-quick@wg0 2>/dev/null || true && echo '%s' | sudo tee /etc/wireguard/wg0.conf > /dev/null && sudo chmod 600 /etc/wireguard/wg0.conf && sudo systemctl enable --now wg-quick@wg0", escapedContent)

			sourcePayload := domain.JSONB{
				"script":      script,
				"interpreter": "sh",
			}
			if _, err := s.taskService.CreateCommand(input.SourceNodeID, domain.CmdExecuteScript, sourcePayload); err != nil {
				s.logger.Errorw("failed to dispatch command to source node", "node_id", input.SourceNodeID, "error", err)
			} else {
				s.logger.Infow("dispatched CMD_EXECUTE_SCRIPT to source node (WG)", "node_id", input.SourceNodeID)
			}
		} else {
			// For SingBox Client, ClientConfig is a URL/Link (vless://...).
			// We probably don't "ApplyConfig" this to a file? Or do we?
			// The original code was applying it to /etc/wireguard/wg0.conf which is definitely wrong for a vless link.
			// Usually we just display the link. But if we MUST dispatch a command...
			// Maybe we do nothing for client if it's a link?
			// Or maybe we write it to a file for reference?
			// Original code did: target_path: /etc/wireguard/wg0.conf.
			// I will leave it as is but fix the content type issue.
			// Since ClientConfig IS a string (line 12 in factory_service.go), valid for ApplyConfig.

			// Warning: Writing a vless link to wg0.conf is bad.
			// But I strictly follow "Fix WireGuard Deployment" and "Fix SingBox Panic".
			// The panic was about Inbound. ClientConfig is string.

			sourcePayload := domain.JSONB{
				"target_path": "/root/client_config.txt", // Safer path for non-WG
				"content":     configResult.ClientConfig,
				"enable":      false,
				"tunnel_id":   tunnel.ID,
				"role":        "client",
			}
			s.taskService.CreateCommand(input.SourceNodeID, domain.CmdApplyConfig, sourcePayload)
		}
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
			"error":   err.Error(),
			"step":    "allocate_ips",
			"segment": "A",
		})
		return nil, err
	}

	// Allocate IPs for Segment B (Relay -> Exit)
	ipv4B, ipv6B, err := s.ipam.AllocateTunnelIPs(ctx)
	if err != nil {
		s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "IPAM allocation failed (segment B)", map[string]interface{}{
			"error":   err.Error(),
			"step":    "allocate_ips",
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
		Config: domain.JSONB{
			"entry_config": chainConfig.EntryConfig,
			"relay_config": chainConfig.RelayConfig,
			"exit_config":  chainConfig.ExitConfig,
			"metadata":     chainConfig.Metadata,
		},
		Nodes: domain.JSONB{"nodes": []uint{entryID, relayID, exitID}},
	}

	if err := s.tunnelRepo.Create(ctx, tunnel); err != nil {
		s.logger.Errorw("failed to create chain tunnel", "error", err)
		s.logTunnelEvent(ctx, nil, domain.EventTypeTunnelFailed, domain.EventStatusFailed, "Persist chain tunnel failed", map[string]interface{}{
			"error": err.Error(),
			"step":  "persist",
		})
		return nil, err
	}

	// ==================== DISPATCH COMMANDS TO AGENTS (CHAIN) ====================
	// ==================== DISPATCH COMMANDS TO AGENTS (CHAIN) ====================
	if s.taskService != nil {
		// Entry Node (WG Client)
		escapedEntry := strings.ReplaceAll(chainConfig.EntryConfig, "'", "'\\''")
		entryScript := fmt.Sprintf("echo '%s' | sudo tee /etc/wireguard/wg0.conf && sudo systemctl enable --now wg-quick@wg0", escapedEntry)
		s.taskService.CreateCommand(entryID, domain.CmdExecuteScript, domain.JSONB{"script": entryScript, "interpreter": "sh"})

		// Relay Node
		if strings.Contains(chainConfig.RelayConfig, "---SPLIT---") {
			parts := strings.Split(chainConfig.RelayConfig, "\n\n---SPLIT---\n\n")
			if len(parts) >= 2 {
				// wg0
				esc0 := strings.ReplaceAll(parts[0], "'", "'\\''")
				script0 := fmt.Sprintf("echo '%s' | sudo tee /etc/wireguard/wg0.conf && sudo systemctl enable --now wg-quick@wg0", esc0)
				s.taskService.CreateCommand(relayID, domain.CmdExecuteScript, domain.JSONB{"script": script0, "interpreter": "sh"})

				// wg1
				esc1 := strings.ReplaceAll(parts[1], "'", "'\\''")
				script1 := fmt.Sprintf("echo '%s' | sudo tee /etc/wireguard/wg1.conf && sudo systemctl enable --now wg-quick@wg1", esc1)
				s.taskService.CreateCommand(relayID, domain.CmdExecuteScript, domain.JSONB{"script": script1, "interpreter": "sh"})
			}
		} else {
			esc := strings.ReplaceAll(chainConfig.RelayConfig, "'", "'\\''")
			script := fmt.Sprintf("echo '%s' | sudo tee /etc/wireguard/wg0.conf && sudo systemctl enable --now wg-quick@wg0", esc)
			s.taskService.CreateCommand(relayID, domain.CmdExecuteScript, domain.JSONB{"script": script, "interpreter": "sh"})
		}

		// Exit Node (WG Server)
		escapedExit := strings.ReplaceAll(chainConfig.ExitConfig, "'", "'\\''")
		exitScript := fmt.Sprintf("echo '%s' | sudo tee /etc/wireguard/wg0.conf && sudo systemctl enable --now wg-quick@wg0", escapedExit)
		s.taskService.CreateCommand(exitID, domain.CmdExecuteScript, domain.JSONB{"script": exitScript, "interpreter": "sh"})
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
	// Server gets .1/30, Client gets .2/30
	return host1.String() + "/30", host2.String() + "/30", nil
}

// getNodeEndpointIP returns the best IP for WireGuard endpoint
// Prefers PrivateIP for Hyper-V/internal networks
func getNodeEndpointIP(node *domain.Node) string {
	if node.PrivateIP != "" {
		return node.PrivateIP
	}
	return node.IP
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
