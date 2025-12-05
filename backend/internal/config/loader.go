package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Logger   LoggerConfig   `mapstructure:"logger"`
    Security SecurityConfig `mapstructure:"security"`
    IPAM     IPAMConfig     `mapstructure:"ipam"`
    PortAM   PortAMConfig   `mapstructure:"portam"`
    Features FeaturesConfig `mapstructure:"features"`
    Auth     AuthConfig     `mapstructure:"auth"`
}

type IPAMConfig struct {
	IPv4CIDR string `mapstructure:"ipv4_cidr"`
	IPv6CIDR string `mapstructure:"ipv6_cidr"`
}

type PortAMConfig struct {
	MinPort int `mapstructure:"min_port"`
	MaxPort int `mapstructure:"max_port"`
}

type SecurityConfig struct {
	EncryptionKey string `mapstructure:"encryption_key"`
	GeoIPToken    string `mapstructure:"geoip_token"`
	PublicURL     string `mapstructure:"public_url"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type LoggerConfig struct {
    Level            string   `mapstructure:"level"`
    Encoding         string   `mapstructure:"encoding"`
    OutputPaths      []string `mapstructure:"output_paths"`
    ErrorOutputPaths []string `mapstructure:"error_output_paths"`
}

type FeaturesConfig struct {
    EnableLocks           bool   `mapstructure:"enable_locks"`
    RequestIDHeader       string `mapstructure:"request_id_header"`
    EnableTaskCorrelation bool   `mapstructure:"enable_task_correlation"`
    EnableRequestLogging  bool   `mapstructure:"enable_request_logging"`
}

type AuthConfig struct {
    AdminAPIKey    string   `mapstructure:"admin_api_key"`
    AgentToken     string   `mapstructure:"agent_token"`
    AllowedOrigins []string `mapstructure:"allowed_origins"`
}


func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetEnvPrefix("NETLY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
