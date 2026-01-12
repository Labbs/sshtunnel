// Package tunnel implements SSH tunnel management including local (-L),
// remote (-R), and dynamic (-D SOCKS5) port forwarding with automatic reconnection.
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/ssh"
)

// Status represents the current state of a tunnel
type Status int

const (
	StatusStopped Status = iota
	StatusConnecting
	StatusRunning
	StatusReconnecting
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusConnecting:
		return "connecting"
	case StatusRunning:
		return "running"
	case StatusReconnecting:
		return "reconnecting"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Tunnel represents an SSH tunnel
type Tunnel interface {
	// Start starts the tunnel
	Start(ctx context.Context) error
	// Stop stops the tunnel
	Stop() error
	// Status returns the current status
	Status() Status
	// Stats returns the tunnel statistics
	Stats() *Stats
	// Config returns the tunnel configuration
	Config() config.TunnelConfig
	// Error returns the last error, if any
	Error() error
	// Name returns the tunnel name
	Name() string
}

// BaseTunnel contains common tunnel functionality
type BaseTunnel struct {
	config     config.TunnelConfig
	hostConfig config.HostConfig
	hosts      map[string]config.HostConfig
	keepAlive  ssh.KeepAliveConfig

	sshClient ssh.SSHClientInterface
	stats     *Stats
	status    Status
	lastError error

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// NewBaseTunnel creates a new base tunnel
func NewBaseTunnel(cfg config.TunnelConfig, hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive ssh.KeepAliveConfig) *BaseTunnel {
	return &BaseTunnel{
		config:     cfg,
		hostConfig: hostConfig,
		hosts:      hosts,
		keepAlive:  keepAlive,
		stats:      NewStats(),
		status:     StatusStopped,
	}
}

// Config returns the tunnel configuration
func (t *BaseTunnel) Config() config.TunnelConfig {
	return t.config
}

// Name returns the tunnel name
func (t *BaseTunnel) Name() string {
	return t.config.Name
}

// Status returns the current tunnel status
func (t *BaseTunnel) Status() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// SetStatus sets the tunnel status
func (t *BaseTunnel) SetStatus(status Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = status
	if status == StatusRunning {
		t.stats.SetConnected()
	} else if status == StatusStopped || status == StatusError {
		t.stats.SetDisconnected()
	}
}

// Stats returns the tunnel statistics
func (t *BaseTunnel) Stats() *Stats {
	return t.stats
}

// Error returns the last error
func (t *BaseTunnel) Error() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastError
}

// SetError sets the last error
func (t *BaseTunnel) SetError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastError = err
}

// Connect establishes the SSH connection
func (t *BaseTunnel) Connect(ctx context.Context) error {
	t.SetStatus(StatusConnecting)

	client, err := ssh.NewSSHClient(t.hostConfig, t.hosts, t.keepAlive)
	if err != nil {
		t.SetError(err)
		t.SetStatus(StatusError)
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		t.SetError(err)
		t.SetStatus(StatusError)
		return fmt.Errorf("failed to connect: %w", err)
	}

	t.mu.Lock()
	t.sshClient = client
	t.mu.Unlock()

	return nil
}

// Disconnect closes the SSH connection
func (t *BaseTunnel) Disconnect() error {
	t.mu.Lock()
	client := t.sshClient
	t.sshClient = nil
	t.mu.Unlock()

	if client != nil {
		return client.Close()
	}
	return nil
}

// SSHClient returns the SSH client
func (t *BaseTunnel) SSHClient() ssh.SSHClientInterface {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sshClient
}

// NewTunnel creates a new tunnel based on the configuration type
func NewTunnel(cfg config.TunnelConfig, hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive ssh.KeepAliveConfig) (Tunnel, error) {
	switch cfg.Type {
	case config.TunnelTypeLocal:
		return NewLocalTunnel(cfg, hostConfig, hosts, keepAlive), nil
	case config.TunnelTypeRemote:
		return NewRemoteTunnel(cfg, hostConfig, hosts, keepAlive), nil
	case config.TunnelTypeDynamic:
		return NewDynamicTunnel(cfg, hostConfig, hosts, keepAlive), nil
	default:
		return nil, fmt.Errorf("unknown tunnel type: %s", cfg.Type)
	}
}

// FormatLocalAddr formats the local address for display
func FormatLocalAddr(cfg config.TunnelConfig) string {
	addr := cfg.Local.Address
	if addr == "" {
		addr = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", addr, cfg.Local.Port)
}

// FormatRemoteAddr formats the remote address for display
func FormatRemoteAddr(cfg config.TunnelConfig) string {
	if cfg.Type == config.TunnelTypeDynamic {
		return "(SOCKS5)"
	}
	addr := cfg.Remote.Address
	if addr == "" {
		addr = "localhost"
	}
	return fmt.Sprintf("%s:%d", addr, cfg.Remote.Port)
}

// FormatTunnelType formats the tunnel type for display
func FormatTunnelType(cfg config.TunnelConfig) string {
	switch cfg.Type {
	case config.TunnelTypeLocal:
		return "-L"
	case config.TunnelTypeRemote:
		return "-R"
	case config.TunnelTypeDynamic:
		return "-D"
	default:
		return "?"
	}
}

// Stats holds tunnel statistics
type Stats struct {
	mu             sync.RWMutex
	bytesSent      uint64
	bytesReceived  uint64
	connections    int64
	startTime      time.Time
	connectedAt    time.Time
	isConnected    bool
	reconnectCount int
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		startTime: time.Now(),
	}
}

// AddBytesSent adds to the bytes sent counter
func (s *Stats) AddBytesSent(n uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytesSent += n
}

// AddBytesReceived adds to the bytes received counter
func (s *Stats) AddBytesReceived(n uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytesReceived += n
}

// IncrementConnections increments the active connections counter
func (s *Stats) IncrementConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections++
}

// DecrementConnections decrements the active connections counter
func (s *Stats) DecrementConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connections > 0 {
		s.connections--
	}
}

// SetConnected marks the tunnel as connected
func (s *Stats) SetConnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isConnected = true
	s.connectedAt = time.Now()
}

// SetDisconnected marks the tunnel as disconnected
func (s *Stats) SetDisconnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isConnected = false
}

// IncrementReconnectCount increments the reconnect counter
func (s *Stats) IncrementReconnectCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconnectCount++
}

// GetStats returns all statistics
func (s *Stats) GetStats() (bytesSent, bytesReceived uint64, connections int64, uptime time.Duration, reconnects int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var up time.Duration
	if s.isConnected {
		up = time.Since(s.connectedAt)
	}

	return s.bytesSent, s.bytesReceived, s.connections, up, s.reconnectCount
}

// Connections returns the current number of active connections
func (s *Stats) Connections() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections
}

// Uptime returns the current uptime
func (s *Stats) Uptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.isConnected {
		return 0
	}
	return time.Since(s.connectedAt)
}

// BytesSent returns the total bytes sent
func (s *Stats) BytesSent() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bytesSent
}

// BytesReceived returns the total bytes received
func (s *Stats) BytesReceived() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bytesReceived
}

// isClosedConnectionError checks if the error is due to a closed connection.
// These errors are expected during normal shutdown and should not be reported.
func isClosedConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe")
}
