package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/labbs/sshtunnel/internal/ui/messages"
	"github.com/labbs/sshtunnel/internal/ui/styles"
)

// HelpKeyMap defines the key bindings for the help view
type HelpKeyMap struct {
	Back key.Binding
	Quit key.Binding
}

// DefaultHelpKeyMap returns the default key bindings
func DefaultHelpKeyMap() HelpKeyMap {
	return HelpKeyMap{
		Back: key.NewBinding(
			key.WithKeys("esc", "?", "backspace", "enter"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// HelpView represents the help view
type HelpView struct {
	width  int
	height int
	keyMap HelpKeyMap
}

// NewHelpView creates a new help view
func NewHelpView() *HelpView {
	return &HelpView{
		keyMap: DefaultHelpKeyMap(),
	}
}

// Init initializes the view
func (v *HelpView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Back):
			return v, func() tea.Msg {
				return messages.ChangeViewMsg{View: messages.ViewList}
			}
		case key.Matches(msg, v.keyMap.Quit):
			return v, tea.Quit
		}

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}

	return v, nil
}

// View renders the view
func (v *HelpView) View() string {
	var b strings.Builder

	// Header
	headerLeft := "  Help"
	headerRight := "[Esc] Back"
	headerPadding := v.width - len(headerLeft) - len(headerRight)
	if headerPadding < 2 {
		headerPadding = 2
	}
	header := styles.HeaderStyle.Width(v.width).Render(
		headerLeft + strings.Repeat(" ", headerPadding) + headerRight)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Navigation section
	b.WriteString(styles.SectionHeaderStyle.Render("  NAVIGATION"))
	b.WriteString("\n")
	b.WriteString(v.renderKeyBinding("↑/k", "Move cursor up"))
	b.WriteString(v.renderKeyBinding("↓/j", "Move cursor down"))
	b.WriteString(v.renderKeyBinding("Enter/Space", "Toggle selected tunnel"))
	b.WriteString("\n")

	// Actions section
	b.WriteString(styles.SectionHeaderStyle.Render("  ACTIONS"))
	b.WriteString("\n")
	b.WriteString(v.renderKeyBinding("a", "Start all tunnels"))
	b.WriteString(v.renderKeyBinding("s", "Stop all tunnels"))
	b.WriteString(v.renderKeyBinding("r", "Reload configuration"))
	b.WriteString("\n")

	// Views section
	b.WriteString(styles.SectionHeaderStyle.Render("  VIEWS"))
	b.WriteString("\n")
	b.WriteString(v.renderKeyBinding("g", "Groups view"))
	b.WriteString(v.renderKeyBinding("?", "This help screen"))
	b.WriteString("\n")

	// Other section
	b.WriteString(styles.SectionHeaderStyle.Render("  OTHER"))
	b.WriteString("\n")
	b.WriteString(v.renderKeyBinding("Esc", "Go back / Cancel"))
	b.WriteString(v.renderKeyBinding("q/Ctrl+C", "Quit application"))
	b.WriteString("\n")

	// Status indicators section
	b.WriteString(styles.SectionHeaderStyle.Render("  STATUS INDICATORS"))
	b.WriteString("\n")
	b.WriteString(v.renderIndicator(styles.StatusRunning.String(), "Connected/Running"))
	b.WriteString(v.renderIndicator(styles.StatusStopped.String(), "Stopped"))
	b.WriteString(v.renderIndicator(styles.StatusConnecting.String(), "Connecting"))
	b.WriteString(v.renderIndicator(styles.StatusReconnecting.String(), "Reconnecting"))
	b.WriteString(v.renderIndicator(styles.StatusError.String(), "Error"))
	b.WriteString("\n")

	// Tunnel types section
	b.WriteString(styles.SectionHeaderStyle.Render("  TUNNEL TYPES"))
	b.WriteString("\n")
	b.WriteString(v.renderType("-L", "Local port forwarding (forward local port to remote)"))
	b.WriteString(v.renderType("-R", "Remote port forwarding (forward remote port to local)"))
	b.WriteString(v.renderType("-D", "Dynamic SOCKS5 proxy"))
	b.WriteString("\n")

	// Footer
	b.WriteString(styles.MutedStyle.Render("\n  Press Esc or ? to return to the main view"))

	return b.String()
}

// renderKeyBinding renders a key binding line
func (v *HelpView) renderKeyBinding(key, desc string) string {
	keyStyle := styles.HelpKeyStyle.Width(16).Render(key)
	descStyle := styles.HelpDescStyle.Render(desc)
	return "  " + keyStyle + descStyle + "\n"
}

// renderIndicator renders a status indicator line
func (v *HelpView) renderIndicator(indicator, desc string) string {
	return "  " + indicator + "  " + styles.HelpDescStyle.Render(desc) + "\n"
}

// renderType renders a tunnel type line
func (v *HelpView) renderType(typeStr, desc string) string {
	typeStyle := styles.TunnelTypeStyle.Width(6).Render(typeStr)
	descStyle := styles.HelpDescStyle.Render(desc)
	return "  " + typeStyle + descStyle + "\n"
}
