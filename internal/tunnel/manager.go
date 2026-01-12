package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/ssh"
)

// Manager manages multiple tunnels
type Manager struct {
	config    *config.Config
	tunnels   map[string]Tunnel
	keepAlive ssh.KeepAliveConfig

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewManager creates a new tunnel manager
func NewManager(cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:  cfg,
		tunnels: make(map[string]Tunnel),
		keepAlive: ssh.KeepAliveConfig{
			Enabled:  cfg.Global.KeepAlive.Enabled,
			Interval: cfg.Global.KeepAlive.Interval,
			Timeout:  cfg.Global.KeepAlive.Timeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize creates all tunnel instances from config
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, tunnelCfg := range m.config.Tunnels {
		hostConfig, ok := m.config.Hosts[tunnelCfg.Host]
		if !ok {
			return fmt.Errorf("host '%s' not found for tunnel '%s'", tunnelCfg.Host, tunnelCfg.Name)
		}

		tunnel, err := NewTunnel(tunnelCfg, hostConfig, m.config.Hosts, m.keepAlive)
		if err != nil {
			return fmt.Errorf("failed to create tunnel '%s': %w", tunnelCfg.Name, err)
		}

		m.tunnels[tunnelCfg.Name] = tunnel
	}

	return nil
}

// StartAutoStart starts all tunnels marked with auto_start
func (m *Manager) StartAutoStart() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tunnel := range m.tunnels {
		if tunnel.Config().AutoStart {
			if err := tunnel.Start(m.ctx); err != nil {
				// Log error but continue with other tunnels
				continue
			}
		}
	}

	return nil
}

// Start starts a tunnel by name
func (m *Manager) Start(name string) error {
	m.mu.RLock()
	tunnel, ok := m.tunnels[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("tunnel '%s' not found", name)
	}

	return tunnel.Start(m.ctx)
}

// Stop stops a tunnel by name
func (m *Manager) Stop(name string) error {
	m.mu.RLock()
	tunnel, ok := m.tunnels[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("tunnel '%s' not found", name)
	}

	return tunnel.Stop()
}

// Toggle toggles a tunnel (start if stopped, stop if running)
func (m *Manager) Toggle(name string) error {
	m.mu.RLock()
	tunnel, ok := m.tunnels[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("tunnel '%s' not found", name)
	}

	if tunnel.Status() == StatusRunning {
		return tunnel.Stop()
	}
	return tunnel.Start(m.ctx)
}

// StartAll starts all tunnels
func (m *Manager) StartAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tunnel := range m.tunnels {
		if tunnel.Status() != StatusRunning {
			tunnel.Start(m.ctx)
		}
	}
	return nil
}

// StopAll stops all tunnels
func (m *Manager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tunnel := range m.tunnels {
		if tunnel.Status() == StatusRunning {
			tunnel.Stop()
		}
	}
	return nil
}

// StartGroup starts all tunnels in a group
func (m *Manager) StartGroup(groupName string) error {
	group := m.findGroup(groupName)
	if group == nil {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tunnelName := range group.Tunnels {
		if tunnel, ok := m.tunnels[tunnelName]; ok {
			if tunnel.Status() != StatusRunning {
				tunnel.Start(m.ctx)
			}
		}
	}
	return nil
}

// StopGroup stops all tunnels in a group
func (m *Manager) StopGroup(groupName string) error {
	group := m.findGroup(groupName)
	if group == nil {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tunnelName := range group.Tunnels {
		if tunnel, ok := m.tunnels[tunnelName]; ok {
			if tunnel.Status() == StatusRunning {
				tunnel.Stop()
			}
		}
	}
	return nil
}

// findGroup finds a group by name
func (m *Manager) findGroup(name string) *config.GroupConfig {
	for _, group := range m.config.Groups {
		if group.Name == name {
			return &group
		}
	}
	return nil
}

// GetTunnel returns a tunnel by name
func (m *Manager) GetTunnel(name string) (Tunnel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tunnel, ok := m.tunnels[name]
	return tunnel, ok
}

// GetTunnels returns all tunnels
func (m *Manager) GetTunnels() []Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]Tunnel, 0, len(m.tunnels))
	// Return in config order
	for _, cfg := range m.config.Tunnels {
		if t, ok := m.tunnels[cfg.Name]; ok {
			tunnels = append(tunnels, t)
		}
	}
	return tunnels
}

// GetGroups returns all groups
func (m *Manager) GetGroups() []config.GroupConfig {
	return m.config.Groups
}

// GetGroupStatus returns the status of a group (active/total)
func (m *Manager) GetGroupStatus(groupName string) (active, total int) {
	group := m.findGroup(groupName)
	if group == nil {
		return 0, 0
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(group.Tunnels)
	for _, tunnelName := range group.Tunnels {
		if tunnel, ok := m.tunnels[tunnelName]; ok {
			if tunnel.Status() == StatusRunning {
				active++
			}
		}
	}
	return active, total
}

// GetStats returns aggregated statistics
func (m *Manager) GetStats() (running, total int, bytesSent, bytesReceived uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.tunnels)
	for _, tunnel := range m.tunnels {
		if tunnel.Status() == StatusRunning {
			running++
		}
		stats := tunnel.Stats()
		bytesSent += stats.BytesSent()
		bytesReceived += stats.BytesReceived()
	}
	return
}

// Shutdown stops all tunnels and cleans up
func (m *Manager) Shutdown(timeout time.Duration) error {
	m.cancel()

	// Stop all tunnels
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tunnel := range m.tunnels {
		tunnel.Stop()
	}

	return nil
}

// Reload reloads the configuration
func (m *Manager) Reload(newConfig *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store old tunnels status
	oldStatus := make(map[string]bool)
	for name, tunnel := range m.tunnels {
		oldStatus[name] = tunnel.Status() == StatusRunning
	}

	// Stop all tunnels
	for _, tunnel := range m.tunnels {
		tunnel.Stop()
	}

	// Update config
	m.config = newConfig
	m.keepAlive = ssh.KeepAliveConfig{
		Enabled:  newConfig.Global.KeepAlive.Enabled,
		Interval: newConfig.Global.KeepAlive.Interval,
		Timeout:  newConfig.Global.KeepAlive.Timeout,
	}

	// Recreate tunnels
	m.tunnels = make(map[string]Tunnel)

	var errs []error
	for _, tunnelCfg := range m.config.Tunnels {
		hostConfig, ok := m.config.Hosts[tunnelCfg.Host]
		if !ok {
			errs = append(errs, fmt.Errorf("tunnel %q: host %q not found", tunnelCfg.Name, tunnelCfg.Host))
			continue
		}

		tunnel, err := NewTunnel(tunnelCfg, hostConfig, m.config.Hosts, m.keepAlive)
		if err != nil {
			errs = append(errs, fmt.Errorf("tunnel %q: %w", tunnelCfg.Name, err))
			continue
		}

		m.tunnels[tunnelCfg.Name] = tunnel

		// Restart if was running before
		if oldStatus[tunnelCfg.Name] {
			tunnel.Start(m.ctx)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to reload %d tunnel(s): %v", len(errs), errs)
	}

	return nil
}
