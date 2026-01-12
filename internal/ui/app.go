// Package ui provides the terminal user interface using the BubbleTea framework.
// It includes views for tunnel management, groups, and help.
package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/labbs/sshtunnel/internal/config"
	"github.com/labbs/sshtunnel/internal/tunnel"
	"github.com/labbs/sshtunnel/internal/ui/messages"
	"github.com/labbs/sshtunnel/internal/ui/views"
)

// App represents the main application model
type App struct {
	manager    *tunnel.Manager
	configPath string

	currentView messages.ViewType
	listView    *views.ListView
	groupsView  *views.GroupsView
	helpView    *views.HelpView

	width  int
	height int

	err error
}

// NewApp creates a new application
func NewApp(manager *tunnel.Manager, configPath string) *App {
	return &App{
		manager:     manager,
		configPath:  configPath,
		currentView: messages.ViewList,
		listView:    views.NewListView(manager),
		groupsView:  views.NewGroupsView(manager),
		helpView:    views.NewHelpView(),
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.tickCmd(),
	)
}

// tickCmd returns a command that sends tick messages
func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return messages.TickMsg{}
	})
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Forward to all views
		a.listView.Update(msg)
		a.groupsView.Update(msg)
		a.helpView.Update(msg)

	case messages.ChangeViewMsg:
		a.currentView = msg.View

	case messages.ErrorMsg:
		a.err = msg.Error

	case messages.ReloadConfigMsg:
		cmds = append(cmds, a.reloadConfig())

	case messages.TickMsg:
		// Continue ticking
		cmds = append(cmds, a.tickCmd())
	}

	// Route to current view
	var cmd tea.Cmd
	switch a.currentView {
	case messages.ViewList:
		_, cmd = a.listView.Update(msg)
	case messages.ViewGroups:
		_, cmd = a.groupsView.Update(msg)
	case messages.ViewHelp:
		_, cmd = a.helpView.Update(msg)
	}

	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View renders the application
func (a *App) View() string {
	switch a.currentView {
	case messages.ViewList:
		return a.listView.View()
	case messages.ViewGroups:
		return a.groupsView.View()
	case messages.ViewHelp:
		return a.helpView.View()
	default:
		return a.listView.View()
	}
}

// reloadConfig reloads the configuration
func (a *App) reloadConfig() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load(a.configPath)
		if err != nil {
			return messages.ErrorMsg{Error: err}
		}

		if err := a.manager.Reload(cfg); err != nil {
			return messages.ErrorMsg{Error: err}
		}

		// Update views
		a.listView = views.NewListView(a.manager)
		a.groupsView = views.NewGroupsView(a.manager)

		return messages.RefreshMsg{}
	}
}

// Run starts the application
func Run(manager *tunnel.Manager, configPath string) error {
	app := NewApp(manager, configPath)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
