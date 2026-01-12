package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/labbs/sshtunnel/internal/format"
	"github.com/labbs/sshtunnel/internal/ui/styles"
)

// StatusBar represents the bottom status bar
type StatusBar struct {
	width int
}

// NewStatusBar creates a new status bar
func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

// SetWidth sets the width of the status bar
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// Render renders the status bar
func (s *StatusBar) Render(running, total int, bytesSent, bytesReceived uint64) string {
	width := s.width
	if width < 40 {
		width = 80
	}

	left := fmt.Sprintf(" %d/%d tunnels active", running, total)

	// Format bytes
	sentStr := format.Bytes(bytesSent)
	recvStr := format.Bytes(bytesReceived)
	right := fmt.Sprintf("↑ %s  ↓ %s ", sentStr, recvStr)

	// Calculate padding
	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	spacer := ""
	for i := 0; i < padding; i++ {
		spacer += " "
	}

	return styles.StatusBarStyle.Width(width).Render(left + spacer + right)
}
