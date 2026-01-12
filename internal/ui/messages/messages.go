// Package messages defines the message types used for communication between UI components.
package messages

// ViewType represents the current view
type ViewType int

const (
	ViewList ViewType = iota
	ViewGroups
	ViewHelp
)

// ChangeViewMsg is sent to change the current view
type ChangeViewMsg struct {
	View ViewType
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
