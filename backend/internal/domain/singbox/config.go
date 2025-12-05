package singbox

// Config represents the root Sing-box configuration
type Config struct {
	Log       *LogConfig    `json:"log,omitempty"`
	DNS       *DNSConfig    `json:"dns,omitempty"`
	Inbounds  []Inbound     `json:"inbounds,omitempty"`
	Outbounds []Outbound    `json:"outbounds,omitempty"`
	Route     *RouteConfig  `json:"route,omitempty"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level     string `json:"level,omitempty"`
	Timestamp bool   `json:"timestamp,omitempty"`
	Output    string `json:"output,omitempty"`
}

// DNSConfig represents DNS configuration
type DNSConfig struct {
	Servers []DNSServer `json:"servers,omitempty"`
}

type DNSServer struct {
	Tag     string `json:"tag,omitempty"`
	Address string `json:"address,omitempty"`
}

// RouteConfig represents routing configuration
type RouteConfig struct {
	Rules         []RouteRule `json:"rules,omitempty"`
	Final         string      `json:"final,omitempty"`
	AutoDetect    bool        `json:"auto_detect_interface,omitempty"`
}

type RouteRule struct {
	Inbound  []string `json:"inbound,omitempty"`
	Outbound string   `json:"outbound,omitempty"`
}

// Inbound represents an inbound configuration
type Inbound struct {
	Type           string          `json:"type"`
	Tag            string          `json:"tag,omitempty"`
	Listen         string          `json:"listen,omitempty"`
	ListenPort     int             `json:"listen_port,omitempty"`
	Users          []User          `json:"users,omitempty"`
	TLS            *TLSConfig      `json:"tls,omitempty"`
	Transport      *TransportConfig `json:"transport,omitempty"`
	Multiplex      *MultiplexConfig `json:"multiplex,omitempty"`
	
	// Hysteria2 specific
	UpMbps         int    `json:"up_mbps,omitempty"`
	DownMbps       int    `json:"down_mbps,omitempty"`
	Obfs           *Obfs  `json:"obfs,omitempty"`
	
	// General
	Sniff          bool   `json:"sniff,omitempty"`
}

// Outbound represents an outbound configuration
type Outbound struct {
	Type       string          `json:"type"`
	Tag        string          `json:"tag,omitempty"`
	Server     string          `json:"server,omitempty"`
	ServerPort int             `json:"server_port,omitempty"`
	UUID       string          `json:"uuid,omitempty"`
	Flow       string          `json:"flow,omitempty"`
	Password   string          `json:"password,omitempty"`
	TLS        *TLSConfig      `json:"tls,omitempty"`
	Transport  *TransportConfig `json:"transport,omitempty"`
	Multiplex  *MultiplexConfig `json:"multiplex,omitempty"`
	
	// Hysteria2 specific
	UpMbps     int    `json:"up_mbps,omitempty"`
	DownMbps   int    `json:"down_mbps,omitempty"`
	Obfs       *Obfs  `json:"obfs,omitempty"`
}


// User represents a user configuration
type User struct {
	Name     string `json:"name,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
	Flow     string `json:"flow,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled         bool           `json:"enabled,omitempty"`
	ServerName      string         `json:"server_name,omitempty"`
	Insecure        bool           `json:"insecure,omitempty"`
	ALPN            []string       `json:"alpn,omitempty"`
	MinVersion      string         `json:"min_version,omitempty"`
	MaxVersion      string         `json:"max_version,omitempty"`
	CertificatePath string         `json:"certificate_path,omitempty"`
	KeyPath         string         `json:"key_path,omitempty"`
	Reality         *RealityConfig `json:"reality,omitempty"`
	UTLS            *UTLSConfig    `json:"utls,omitempty"`
}

// RealityConfig represents Reality protocol configuration
type RealityConfig struct {
	Enabled     bool       `json:"enabled,omitempty"`
	Handshake   *Handshake `json:"handshake,omitempty"`
	PrivateKey  string     `json:"private_key,omitempty"`
	PublicKey   string     `json:"public_key,omitempty"`
	ShortID     []string   `json:"short_id,omitempty"`
	ServerName  string     `json:"server_name,omitempty"`
}

// Handshake represents Reality handshake configuration
type Handshake struct {
	Server     string `json:"server,omitempty"`
	ServerPort int    `json:"server_port,omitempty"`
}

// UTLSConfig represents uTLS fingerprint configuration
type UTLSConfig struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// TransportConfig represents transport layer configuration
type TransportConfig struct {
	Type        string            `json:"type,omitempty"`
	Host        string            `json:"host,omitempty"`
	Path        string            `json:"path,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ServiceName string            `json:"service_name,omitempty"`
}

// Obfs represents obfuscation configuration for Hysteria2
type Obfs struct {
	Type     string `json:"type,omitempty"`
	Password string `json:"password,omitempty"`
}

// MultiplexConfig represents multiplexing configuration
type MultiplexConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	Padding bool `json:"padding,omitempty"`
	Brutal  bool `json:"brutal,omitempty"`
}

// WireGuard specific structures

// WireGuardConfig represents a WireGuard configuration file
type WireGuardConfig struct {
	Interface WireGuardInterface `json:"interface"`
	Peers     []WireGuardPeer    `json:"peers"`
}

type WireGuardInterface struct {
	PrivateKey string   `json:"private_key"`
	Address    []string `json:"address"`
	ListenPort int      `json:"listen_port,omitempty"`
	MTU        int      `json:"mtu,omitempty"`
	DNS        []string `json:"dns,omitempty"`
}

type WireGuardPeer struct {
	PublicKey           string   `json:"public_key"`
	PresharedKey        string   `json:"preshared_key,omitempty"`
	Endpoint            string   `json:"endpoint,omitempty"`
	AllowedIPs          []string `json:"allowed_ips"`
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
}
