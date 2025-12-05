package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var defaultConfigPaths = []string{
	"./agent.yaml",
	"/etc/netly/agent.yaml",
}

type Config struct {
	BackendURL        string        `yaml:"backend_url"`
	NodeToken         string        `yaml:"node_token"`
	LogPath           string        `yaml:"log_path"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

func Load(path string) (*Config, error) {
	var configPath string

	if path != "" {
		configPath = path
	} else {
		for _, p := range defaultConfigPaths {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				break
			}
		}
	}

	if configPath == "" {
		return nil, fmt.Errorf("config file not found in default paths")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 10 * time.Second
	}
	if cfg.LogPath == "" {
		cfg.LogPath = "/var/log/netly-agent.log"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.BackendURL == "" {
		return fmt.Errorf("backend_url is required")
	}
	if c.NodeToken == "" {
		return fmt.Errorf("node_token is required")
	}
	return nil
}
