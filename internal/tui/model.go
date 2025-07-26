package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tech-arch1tect/lssh/internal/provider"
	"github.com/tech-arch1tect/lssh/pkg/types"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("241"))
)

type ViewMode int

const (
	AllHostsView ViewMode = iota
	GroupView
	HostView
)

type Model struct {
	providers    []provider.Provider
	groups       []*types.Group
	hosts        []*types.Host
	currentGroup *types.Group
	viewMode     ViewMode
	cursor       int
	selected     map[int]struct{}
	choice       *types.Host
	err          error
	quitting     bool
	loading      bool
	breadcrumb   []string
}

type dataLoadedMsg struct {
	groups []*types.Group
	hosts  []*types.Host
	err    error
}

func NewModel(providers []provider.Provider) Model {
	return Model{
		providers:  providers,
		selected:   make(map[int]struct{}),
		loading:    true,
		viewMode:   AllHostsView,
		breadcrumb: []string{"All Hosts"},
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadData()
}

func (m Model) loadData() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var allGroups []*types.Group
		var allHosts []*types.Host

		for _, p := range m.providers {
			groups, err := p.GetGroups(context.Background())
			if err != nil {
				return dataLoadedMsg{err: fmt.Errorf("failed to load data from %s: %w", p.Name(), err)}
			}

			allGroups = append(allGroups, groups...)

			for _, group := range groups {
				allHosts = append(allHosts, group.AllHosts()...)
			}
		}

		return dataLoadedMsg{groups: allGroups, hosts: allHosts}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxItems := m.getMaxCursor()
			if m.cursor < maxItems-1 {
				m.cursor++
			}

		case "enter", " ":
			if m.viewMode == GroupView {
				return m.enterGroup()
			} else {
				return m.selectHost()
			}

		case "backspace", "h":
			if m.viewMode == HostView {
				return m.backToGroups()
			}

		case "tab":
			return m.switchView()
		}

	case dataLoadedMsg:
		m.loading = false
		m.groups = msg.groups
		m.hosts = msg.hosts
		m.err = msg.err
	}

	return m, nil
}

func (m Model) getMaxCursor() int {
	switch m.viewMode {
	case AllHostsView:
		return len(m.hosts)
	case GroupView:
		return len(m.groups)
	case HostView:
		if m.currentGroup != nil {
			return len(m.currentGroup.Hosts)
		}
		return 0
	default:
		return 0
	}
}

func (m Model) enterGroup() (tea.Model, tea.Cmd) {
	if len(m.groups) == 0 || m.cursor >= len(m.groups) {
		return m, nil
	}

	selectedGroup := m.groups[m.cursor]
	m.currentGroup = selectedGroup
	m.viewMode = HostView
	m.cursor = 0
	m.breadcrumb = append(m.breadcrumb, selectedGroup.Name)

	return m, nil
}

func (m Model) selectHost() (tea.Model, tea.Cmd) {
	var hosts []*types.Host

	switch m.viewMode {
	case AllHostsView:
		hosts = m.hosts
	case HostView:
		if m.currentGroup != nil {
			hosts = m.currentGroup.Hosts
		}
	}

	if len(hosts) > 0 && m.cursor < len(hosts) {
		m.choice = hosts[m.cursor]
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) switchView() (tea.Model, tea.Cmd) {
	m.cursor = 0

	switch m.viewMode {
	case AllHostsView:
		m.viewMode = GroupView
		m.breadcrumb = []string{"All Groups"}
		m.currentGroup = nil
	case GroupView:
		m.viewMode = AllHostsView
		m.breadcrumb = []string{"All Hosts"}
		m.currentGroup = nil
	case HostView:
		m.viewMode = AllHostsView
		m.breadcrumb = []string{"All Hosts"}
		m.currentGroup = nil
	}

	return m, nil
}

func (m Model) backToGroups() (tea.Model, tea.Cmd) {
	if len(m.breadcrumb) > 1 {
		m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
	}

	if len(m.breadcrumb) == 1 && m.breadcrumb[0] == "All Groups" {
		m.viewMode = GroupView
		m.currentGroup = nil
	}

	m.cursor = 0
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		if m.choice != nil {
			return fmt.Sprintf("Connecting to %s...\n", m.choice.Name)
		}
		return "Goodbye!\n"
	}

	if m.loading {
		return "Loading data...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.\n", m.err)
	}

	s := titleStyle.Render("LSSH - SSH Host Manager")
	s += "\n\n"

	breadcrumbStr := ""
	for i, crumb := range m.breadcrumb {
		if i > 0 {
			breadcrumbStr += " > "
		}
		breadcrumbStr += crumb
	}
	s += helpStyle.Render(breadcrumbStr) + "\n\n"

	switch m.viewMode {
	case AllHostsView:
		return m.renderAllHostsView(s)
	case GroupView:
		return m.renderGroupView(s)
	case HostView:
		return m.renderHostView(s)
	default:
		return s + "Unknown view mode"
	}
}

func (m Model) renderAllHostsView(header string) string {
	s := header

	if len(m.hosts) == 0 {
		s += "No hosts available.\n"
		s += helpStyle.Render("Press Tab to switch to group view, q to quit.")
		return s
	}

	for i, host := range m.hosts {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		style := itemStyle
		if m.cursor == i {
			style = selectedItemStyle
		}

		hostLine := fmt.Sprintf("%s %s (%s)", cursor, host.Name, host.Hostname)
		s += style.Render(hostLine) + "\n"
	}

	s += "\n" + helpStyle.Render("Press ↑/↓ to navigate, Enter to connect, Tab to switch to group view, q to quit.")
	return s
}

func (m Model) renderGroupView(header string) string {
	s := header

	if len(m.groups) == 0 {
		s += "No groups available.\n"
		s += helpStyle.Render("Press q to quit.")
		return s
	}

	for i, group := range m.groups {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		style := itemStyle
		if m.cursor == i {
			style = selectedItemStyle
		}

		hostCount := len(group.AllHosts())
		groupLine := fmt.Sprintf("%s %s (%d hosts)", cursor, group.Name, hostCount)
		if group.Description != "" {
			groupLine += fmt.Sprintf(" - %s", group.Description)
		}

		s += style.Render(groupLine) + "\n"
	}

	s += "\n" + helpStyle.Render("Press ↑/↓ to navigate, Enter to enter group, Tab to switch to all hosts, q to quit.")
	return s
}

func (m Model) renderHostView(header string) string {
	s := header

	var hosts []*types.Host
	if m.currentGroup != nil {
		hosts = m.currentGroup.Hosts
	} else {
		hosts = m.hosts
	}

	if len(hosts) == 0 {
		s += "No hosts available in this group.\n"
		if m.viewMode == HostView && len(m.breadcrumb) > 1 {
			s += helpStyle.Render("Press h/backspace to go back, q to quit.")
		} else {
			s += helpStyle.Render("Press q to quit.")
		}
		return s
	}

	for i, host := range hosts {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		style := itemStyle
		if m.cursor == i {
			style = selectedItemStyle
		}

		hostLine := fmt.Sprintf("%s %s (%s)", cursor, host.Name, host.Hostname)

		s += style.Render(hostLine) + "\n"
	}

	helpText := "Press ↑/↓ to navigate, Enter to connect"
	if len(m.breadcrumb) > 1 {
		helpText += ", h/backspace to go back"
	}
	helpText += ", Tab to switch to all hosts, q to quit."

	s += "\n" + helpStyle.Render(helpText)
	return s
}

func (m Model) Choice() *types.Host {
	return m.choice
}
