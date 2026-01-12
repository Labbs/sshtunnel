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

// LocalTunnel implements local port forwarding (-L)
type LocalTunnel struct {
	*BaseTunnel
	listener net.Listener
}

// NewLocalTunnel creates a new local tunnel
func NewLocalTunnel(cfg config.TunnelConfig, hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive ssh.KeepAliveConfig) *LocalTunnel {
	return &LocalTunnel{
		BaseTunnel: NewBaseTunnel(cfg, hostConfig, hosts, keepAlive),
	}
}

// Start starts the local tunnel
func (t *LocalTunnel) Start(ctx context.Context) error {
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

	// Start local listener
	localAddr := fmt.Sprintf("%s:%d", t.config.Local.Address, t.config.Local.Port)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		t.Disconnect()
		t.SetError(err)
		t.SetStatus(StatusError)
		return fmt.Errorf("failed to listen on %s: %w", localAddr, err)
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

// acceptLoop accepts incoming connections
func (t *LocalTunnel) acceptLoop() {
	defer t.wg.Done()

	for {
		conn, err := t.listener.Accept()
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
		go t.handleConnection(conn)
	}
}

// handleConnection handles a single connection
func (t *LocalTunnel) handleConnection(localConn net.Conn) {
	defer t.wg.Done()
	defer localConn.Close()
	defer t.stats.DecrementConnections()

	// Connect to remote through SSH
	remoteAddr := fmt.Sprintf("%s:%d", t.config.Remote.Address, t.config.Remote.Port)
	client := t.SSHClient()
	if client == nil {
		return
	}

	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Local -> Remote
	go func() {
		defer wg.Done()
		n, err := io.Copy(remoteConn, localConn)
		t.stats.AddBytesSent(uint64(n))
		if err != nil && !isClosedConnectionError(err) {
			t.SetError(fmt.Errorf("copy local->remote: %w", err))
		}
	}()

	// Remote -> Local
	go func() {
		defer wg.Done()
		n, err := io.Copy(localConn, remoteConn)
		t.stats.AddBytesReceived(uint64(n))
		if err != nil && !isClosedConnectionError(err) {
			t.SetError(fmt.Errorf("copy remote->local: %w", err))
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

// Stop stops the local tunnel
func (t *LocalTunnel) Stop() error {
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
