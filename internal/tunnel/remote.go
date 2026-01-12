package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/ssh"
)

// RemoteTunnel implements remote port forwarding (-R)
type RemoteTunnel struct {
	*BaseTunnel
	listener net.Listener
}

// NewRemoteTunnel creates a new remote tunnel
func NewRemoteTunnel(cfg config.TunnelConfig, hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive ssh.KeepAliveConfig) *RemoteTunnel {
	return &RemoteTunnel{
		BaseTunnel: NewBaseTunnel(cfg, hostConfig, hosts, keepAlive),
	}
}

// Start starts the remote tunnel
func (t *RemoteTunnel) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.status == StatusRunning || t.status == StatusConnecting {
		t.mu.Unlock()
		return fmt.Errorf("tunnel already running")
	}
	t.ctx, t.cancel = context.WithCancel(ctx)
	t.mu.Unlock()

	// Connect to SSH server
	if err := t.Connect(t.ctx); err != nil {
		return err
	}

	client := t.SSHClient()
	if client == nil {
		t.SetError(fmt.Errorf("SSH client not available"))
		t.SetStatus(StatusError)
		return fmt.Errorf("SSH client not available")
	}

	// Start remote listener on the SSH server
	remoteAddr := fmt.Sprintf("%s:%d", t.config.Remote.Address, t.config.Remote.Port)
	listener, err := client.Listen("tcp", remoteAddr)
	if err != nil {
		t.Disconnect()
		t.SetError(err)
		t.SetStatus(StatusError)
		return fmt.Errorf("failed to listen on remote %s: %w", remoteAddr, err)
	}

	t.mu.Lock()
	t.listener = listener
	t.mu.Unlock()

	t.SetStatus(StatusRunning)
	t.SetError(nil)

	// Start accepting connections
	t.wg.Add(1)
	go t.acceptLoop()

	return nil
}

// acceptLoop accepts incoming connections from the remote side
func (t *RemoteTunnel) acceptLoop() {
	defer t.wg.Done()

	for {
		remoteConn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.ctx.Done():
				return
			default:
				// Listener was closed
				return
			}
		}

		t.stats.IncrementConnections()
		t.wg.Add(1)
		go t.handleConnection(remoteConn)
	}
}

// handleConnection handles a single connection from the remote side
func (t *RemoteTunnel) handleConnection(remoteConn net.Conn) {
	defer t.wg.Done()
	defer remoteConn.Close()
	defer t.stats.DecrementConnections()

	// Connect to local service
	localAddr := fmt.Sprintf("%s:%d", t.config.Local.Address, t.config.Local.Port)
	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		return
	}
	defer localConn.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Remote -> Local
	go func() {
		defer wg.Done()
		n, err := io.Copy(localConn, remoteConn)
		t.stats.AddBytesReceived(uint64(n))
		if err != nil && !isClosedConnectionError(err) {
			t.SetError(fmt.Errorf("copy remote->local: %w", err))
		}
	}()

	// Local -> Remote
	go func() {
		defer wg.Done()
		n, err := io.Copy(remoteConn, localConn)
		t.stats.AddBytesSent(uint64(n))
		if err != nil && !isClosedConnectionError(err) {
			t.SetError(fmt.Errorf("copy local->remote: %w", err))
		}
	}()

	// Wait for either direction to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-t.ctx.Done():
	case <-done:
	}
}

// Stop stops the remote tunnel
func (t *RemoteTunnel) Stop() error {
	t.mu.Lock()
	if t.status == StatusStopped {
		t.mu.Unlock()
		return nil
	}

	if t.cancel != nil {
		t.cancel()
	}

	listener := t.listener
	t.listener = nil
	t.mu.Unlock()

	if listener != nil {
		listener.Close()
	}

	// Wait for all goroutines to finish
	t.wg.Wait()

	// Disconnect SSH
	t.Disconnect()

	t.SetStatus(StatusStopped)
	return nil
}
