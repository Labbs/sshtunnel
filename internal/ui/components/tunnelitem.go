// Package components provides reusable UI components for the terminal interface.
package components

import (
	"fmt"

	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/format"
	"github.com/labbs/sshtunnel/internal/tunnel"
	"github.com/labbs/sshtunnel/internal/ui/styles"
)

// TunnelItem represents a tunnel item in the list
type TunnelItem struct {
	tunnel   tunnel.Tunnel
	selected bool
	width    int
}

// NewTunnelItem creates a new tunnel item
func NewTunnelItem(t tunnel.Tunnel) *TunnelItem {
	return &TunnelItem{
		tunnel: t,
	}
}

// SetSelected sets whether the item is selected
func (t *TunnelItem) SetSelected(selected bool) {
	t.selected = selected
}

// SetWidth sets the width of the item
func (t *TunnelItem) SetWidth(width int) {
	t.width = width
}

// Render renders the tunnel item
func (t *TunnelItem) Render() string {
	cfg := t.tunnel.Config()
	status := t.tunnel.Status()
	stats := t.tunnel.Stats()

	// Selection marker
	var marker string
	if t.selected {
		marker = styles.HelpKeyStyle.Render(">")
	} else {
		marker = " "
	}

	// Status indicator
	indicator := getStatusIndicator(status)

	// Tunnel name
	name := styles.TunnelNameStyle.Render(cfg.Name)

	// Tunnel type and addresses
	typeStr := tunnel.FormatTunnelType(cfg)
	localAddr := tunnel.FormatLocalAddr(cfg)
	remoteAddr := tunnel.FormatRemoteAddr(cfg)

	var addrStr string
	if cfg.Type == config.TunnelTypeRemote {
		addrStr = fmt.Sprintf("%s %s ← %s", typeStr, remoteAddr, localAddr)
	} else if cfg.Type == config.TunnelTypeDynamic {
		addrStr = fmt.Sprintf("%s %s %s", typeStr, localAddr, remoteAddr)
	} else {
		addrStr = fmt.Sprintf("%s %s → %s", typeStr, localAddr, remoteAddr)
	}
	addrStr = styles.TunnelTypeStyle.Render(addrStr)

	// First line: marker, indicator, name, address
	line1 := fmt.Sprintf("%s %s %s    %s", marker, indicator, name, addrStr)

	// Second line: status info
	var line2 string
	switch status {
	case tunnel.StatusRunning:
		uptime := format.Duration(stats.Uptime())
		sent, recv := stats.BytesSent(), stats.BytesReceived()
		conns := stats.Connections()
		line2 = styles.StatsStyle.Render(fmt.Sprintf("    ↑ %s  ⇅ %s / %s  ⚡ %d conn",
			uptime, format.Bytes(sent), format.Bytes(recv), conns))
	case tunnel.StatusStopped:
		line2 = styles.MutedStyle.Render("    Stopped")
	case tunnel.StatusConnecting:
		line2 = styles.WarningStyle.Render("    Connecting...")
	case tunnel.StatusReconnecting:
		line2 = styles.WarningStyle.Render("    Reconnecting...")
	case tunnel.StatusError:
		errMsg := "Unknown error"
		if err := t.tunnel.Error(); err != nil {
			errMsg = err.Error()
			if len(errMsg) > 50 {
				errMsg = errMsg[:50] + "..."
			}
		}
		line2 = styles.ErrorStyle.Render(fmt.Sprintf("    Error: %s", errMsg))
	}

	return line1 + "\n" + line2
}

// getStatusIndicator returns the status indicator string for a tunnel status.
func getStatusIndicator(status tunnel.Status) string {
	switch status {
	case tunnel.StatusRunning:
		return styles.StatusRunning.String()
	case tunnel.StatusStopped:
		return styles.StatusStopped.String()
	case tunnel.StatusReconnecting:
		return styles.StatusReconnecting.String()
	case tunnel.StatusError:
		return styles.StatusError.String()
	case tunnel.StatusConnecting:
		return styles.StatusConnecting.String()
	default:
		return styles.StatusStopped.String()
	}
}
