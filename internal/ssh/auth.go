package ssh

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AuthMethod represents an SSH authentication method
type AuthMethod interface {
	Method() ssh.AuthMethod
	Name() string
}

// KeyAuth represents SSH key-based authentication
type KeyAuth struct {
	keyPath string
	signer  ssh.Signer
}

// NewKeyAuth creates a new key-based authentication method
func NewKeyAuth(keyPath string) (*KeyAuth, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &KeyAuth{
		keyPath: keyPath,
		signer:  signer,
	}, nil
}

// Method returns the SSH auth method
func (k *KeyAuth) Method() ssh.AuthMethod {
	return ssh.PublicKeys(k.signer)
}

// Name returns the name of this auth method
func (k *KeyAuth) Name() string {
	return fmt.Sprintf("publickey (%s)", k.keyPath)
}

// AgentAuth represents SSH agent-based authentication
type AgentAuth struct {
	conn   net.Conn
	client agent.ExtendedAgent
}

// NewAgentAuth creates a new agent-based authentication method
func NewAgentAuth() (*AgentAuth, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	client := agent.NewClient(conn)

	return &AgentAuth{
		conn:   conn,
		client: client,
	}, nil
}

// Method returns the SSH auth method
func (a *AgentAuth) Method() ssh.AuthMethod {
	return ssh.PublicKeysCallback(a.client.Signers)
}

// Name returns the name of this auth method
func (a *AgentAuth) Name() string {
	return "ssh-agent"
}

// Close closes the agent connection
func (a *AgentAuth) Close() error {
	return a.conn.Close()
}

// Agent returns the SSH agent client for forwarding
func (a *AgentAuth) Agent() agent.ExtendedAgent {
	return a.client
}

// GetAuthMethods returns authentication methods to try for a given key path
// It tries key-based auth first, then falls back to agent if available
func GetAuthMethods(keyPath string, useAgent bool) ([]ssh.AuthMethod, func(), error) {
	var methods []ssh.AuthMethod
	var cleanup func()

	// Try key-based auth first if key path is specified
	if keyPath != "" {
		keyAuth, err := NewKeyAuth(keyPath)
		if err == nil {
			methods = append(methods, keyAuth.Method())
		}
		// Don't fail if key auth doesn't work, try agent next
	}

	// Try agent auth if enabled or as fallback
	if useAgent || len(methods) == 0 {
		agentAuth, err := NewAgentAuth()
		if err == nil {
			methods = append(methods, agentAuth.Method())
			cleanup = func() {
				agentAuth.Close()
			}
		}
	}

	if len(methods) == 0 {
		return nil, nil, fmt.Errorf("no authentication methods available")
	}

	return methods, cleanup, nil
}
