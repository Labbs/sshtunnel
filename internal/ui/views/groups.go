package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/tunnel"
	"github.com/labbs/sshtunnel/internal/ui/messages"
	"github.com/labbs/sshtunnel/internal/ui/styles"
)

// GroupsKeyMap defines the key bindings for the groups view
type GroupsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Start  key.Binding
	Stop   key.Binding
	Back   key.Binding
	Quit   key.Binding
}

// DefaultGroupsKeyMap returns the default key bindings
func DefaultGroupsKeyMap() GroupsKeyMap {
	return GroupsKeyMap{
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
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start group"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop group"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// GroupsView represents the groups view
type GroupsView struct {
	manager *tunnel.Manager
	groups  []config.GroupConfig
	cursor  int
	width   int
	height  int
	keyMap  GroupsKeyMap
}

// NewGroupsView creates a new groups view
func NewGroupsView(manager *tunnel.Manager) *GroupsView {
	return &GroupsView{
		manager: manager,
		groups:  manager.GetGroups(),
		keyMap:  DefaultGroupsKeyMap(),
	}
}

// Init initializes the view
func (v *GroupsView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *GroupsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keyMap.Down):
			if v.cursor < len(v.groups)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Toggle), key.Matches(msg, v.keyMap.Start):
			if v.cursor < len(v.groups) {
				return v, v.startGroup(v.groups[v.cursor].Name)
			}
		case key.Matches(msg, v.keyMap.Stop):
			if v.cursor < len(v.groups) {
				return v, v.stopGroup(v.groups[v.cursor].Name)
			}
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

	case messages.RefreshMsg:
		v.groups = v.manager.GetGroups()
	}

	return v, nil
}

// View renders the view
func (v *GroupsView) View() string {
	var b strings.Builder

	// Header
	header := styles.HeaderStyle.Width(v.width).Render(
		"  Groups                                                     [Esc] Back")
	b.WriteString(header)
	b.WriteString("\n\n")

	if len(v.groups) == 0 {
		b.WriteString(styles.MutedStyle.Render("  No groups configured"))
		return b.String()
	}

	// Groups list
	for i, group := range v.groups {
		active, total := v.manager.GetGroupStatus(group.Name)

		// Status indicator
		var indicator string
		if active == total && total > 0 {
			indicator = styles.StatusRunning.String()
		} else if active > 0 {
			indicator = styles.StatusReconnecting.String()
		} else {
			indicator = styles.StatusStopped.String()
		}

		// Group name and status
		name := styles.TunnelNameStyle.Render(group.Name)
		status := styles.StatsStyle.Render(fmt.Sprintf("%d/%d active", active, total))

		line := fmt.Sprintf("%s %s    %s", indicator, name, status)

		// Description
		var descLine string
		if group.Description != "" {
			descLine = styles.DimStyle.Render("  " + group.Description)
		}

		// Tunnels in group
		tunnelList := strings.Join(group.Tunnels, ", ")
		if len(tunnelList) > 60 {
			tunnelList = tunnelList[:57] + "..."
		}
		tunnelsLine := styles.MutedStyle.Render("  Tunnels: " + tunnelList)

		var content string
		if descLine != "" {
			content = line + "\n" + descLine + "\n" + tunnelsLine
		} else {
			content = line + "\n" + tunnelsLine
		}

		if i == v.cursor {
			b.WriteString(styles.SelectedStyle.Width(v.width - 2).Render(content))
		} else {
			b.WriteString(styles.NormalStyle.Width(v.width - 2).Render(content))
		}
		b.WriteString("\n\n")
	}

	// Help line
	helpLine := v.renderHelpLine()
	b.WriteString("\n")
	b.WriteString(helpLine)

	return b.String()
}

// renderHelpLine renders the help line
func (v *GroupsView) renderHelpLine() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"Enter", "Start Group"},
		{"x", "Stop Group"},
		{"Esc", "Back"},
		{"q", "Quit"},
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

// startGroup returns a command to start a group
func (v *GroupsView) startGroup(name string) tea.Cmd {
	return func() tea.Msg {
		v.manager.StartGroup(name)
		return messages.RefreshMsg{}
	}
}

// stopGroup returns a command to stop a group
func (v *GroupsView) stopGroup(name string) tea.Cmd {
	return func() tea.Msg {
		v.manager.StopGroup(name)
		return messages.RefreshMsg{}
	}
}
