// Package ssh provides SSH client functionality including direct connections,
// jump host support, and authentication methods (key-based and SSH agent).
package ssh

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/labbs/sshtunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// Client represents an SSH client connection
type Client struct {
	config     config.HostConfig
	sshClient  *ssh.Client
	mu         sync.RWMutex
	keepAlive  KeepAliveConfig
	cancelFunc context.CancelFunc
}

// KeepAliveConfig contains keep-alive settings
type KeepAliveConfig struct {
	Enabled  bool
	Interval time.Duration
	Timeout  time.Duration
}

// NewClient creates a new SSH client
func NewClient(hostConfig config.HostConfig, keepAlive KeepAliveConfig) *Client {
	return &Client{
		config:    hostConfig,
		keepAlive: keepAlive,
	}
}

// Connect establishes an SSH connection
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sshClient != nil {
		return nil // Already connected
	}

	// Get auth methods
	authMethods, cleanup, err := GetAuthMethods(c.config.IdentityFile, true)
	if err != nil {
		return fmt.Errorf("failed to get auth methods: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
		Timeout:         30 * time.Second,
	}

	// Build the address
	addr := fmt.Sprintf("%s:%d", c.config.Hostname, c.config.Port)

	// Connect (possibly through jump host)
	client, err := c.dial(ctx, addr, sshConfig)
	if err != nil {
		return err
	}

	c.sshClient = client

	// Start keep-alive if enabled
	if c.keepAlive.Enabled {
		keepAliveCtx, cancel := context.WithCancel(ctx)
		c.cancelFunc = cancel
		go c.runKeepAlive(keepAliveCtx)
	}

	return nil
}

// dial connects to the SSH server, optionally through a jump host
func (c *Client) dial(ctx context.Context, addr string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	// Direct connection
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

// runKeepAlive sends periodic keep-alive requests
func (c *Client) runKeepAlive(ctx context.Context) {
	ticker := time.NewTicker(c.keepAlive.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			client := c.sshClient
			c.mu.RUnlock()

			if client == nil {
				return
			}

			// Send keep-alive request
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				// Connection might be dead, but we don't close here
				// Let the tunnel handler deal with reconnection
				continue
			}
		}
	}
}

// Close closes the SSH connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}

	if c.sshClient != nil {
		err := c.sshClient.Close()
		c.sshClient = nil
		return err
	}
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshClient != nil
}

// SSHClient returns the underlying SSH client
func (c *Client) SSHClient() *ssh.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshClient
}

// OpenChannel opens a new channel on the SSH connection
func (c *Client) OpenChannel(name string, data []byte) (ssh.Channel, <-chan *ssh.Request, error) {
	c.mu.RLock()
	client := c.sshClient
	c.mu.RUnlock()

	if client == nil {
		return nil, nil, fmt.Errorf("not connected")
	}

	return client.OpenChannel(name, data)
}

// Dial opens a connection to the remote address through the SSH connection
func (c *Client) Dial(network, addr string) (net.Conn, error) {
	c.mu.RLock()
	client := c.sshClient
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected")
	}

	return client.Dial(network, addr)
}

// Listen creates a listener on the remote SSH server
func (c *Client) Listen(network, addr string) (net.Listener, error) {
	c.mu.RLock()
	client := c.sshClient
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected")
	}

	return client.Listen(network, addr)
}
