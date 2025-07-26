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

type Model struct {
	providers []provider.Provider
	hosts     []*types.Host
	cursor    int
	selected  map[int]struct{}
	choice    *types.Host
	err       error
	quitting  bool
	loading   bool
}

type hostsLoadedMsg struct {
	hosts []*types.Host
	err   error
}

func NewModel(providers []provider.Provider) Model {
	return Model{
		providers: providers,
		selected:  make(map[int]struct{}),
		loading:   true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadHosts()
}

func (m Model) loadHosts() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var allHosts []*types.Host

		for _, p := range m.providers {
			hosts, err := p.GetHosts(context.Background())
			if err != nil {
				return hostsLoadedMsg{err: fmt.Errorf("failed to load hosts from %s: %w", p.Name(), err)}
			}
			allHosts = append(allHosts, hosts...)
		}

		return hostsLoadedMsg{hosts: allHosts}
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
			if m.cursor < len(m.hosts)-1 {
				m.cursor++
			}

		case "enter", " ":
			if len(m.hosts) > 0 && m.cursor < len(m.hosts) {
				m.choice = m.hosts[m.cursor]
				m.quitting = true
				return m, tea.Quit
			}
		}

	case hostsLoadedMsg:
		m.loading = false
		m.hosts = msg.hosts
		m.err = msg.err
	}

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
		return "Loading hosts...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.\n", m.err)
	}

	s := titleStyle.Render("LSSH - Select SSH Host")
	s += "\n\n"

	if len(m.hosts) == 0 {
		s += "No hosts available.\n"
		s += helpStyle.Render("Press q to quit.")
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

	s += "\n" + helpStyle.Render("Press ↑/↓ to navigate, Enter to connect, q to quit.")

	return s
}

func (m Model) Choice() *types.Host {
	return m.choice
}
