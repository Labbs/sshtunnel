// Package views implements the different screens of the terminal UI.
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/labbs/sshtunnel/internal/tunnel"
	"github.com/labbs/sshtunnel/internal/ui/components"
	"github.com/labbs/sshtunnel/internal/ui/messages"
	"github.com/labbs/sshtunnel/internal/ui/styles"
)

// ListKeyMap defines the key bindings for the list view
type ListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	StartAll key.Binding
	StopAll  key.Binding
	Groups   key.Binding
	Reload   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultListKeyMap returns the default key bindings
func DefaultListKeyMap() ListKeyMap {
	return ListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "toggle"),
		),
		StartAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "start all"),
		),
		StopAll: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop all"),
		),
		Groups: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "groups"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ListView represents the main tunnel list view
type ListView struct {
	manager   *tunnel.Manager
	cursor    int
	width     int
	height    int
	keyMap    ListKeyMap
	statusBar *components.StatusBar
	filter    string
}

// NewListView creates a new list view
func NewListView(manager *tunnel.Manager) *ListView {
	return &ListView{
		manager:   manager,
		keyMap:    DefaultListKeyMap(),
		statusBar: components.NewStatusBar(),
	}
}

// Init initializes the view
func (v *ListView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keyMap.Down):
			tunnels := v.getFilteredTunnels()
			if v.cursor < len(tunnels)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Toggle):
			tunnels := v.getFilteredTunnels()
			if v.cursor < len(tunnels) {
				return v, v.toggleTunnel(tunnels[v.cursor].Name())
			}
		case key.Matches(msg, v.keyMap.StartAll):
			return v, v.startAll()
		case key.Matches(msg, v.keyMap.StopAll):
			return v, v.stopAll()
		case key.Matches(msg, v.keyMap.Groups):
			return v, func() tea.Msg {
				return messages.ChangeViewMsg{View: messages.ViewGroups}
			}
		case key.Matches(msg, v.keyMap.Help):
			return v, func() tea.Msg {
				return messages.ChangeViewMsg{View: messages.ViewHelp}
			}
		case key.Matches(msg, v.keyMap.Reload):
			return v, func() tea.Msg {
				return messages.ReloadConfigMsg{}
			}
		case key.Matches(msg, v.keyMap.Quit):
			return v, tea.Quit
		}

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.statusBar.SetWidth(msg.Width)

	case messages.TickMsg:
		// Validate cursor bounds after potential filter changes
		tunnels := v.getFilteredTunnels()
		if v.cursor >= len(tunnels) {
			v.cursor = len(tunnels) - 1
		}
		if v.cursor < 0 {
			v.cursor = 0
		}
		return v, nil
	}

	return v, nil
}

// View renders the view
func (v *ListView) View() string {
	// Ensure minimum width
	width := v.width
	if width < 40 {
		width = 80
	}

	var b strings.Builder

	// Header
	headerLeft := "  SSH Tunnel Manager"
	headerRight := "[?] Help  [q] Quit"
	headerPadding := width - len(headerLeft) - len(headerRight)
	if headerPadding < 2 {
		headerPadding = 2
	}
	header := styles.HeaderStyle.Width(width).Render(
		headerLeft + strings.Repeat(" ", headerPadding) + headerRight)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tunnel count
	running, total, _, _ := v.manager.GetStats()
	tunnelLeft := "  TUNNELS"
	tunnelRight := fmt.Sprintf("%d active / %d", running, total)
	tunnelPadding := width - len(tunnelLeft) - len(tunnelRight) - 2
	if tunnelPadding < 2 {
		tunnelPadding = 2
	}
	countStr := styles.SectionHeaderStyle.Render(
		tunnelLeft + strings.Repeat(" ", tunnelPadding) + tunnelRight)
	b.WriteString(countStr)
	b.WriteString("\n")

	separatorWidth := width - 4
	if separatorWidth < 1 {
		separatorWidth = 1
	}
	b.WriteString(styles.MutedStyle.Render("  " + strings.Repeat("─", separatorWidth)))
	b.WriteString("\n")

	// Tunnels
	tunnels := v.getFilteredTunnels()
	for i, t := range tunnels {
		item := components.NewTunnelItem(t)
		item.SetSelected(i == v.cursor)
		item.SetWidth(width - 2)
		b.WriteString(item.Render())
		b.WriteString("\n")
	}

	// Groups section
	groups := v.manager.GetGroups()
	if len(groups) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.SectionHeaderStyle.Render("  GROUPS"))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("  " + strings.Repeat("─", separatorWidth)))
		b.WriteString("\n  ")

		for i, group := range groups {
			active, total := v.manager.GetGroupStatus(group.Name)
			var indicator string
			if active == total && total > 0 {
				indicator = styles.StatusRunning.String()
			} else if active > 0 {
				indicator = styles.StatusReconnecting.String()
			} else {
				indicator = styles.StatusStopped.String()
			}
			groupStr := fmt.Sprintf("[%s] %d/%d %s", group.Name, active, total, indicator)
			b.WriteString(styles.GroupStyle.Render(groupStr))
			if i < len(groups)-1 {
				b.WriteString("    ")
			}
		}
		b.WriteString("\n")
	}

	// Calculate available height for content
	height := v.height
	if height < 10 {
		height = 24
	}

	contentHeight := lipgloss.Height(b.String())
	availableHeight := height - contentHeight - 4 // 4 for help and status bar

	// Add padding
	if availableHeight > 0 {
		b.WriteString(strings.Repeat("\n", availableHeight))
	}

	// Help line
	helpLine := v.renderHelpLine()
	b.WriteString(helpLine)
	b.WriteString("\n")

	// Status bar
	v.statusBar.SetWidth(width)
	_, _, bytesSent, bytesReceived := v.manager.GetStats()
	b.WriteString(v.statusBar.Render(running, total, bytesSent, bytesReceived))

	return b.String()
}

// renderHelpLine renders the help line at the bottom
func (v *ListView) renderHelpLine() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"Enter", "Toggle"},
		{"a", "Start All"},
		{"s", "Stop All"},
		{"g", "Groups"},
		{"r", "Reload"},
		{"?", "Help"},
	}

	var parts []string
	for _, k := range keys {
		part := fmt.Sprintf("%s %s",
			styles.HelpKeyStyle.Render("["+k.key+"]"),
			styles.HelpDescStyle.Render(k.desc))
		parts = append(parts, part)
	}

	return "  " + strings.Join(parts, "  ")
}

// getFilteredTunnels returns tunnels filtered by the current filter
func (v *ListView) getFilteredTunnels() []tunnel.Tunnel {
	tunnels := v.manager.GetTunnels()
	if v.filter == "" {
		return tunnels
	}

	var filtered []tunnel.Tunnel
	filter := strings.ToLower(v.filter)
	for _, t := range tunnels {
		if strings.Contains(strings.ToLower(t.Name()), filter) ||
			strings.Contains(strings.ToLower(t.Config().Description), filter) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// toggleTunnel returns a command to toggle a tunnel
func (v *ListView) toggleTunnel(name string) tea.Cmd {
	return func() tea.Msg {
		err := v.manager.Toggle(name)
		if err != nil {
			return messages.ErrorMsg{Error: err}
		}
		return messages.RefreshMsg{}
	}
}

// startAll returns a command to start all tunnels
func (v *ListView) startAll() tea.Cmd {
	return func() tea.Msg {
		v.manager.StartAll()
		return messages.RefreshMsg{}
	}
}

// stopAll returns a command to stop all tunnels
func (v *ListView) stopAll() tea.Cmd {
	return func() tea.Msg {
		v.manager.StopAll()
		return messages.RefreshMsg{}
	}
}

// SetFilter sets the filter
func (v *ListView) SetFilter(filter string) {
	v.filter = filter
	v.cursor = 0
}

// SelectedTunnel returns the currently selected tunnel
func (v *ListView) SelectedTunnel() tunnel.Tunnel {
	tunnels := v.getFilteredTunnels()
	if v.cursor < len(tunnels) {
		return tunnels[v.cursor]
	}
	return nil
}
