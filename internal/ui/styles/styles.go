// Package styles defines the visual styling for the terminal UI using lipgloss.
package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - using ANSI 256 color codes for better terminal compatibility
	ColorPrimary   = lipgloss.Color("135") // Purple
	ColorSecondary = lipgloss.Color("69")  // Blue
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorWarning   = lipgloss.Color("214") // Yellow/Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("245") // Gray
	ColorText      = lipgloss.Color("255") // White
	ColorDim       = lipgloss.Color("250") // Light gray
	ColorBgDark    = lipgloss.Color("236") // Dark background

	// Status indicators
	StatusRunning      = lipgloss.NewStyle().Foreground(ColorSuccess).SetString("●")
	StatusStopped      = lipgloss.NewStyle().Foreground(ColorMuted).SetString("○")
	StatusReconnecting = lipgloss.NewStyle().Foreground(ColorWarning).SetString("◐")
	StatusError        = lipgloss.NewStyle().Foreground(ColorError).SetString("✗")
	StatusConnecting   = lipgloss.NewStyle().Foreground(ColorWarning).SetString("◌")

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorPrimary)

	// Selected item style
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorSecondary)

	// Normal item style
	NormalStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Dim style for secondary info
	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// Muted style
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	// Help key style
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// Help description style
	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Border style
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)

	// Status bar style
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgDark)

	// Tunnel name style
	TunnelNameStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	// Tunnel type style
	TunnelTypeStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// Stats style
	StatsStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// Group style
	GroupStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Section header style
	SectionHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Bold(true)
)
