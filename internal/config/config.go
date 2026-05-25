package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config is the merged on-disk + environment configuration.
type Config struct {
	Server ServerConfig `yaml:"server"`
	FMP    FMPConfig    `yaml:"fmp"`

	// APIKey is loaded from the FMP_API_KEY environment variable (typically via .env).
	APIKey string `yaml:"-"`
}

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	MCPPath string `yaml:"mcp_path"`
}

type FMPConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIPath        string `yaml:"api_path"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	UserAgent      string `yaml:"user_agent"`
}

// Load reads config.yaml from path (defaulting to ./config.yaml when empty),
// loads .env from the current directory if present, and pulls FMP_API_KEY
// out of the environment.
func Load(path string) (*Config, error) {
	if path == "" {
		path = "config.yaml"
	}

	// Best-effort .env load. Missing file is not an error so production
	// deployments can rely on real environment variables.
	_ = godotenv.Load()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Defaults.
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8086
	}
	if cfg.Server.MCPPath == "" {
		cfg.Server.MCPPath = "/mcp"
	}
	if cfg.FMP.BaseURL == "" {
		cfg.FMP.BaseURL = "https://financialmodelingprep.com"
	}
	if cfg.FMP.APIPath == "" {
		cfg.FMP.APIPath = "/stable"
	}
	if cfg.FMP.TimeoutSeconds == 0 {
		cfg.FMP.TimeoutSeconds = 30
	}
	if cfg.FMP.UserAgent == "" {
		cfg.FMP.UserAgent = "golang-local-fmp-mcp/0.1"
	}

	cfg.APIKey = strings.TrimSpace(os.Getenv("FMP_API_KEY"))
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("FMP_API_KEY not set (put it in .env or export it)")
	}

	return cfg, nil
}

// Addr returns "host:port" formatted for net.Listen / http.Server.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
