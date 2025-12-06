package factory

import (
	"fmt"

	"github.com/netly/backend/internal/domain/singbox"
	"github.com/netly/backend/pkg/utils/keygen"
)

type ConfigResult struct {
	Inbound      interface{}       `json:"inbound"`       // Sing-box Inbound struct or String for WG
	ClientConfig string            `json:"client_config"` // Link or Conf
	Metadata     map[string]string `json:"metadata"`      // Extra info (keys, etc)
}

type FactoryService struct{}

func NewFactoryService() *FactoryService {
	return &FactoryService{}
}

type ConfigParams struct {
	Protocol   string
	Port       int
	ServerIP   string
	SNI        string // Optional, defaults to yahoo.com or bing.com
	ClientIP   string // Required for WireGuard (e.g. 10.10.0.2/32)
	ServerWGIP string // Required for WireGuard (e.g. 10.10.0.1/24)
}

type ChainConfigParams struct {
	Protocol string
	// Entry -> Relay
	SegmentA struct {
		EntryIP       string // Client IP for A (e.g. 10.10.0.2/32)
		RelayIP       string // Server IP for A (e.g. 10.10.0.1/24) - WG Interface IP
		RelayPort     int    // Port Relay listens on for Entry
		RelayPublicIP string
	}
	// Relay -> Exit
	SegmentB struct {
		RelayIP      string // Client IP for B (e.g. 10.10.1.2/32) - WG Interface IP
		ExitIP       string // Server IP for B (e.g. 10.10.1.1/24) - WG Interface IP
		ExitPort     int    // Port Exit listens on for Relay
		ExitPublicIP string
	}
}

type ChainConfigResult struct {
	EntryConfig string            `json:"entry_config"`
	RelayConfig string            `json:"relay_config"`
	ExitConfig  string            `json:"exit_config"`
	Metadata    map[string]string `json:"metadata"`
}

func (s *FactoryService) GenerateChainConfig(params ChainConfigParams) (*ChainConfigResult, error) {
	// Handle "Smart Auto" by defaulting to WireGuard
	if params.Protocol == "Smart Auto" {
		params.Protocol = "wireguard"
	}

	if params.Protocol == "wireguard" {
		return s.generateWireGuardChain(params)
	}
	return nil, fmt.Errorf("unsupported protocol for chain: %s", params.Protocol)
}

func (s *FactoryService) generateWireGuardChain(params ChainConfigParams) (*ChainConfigResult, error) {
	// Keys for Segment A (Entry <-> Relay)
	entryPriv, entryPub, _ := keygen.GenerateWireGuardKeys()
	relayAPriv, relayAPub, _ := keygen.GenerateWireGuardKeys()

	// Keys for Segment B (Relay <-> Exit)
	relayBPriv, relayBPub, _ := keygen.GenerateWireGuardKeys()
	exitPriv, exitPub, _ := keygen.GenerateWireGuardKeys()

	// 1. Entry Config (Client)
	// Connects to Relay (Segment A)
	entryConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 1.1.1.1

[Peer]
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25`,
		entryPriv,
		params.SegmentA.EntryIP,
		relayAPub,
		params.SegmentA.RelayPublicIP,
		params.SegmentA.RelayPort)

	// 2. Relay Config (Middleman)
	// Interface A (Incoming from Entry) - wg0
	relayConfigA := fmt.Sprintf(`[Interface]
# Segment A (Listener)
PrivateKey = %s
ListenPort = %d
Address = %s
PostUp = sysctl -w net.ipv4.ip_forward=1; sysctl -w net.ipv6.conf.all.forwarding=1; iptables -A FORWARD -i wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT
Table = off

[Peer]
# Entry Node
PublicKey = %s
AllowedIPs = %s`,
		relayAPriv,
		params.SegmentA.RelayPort,
		params.SegmentA.RelayIP,
		entryPub,
		params.SegmentA.EntryIP)

	// Interface B (Outgoing to Exit) - wg1
	relayConfigB := fmt.Sprintf(`[Interface]
# Segment B (Client)
PrivateKey = %s
Address = %s
Table = off
PostUp = ip rule add from %s table 200; ip route add default dev wg1 table 200
PostDown = ip rule del from %s table 200

[Peer]
# Exit Node
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25`,
		relayBPriv,
		params.SegmentB.RelayIP,
		params.SegmentA.RelayIP,
		params.SegmentA.RelayIP,
		exitPub,
		params.SegmentB.ExitPublicIP,
		params.SegmentB.ExitPort)

	relayConfig := relayConfigA + "\n\n---SPLIT---\n\n" + relayConfigB

	// 3. Exit Config (Server)
	// Listens for Relay (Segment B)
	exitConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = %d
Address = %s
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
# Relay Node
PublicKey = %s
AllowedIPs = %s`,
		exitPriv,
		params.SegmentB.ExitPort,
		params.SegmentB.ExitIP,
		relayBPub,
		params.SegmentB.RelayIP)

	return &ChainConfigResult{
		EntryConfig: entryConfig,
		RelayConfig: relayConfig,
		ExitConfig:  exitConfig,
		Metadata: map[string]string{
			"entry_pub":   entryPub,
			"relay_a_pub": relayAPub,
			"relay_b_pub": relayBPub,
			"exit_pub":    exitPub,
		},
	}, nil
}

func (s *FactoryService) GenerateConfig(params ConfigParams) (*ConfigResult, error) {
	if params.SNI == "" {
		params.SNI = "yahoo.com"
	}

	// Handle "Smart Auto" by defaulting to WireGuard
	if params.Protocol == "Smart Auto" {
		params.Protocol = "wireguard"
	}

	switch params.Protocol {
	case "vless", "vless_reality":
		return s.generateVLESSReality(params)
	case "wireguard":
		return s.generateWireGuard(params)
	case "hysteria2":
		return s.generateHysteria2(params)
	case "tuic":
		return s.generateTUIC(params)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", params.Protocol)
	}
}

func (s *FactoryService) generateVLESSReality(params ConfigParams) (*ConfigResult, error) {
	// 1. Generate Keys
	privKey, pubKey, err := keygen.GenerateX25519Keys()
	if err != nil {
		return nil, err
	}
	shortId := keygen.GenerateShortId()
	uuid := keygen.GenerateUUID()

	// 2. Construct Inbound
	inbound := singbox.Inbound{
		Type:       "vless",
		Tag:        "vless-reality-in",
		Listen:     "::",
		ListenPort: params.Port,
		Sniff:      true,
		Users: []singbox.User{
			{
				Name: "default-user",
				UUID: uuid,
				Flow: "xtls-rprx-vision",
			},
		},
		TLS: &singbox.TLSConfig{
			Enabled:    true,
			ServerName: params.SNI,
			Reality: &singbox.RealityConfig{
				Enabled: true,
				Handshake: &singbox.Handshake{
					Server:     params.SNI,
					ServerPort: 443,
				},
				PrivateKey: privKey,
				ShortID:    []string{shortId},
			},
		},
	}

	// 3. Generate Client Link
	link := fmt.Sprintf("vless://%s@%s:%d?security=reality&encryption=none&pbk=%s&fp=chrome&type=tcp&flow=xtls-rprx-vision&sni=%s&sid=%s#Netly-%s",
		uuid, params.ServerIP, params.Port, pubKey, params.SNI, shortId, params.ServerIP)

	return &ConfigResult{
		Inbound:      inbound,
		ClientConfig: link,
		Metadata: map[string]string{
			"private_key": privKey,
			"public_key":  pubKey,
			"short_id":    shortId,
			"uuid":        uuid,
			"sni":         params.SNI,
		},
	}, nil
}

func (s *FactoryService) generateWireGuard(params ConfigParams) (*ConfigResult, error) {
	if params.ClientIP == "" || params.ServerWGIP == "" {
		return nil, fmt.Errorf("wireguard requires ClientIP and ServerWGIP")
	}

	serverPriv, serverPub, err := keygen.GenerateWireGuardKeys()
	if err != nil {
		return nil, err
	}
	clientPriv, clientPub, err := keygen.GenerateWireGuardKeys()
	if err != nil {
		return nil, err
	}

	// Extract IP only for AllowedIPs
	// 10.10.0.2/32 -> 10.10.0.2/32

	serverConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = %d
Address = %s
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
PublicKey = %s
AllowedIPs = %s`, serverPriv, params.Port, params.ServerWGIP, clientPub, params.ClientIP)

	clientConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 1.1.1.1

[Peer]
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25`, clientPriv, params.ClientIP, serverPub, params.ServerIP, params.Port)

	return &ConfigResult{
		Inbound:      serverConfig,
		ClientConfig: clientConfig,
		Metadata: map[string]string{
			"server_priv":  serverPriv,
			"server_pub":   serverPub,
			"client_priv":  clientPriv,
			"client_pub":   clientPub,
			"client_ip":    params.ClientIP,
			"server_wg_ip": params.ServerWGIP,
		},
	}, nil
}

func (s *FactoryService) generateHysteria2(params ConfigParams) (*ConfigResult, error) {
	password := keygen.GenerateRandomPassword(16)

	inbound := singbox.Inbound{
		Type:       "hysteria2",
		Tag:        "hy2-in",
		Listen:     "::",
		ListenPort: params.Port,
		Users: []singbox.User{
			{
				Name:     "default-user",
				Password: password,
			},
		},
		TLS: &singbox.TLSConfig{
			Enabled:    true,
			ServerName: params.SNI,
			ALPN:       []string{"h3"},
		},
		UpMbps:   100,
		DownMbps: 100,
	}

	link := fmt.Sprintf("hysteria2://%s@%s:%d?sni=%s&alpn=h3&insecure=1#Netly-Hy2", password, params.ServerIP, params.Port, params.SNI)

	return &ConfigResult{
		Inbound:      inbound,
		ClientConfig: link,
		Metadata: map[string]string{
			"password": password,
			"sni":      params.SNI,
		},
	}, nil
}

func (s *FactoryService) generateTUIC(params ConfigParams) (*ConfigResult, error) {
	uuid := keygen.GenerateUUID()
	password := keygen.GenerateRandomPassword(16)

	inbound := singbox.Inbound{
		Type:       "tuic",
		Tag:        "tuic-in",
		Listen:     "::",
		ListenPort: params.Port,
		Users: []singbox.User{
			{
				Name:     "default-user",
				UUID:     uuid,
				Password: password,
			},
		},
		TLS: &singbox.TLSConfig{
			Enabled:         true,
			ServerName:      params.SNI,
			ALPN:            []string{"h3"},
			CertificatePath: "/root/cert.crt",
			KeyPath:         "/root/private.key",
		},
		Multiplex: &singbox.MultiplexConfig{
			Enabled: true,
			Padding: true,
		},
	}

	link := fmt.Sprintf("tuic://%s:%s@%s:%d?sni=%s&alpn=h3&congestion_control=bbr#Netly-TUIC", uuid, password, params.ServerIP, params.Port, params.SNI)

	return &ConfigResult{
		Inbound:      inbound,
		ClientConfig: link,
		Metadata: map[string]string{
			"uuid":     uuid,
			"password": password,
			"sni":      params.SNI,
		},
	}, nil
}
