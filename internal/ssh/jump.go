package ssh

import (
	"context"
	"fmt"
	"time"

	"github.com/labbs/sshtunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// JumpClient represents an SSH client that connects through a jump host
type JumpClient struct {
	*Client
	jumpClient *Client
	jumpConfig config.HostConfig
	hosts      map[string]config.HostConfig
}

// NewJumpClient creates a new SSH client that connects through a jump host
func NewJumpClient(hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive KeepAliveConfig) (*JumpClient, error) {
	jumpHostName := hostConfig.JumpHost
	if jumpHostName == "" {
		return nil, fmt.Errorf("no jump host specified")
	}

	jumpConfig, ok := hosts[jumpHostName]
	if !ok {
		return nil, fmt.Errorf("jump host '%s' not found", jumpHostName)
	}

	return &JumpClient{
		Client:     NewClient(hostConfig, keepAlive),
		jumpConfig: jumpConfig,
		hosts:      hosts,
	}, nil
}

// Connect establishes an SSH connection through the jump host
func (c *JumpClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sshClient != nil {
		return nil // Already connected
	}

	// First, connect to the jump host
	// Check if the jump host itself has a jump host (chained jumps)
	var jumpClient *Client
	if c.jumpConfig.JumpHost != "" {
		nestedJump, err := NewJumpClient(c.jumpConfig, c.hosts, c.keepAlive)
		if err != nil {
			return fmt.Errorf("failed to create nested jump client: %w", err)
		}
		if err := nestedJump.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to nested jump host: %w", err)
		}
		jumpClient = nestedJump.Client
		c.jumpClient = nestedJump.Client
	} else {
		jumpClient = NewClient(c.jumpConfig, c.keepAlive)
		if err := jumpClient.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to jump host: %w", err)
		}
		c.jumpClient = jumpClient
	}

	// Get auth methods for the target host
	authMethods, cleanup, err := GetAuthMethods(c.config.IdentityFile, true)
	if err != nil {
		jumpClient.Close()
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

	// Dial through the jump host
	targetAddr := fmt.Sprintf("%s:%d", c.config.Hostname, c.config.Port)
	conn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		jumpClient.Close()
		return fmt.Errorf("failed to dial through jump host: %w", err)
	}

	// Establish SSH connection through the tunnel
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, sshConfig)
	if err != nil {
		conn.Close()
		jumpClient.Close()
		return fmt.Errorf("failed to establish SSH connection through jump host: %w", err)
	}

	c.sshClient = ssh.NewClient(sshConn, chans, reqs)

	// Start keep-alive if enabled
	if c.keepAlive.Enabled {
		keepAliveCtx, cancel := context.WithCancel(ctx)
		c.cancelFunc = cancel
		go c.runKeepAlive(keepAliveCtx)
	}

	return nil
}

// Close closes both the main connection and the jump host connection
func (c *JumpClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}

	var err error
	if c.sshClient != nil {
		err = c.sshClient.Close()
		c.sshClient = nil
	}

	if c.jumpClient != nil {
		if jumpErr := c.jumpClient.Close(); jumpErr != nil && err == nil {
			err = jumpErr
		}
		c.jumpClient = nil
	}

	return err
}
