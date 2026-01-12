package ssh

import (
	"context"
	"net"

	"github.com/labbs/sshtunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// SSHClientInterface defines the interface for SSH client operations.
// Both Client and JumpClient implement this interface.
type SSHClientInterface interface {
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool
	Dial(network, addr string) (net.Conn, error)
	Listen(network, addr string) (net.Listener, error)
	SSHClient() *ssh.Client
}

// NewSSHClient creates an appropriate SSH client based on configuration.
// If the host has a jump host configured, it returns a JumpClient,
// otherwise it returns a direct Client.
func NewSSHClient(hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive KeepAliveConfig) (SSHClientInterface, error) {
	if hostConfig.JumpHost != "" {
		return NewJumpClient(hostConfig, hosts, keepAlive)
	}
	return NewClient(hostConfig, keepAlive), nil
}
