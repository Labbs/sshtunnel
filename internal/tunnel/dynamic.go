package tunnel

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/ssh"
)

// DynamicTunnel implements SOCKS5 proxy (-D)
type DynamicTunnel struct {
	*BaseTunnel
	listener net.Listener
}

// NewDynamicTunnel creates a new dynamic (SOCKS5) tunnel
func NewDynamicTunnel(cfg config.TunnelConfig, hostConfig config.HostConfig, hosts map[string]config.HostConfig, keepAlive ssh.KeepAliveConfig) *DynamicTunnel {
	return &DynamicTunnel{
		BaseTunnel: NewBaseTunnel(cfg, hostConfig, hosts, keepAlive),
	}
}

// Start starts the SOCKS5 proxy
func (t *DynamicTunnel) Start(ctx context.Context) error {
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

	// Start local SOCKS5 listener
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

// acceptLoop accepts incoming SOCKS5 connections
func (t *DynamicTunnel) acceptLoop() {
	defer t.wg.Done()

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.ctx.Done():
				return
			default:
				return
			}
		}

		t.stats.IncrementConnections()
		t.wg.Add(1)
		go t.handleSOCKS5(conn)
	}
}

// SOCKS5 constants
const (
	socks5Version    = 0x05
	socks5AuthNone   = 0x00
	socks5CmdConnect = 0x01
	socks5AtypIPv4   = 0x01
	socks5AtypDomain = 0x03
	socks5AtypIPv6   = 0x04
)

// handleSOCKS5 handles a SOCKS5 connection
func (t *DynamicTunnel) handleSOCKS5(conn net.Conn) {
	defer t.wg.Done()
	defer conn.Close()
	defer t.stats.DecrementConnections()

	// Read version and auth methods
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n < 2 {
		return
	}

	if buf[0] != socks5Version {
		return
	}

	// Respond with no auth required
	_, err = conn.Write([]byte{socks5Version, socks5AuthNone})
	if err != nil {
		return
	}

	// Read connection request
	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	if buf[0] != socks5Version || buf[1] != socks5CmdConnect {
		// Only support CONNECT command
		conn.Write([]byte{socks5Version, 0x07, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return
	}

	// Parse destination address
	var destAddr string
	var destPort uint16

	switch buf[3] {
	case socks5AtypIPv4:
		if n < 10 {
			return
		}
		destAddr = net.IP(buf[4:8]).String()
		destPort = binary.BigEndian.Uint16(buf[8:10])
	case socks5AtypDomain:
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return
		}
		destAddr = string(buf[5 : 5+domainLen])
		destPort = binary.BigEndian.Uint16(buf[5+domainLen : 7+domainLen])
	case socks5AtypIPv6:
		if n < 22 {
			return
		}
		destAddr = net.IP(buf[4:20]).String()
		destPort = binary.BigEndian.Uint16(buf[20:22])
	default:
		conn.Write([]byte{socks5Version, 0x08, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return
	}

	// Connect through SSH tunnel
	target := fmt.Sprintf("%s:%d", destAddr, destPort)
	client := t.SSHClient()
	if client == nil {
		conn.Write([]byte{socks5Version, 0x01, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return
	}

	remoteConn, err := client.Dial("tcp", target)
	if err != nil {
		conn.Write([]byte{socks5Version, 0x05, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remoteConn.Close()

	// Send success response
	_, err = conn.Write([]byte{socks5Version, 0x00, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return
	}

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Remote
	go func() {
		defer wg.Done()
		n, err := io.Copy(remoteConn, conn)
		t.stats.AddBytesSent(uint64(n))
		if err != nil && !isClosedConnectionError(err) {
			t.SetError(fmt.Errorf("copy client->remote: %w", err))
		}
	}()

	// Remote -> Client
	go func() {
		defer wg.Done()
		n, err := io.Copy(conn, remoteConn)
		t.stats.AddBytesReceived(uint64(n))
		if err != nil && !isClosedConnectionError(err) {
			t.SetError(fmt.Errorf("copy remote->client: %w", err))
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

// Stop stops the SOCKS5 proxy
func (t *DynamicTunnel) Stop() error {
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
