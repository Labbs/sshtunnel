package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "sshtunnel", "config.yaml")
}

// Load loads the configuration from the specified path
func Load(path string) (*Config, error) {
	v := viper.New()

	// Expand ~ in path
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot expand home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Set defaults
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand paths in config
	expandPaths(&cfg)

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Global defaults
	v.SetDefault("global.log_level", "info")
	v.SetDefault("global.log_file", "")

	// Reconnect defaults
	v.SetDefault("global.reconnect.enabled", true)
	v.SetDefault("global.reconnect.max_attempts", 5)
	v.SetDefault("global.reconnect.initial_delay", 1*time.Second)
	v.SetDefault("global.reconnect.max_delay", 60*time.Second)
	v.SetDefault("global.reconnect.backoff_multiplier", 2.0)

	// Keep-alive defaults
	v.SetDefault("global.keep_alive.enabled", true)
	v.SetDefault("global.keep_alive.interval", 30*time.Second)
	v.SetDefault("global.keep_alive.timeout", 15*time.Second)
}

// expandPaths expands ~ in file paths
func expandPaths(cfg *Config) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Expand log file path
	if strings.HasPrefix(cfg.Global.LogFile, "~/") {
		cfg.Global.LogFile = filepath.Join(home, cfg.Global.LogFile[2:])
	}

	// Expand identity file paths in hosts
	for name, host := range cfg.Hosts {
		if strings.HasPrefix(host.IdentityFile, "~/") {
			host.IdentityFile = filepath.Join(home, host.IdentityFile[2:])
			cfg.Hosts[name] = host
		}
	}
}

// CreateDefaultConfig creates a default configuration file at the specified path
func CreateDefaultConfig(path string) error {
	// Expand ~ in path
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot expand home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %s", path)
	}

	// Write default config
	defaultConfig := `# SSH Tunnel Manager Configuration
# Documentation: https://github.com/labbs/sshtunnel

global:
  log_level: info  # debug, info, warn, error
  log_file: ""     # Leave empty to disable file logging
  reconnect:
    enabled: true
    max_attempts: 5        # 0 = unlimited
    initial_delay: 1s
    max_delay: 60s
    backoff_multiplier: 2.0
  keep_alive:
    enabled: true
    interval: 30s
    timeout: 15s

# SSH hosts (reusable)
hosts:
  example-host:
    hostname: example.com
    port: 22
    user: myuser
    identity_file: ~/.ssh/id_ed25519
    agent_forwarding: false
    # jump_host: another-host  # Optional: use another host as jump/bastion
    # options:
    #   StrictHostKeyChecking: accept-new

# Tunnels
tunnels:
  - name: example-tunnel
    description: "Example local port forwarding"
    host: example-host
    type: local  # local (-L), remote (-R), or dynamic (-D)
    local:
      address: 127.0.0.1
      port: 8080
    remote:
      address: localhost
      port: 80
    auto_start: false
    # auto_reconnect: true  # Override global setting

# Groups (optional)
groups:
  - name: example-group
    description: "Example group"
    tunnels:
      - example-tunnel
    auto_start: false
`

	if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
