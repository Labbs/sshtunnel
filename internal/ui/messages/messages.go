// Package messages defines the message types used for communication between UI components.
package messages

import (
	"github.com/labbs/sshtunnel/internal/tunnel"
)

// ViewType represents the current view
type ViewType int

const (
	ViewList ViewType = iota
	ViewGroups
	ViewDetail
	ViewLogs
	ViewHelp
)

// ChangeViewMsg is sent to change the current view
type ChangeViewMsg struct {
	View ViewType
}

// TunnelActionMsg is sent to perform an action on a tunnel
type TunnelActionMsg struct {
	Action     TunnelAction
	TunnelName string
	GroupName  string
}

// TunnelAction represents an action to perform on a tunnel
type TunnelAction int

const (
	ActionStart TunnelAction = iota
	ActionStop
	ActionToggle
	ActionStartAll
	ActionStopAll
	ActionStartGroup
	ActionStopGroup
)

// TunnelStatusMsg is sent when a tunnel's status changes
type TunnelStatusMsg struct {
	TunnelName string
	Status     tunnel.Status
	Error      error
}

// RefreshMsg is sent to refresh the display
type RefreshMsg struct{}

// TickMsg is sent periodically for updates
type TickMsg struct{}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Error error
}

// ReloadConfigMsg is sent to reload the configuration
type ReloadConfigMsg struct{}

// QuitMsg is sent to quit the application
type QuitMsg struct{}

// FilterMsg is sent to filter the tunnel list
type FilterMsg struct {
	Query string
}
