// Package config provides configuration types and loading for the SSH tunnel manager.
// It supports YAML configuration files with hosts, tunnels, and groups definitions.
package config

import "time"

// Config represents the main configuration structure
type Config struct {
	Global  GlobalConfig          `mapstructure:"global"`
	Hosts   map[string]HostConfig `mapstructure:"hosts"`
	Tunnels []TunnelConfig        `mapstructure:"tunnels"`
	Groups  []GroupConfig         `mapstructure:"groups"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	LogLevel  string          `mapstructure:"log_level"`
	LogFile   string          `mapstructure:"log_file"`
	Reconnect ReconnectConfig `mapstructure:"reconnect"`
	KeepAlive KeepAliveConfig `mapstructure:"keep_alive"`
}

// ReconnectConfig contains auto-reconnect settings
type ReconnectConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	MaxAttempts       int           `mapstructure:"max_attempts"`
	InitialDelay      time.Duration `mapstructure:"initial_delay"`
	MaxDelay          time.Duration `mapstructure:"max_delay"`
	BackoffMultiplier float64       `mapstructure:"backoff_multiplier"`
}

// KeepAliveConfig contains SSH keep-alive settings
type KeepAliveConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// HostConfig represents an SSH host configuration
type HostConfig struct {
	Hostname        string            `mapstructure:"hostname"`
	Port            int               `mapstructure:"port"`
	User            string            `mapstructure:"user"`
	IdentityFile    string            `mapstructure:"identity_file"`
	AgentForwarding bool              `mapstructure:"agent_forwarding"`
	JumpHost        string            `mapstructure:"jump_host"`
	Options         map[string]string `mapstructure:"options"`
}

// TunnelConfig represents a tunnel configuration
type TunnelConfig struct {
	Name          string         `mapstructure:"name"`
	Description   string         `mapstructure:"description"`
	Host          string         `mapstructure:"host"`
	Type          TunnelType     `mapstructure:"type"`
	Local         EndpointConfig `mapstructure:"local"`
	Remote        EndpointConfig `mapstructure:"remote"`
	AutoStart     bool           `mapstructure:"auto_start"`
	AutoReconnect *bool          `mapstructure:"auto_reconnect"`
}

// TunnelType represents the type of tunnel
type TunnelType string

const (
	TunnelTypeLocal   TunnelType = "local"   // -L local port forwarding
	TunnelTypeRemote  TunnelType = "remote"  // -R remote port forwarding
	TunnelTypeDynamic TunnelType = "dynamic" // -D SOCKS5 proxy
)

// EndpointConfig represents a local or remote endpoint
type EndpointConfig struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

// GroupConfig represents a group of tunnels
type GroupConfig struct {
	Name        string   `mapstructure:"name"`
	Description string   `mapstructure:"description"`
	Tunnels     []string `mapstructure:"tunnels"`
	AutoStart   bool     `mapstructure:"auto_start"`
}

// ShouldAutoReconnect returns whether the tunnel should auto-reconnect
// using the tunnel-specific setting if set, otherwise the global setting
func (t *TunnelConfig) ShouldAutoReconnect(global bool) bool {
	if t.AutoReconnect != nil {
		return *t.AutoReconnect
	}
	return global
}

// GetHostConfig returns the host configuration for this tunnel
func (t *TunnelConfig) GetHostConfig(hosts map[string]HostConfig) (HostConfig, bool) {
	host, ok := hosts[t.Host]
	return host, ok
}
