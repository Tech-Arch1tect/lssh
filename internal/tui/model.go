package tui

import (
	"context"
	"fmt"
	"strings"

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
	providers      []provider.Provider
	groups         []*types.Group
	hosts          []*types.Host
	filteredHosts  []*types.Host
	filteredGroups []*types.Group
	currentGroup   *types.Group
	viewMode       ViewMode
	cursorRow      int
	cursorCol      int
	gridCols       int
	terminalWidth  int
	terminalHeight int
	selected       map[int]struct{}
	choice         *types.Host
	err            error
	quitting       bool
	loading        bool
	breadcrumb     []string
	filterMode     bool
	filterText     string
}

type dataLoadedMsg struct {
	groups []*types.Group
	hosts  []*types.Host
	err    error
}

func NewModel(providers []provider.Provider) Model {
	return Model{
		providers:      providers,
		selected:       make(map[int]struct{}),
		loading:        true,
		viewMode:       AllHostsView,
		breadcrumb:     []string{"All Hosts"},
		gridCols:       3,
		terminalWidth:  80,
		terminalHeight: 24,
		filterMode:     false,
		filterText:     "",
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
		if m.filterMode {
			return m.handleFilterInput(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			return m.moveUp()

		case "down", "j":
			return m.moveDown()

		case "left", "h":
			if m.viewMode == HostView {
				return m.backToGroups()
			}
			return m.moveLeft()

		case "right", "l":
			return m.moveRight()

		case "enter", " ":
			if m.viewMode == GroupView {
				return m.enterGroup()
			} else {
				return m.selectHost()
			}

		case "backspace":
			if m.viewMode == HostView {
				return m.backToGroups()
			}

		case "tab":
			return m.switchView()

		case "/":
			m.filterMode = true
			return m, nil

		case "esc":
			if m.filterText != "" {
				m.filterText = ""
				m.updateFilteredData()
				return m, nil
			}
		}

	case dataLoadedMsg:
		m.loading = false
		m.groups = msg.groups
		m.hosts = msg.hosts
		m.err = msg.err
		m.updateFilteredData()
	}

	return m, nil
}

func (m *Model) updateFilteredData() {
	if m.filterText == "" {
		m.filteredHosts = m.hosts
		m.filteredGroups = m.groups
		return
	}

	filterLower := strings.ToLower(m.filterText)

	m.filteredHosts = nil
	for _, host := range m.hosts {
		if strings.Contains(strings.ToLower(host.Name), filterLower) ||
			strings.Contains(strings.ToLower(host.Hostname), filterLower) {
			m.filteredHosts = append(m.filteredHosts, host)
		}
	}

	m.filteredGroups = nil
	for _, group := range m.groups {
		hasMatchingHost := false
		for _, host := range group.AllHosts() {
			if strings.Contains(strings.ToLower(host.Name), filterLower) ||
				strings.Contains(strings.ToLower(host.Hostname), filterLower) {
				hasMatchingHost = true
				break
			}
		}
		if hasMatchingHost || strings.Contains(strings.ToLower(group.Name), filterLower) {
			m.filteredGroups = append(m.filteredGroups, group)
		}
	}

	m.resetCursor()
}

func (m *Model) resetCursor() {
	m.cursorRow = 0
	m.cursorCol = 0
}

func (m Model) getCurrentItems() interface{} {
	switch m.viewMode {
	case AllHostsView:
		if m.currentGroup != nil {
			return m.getFilteredGroupHosts()
		}
		return m.filteredHosts
	case GroupView:
		return m.filteredGroups
	case HostView:
		return m.getFilteredGroupHosts()
	default:
		return nil
	}
}

func (m Model) getFilteredGroupHosts() []*types.Host {
	if m.currentGroup == nil {
		return nil
	}

	if m.filterText == "" {
		return m.currentGroup.Hosts
	}

	filterLower := strings.ToLower(m.filterText)
	var filtered []*types.Host
	for _, host := range m.currentGroup.Hosts {
		if strings.Contains(strings.ToLower(host.Name), filterLower) ||
			strings.Contains(strings.ToLower(host.Hostname), filterLower) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

func (m Model) getCurrentItemCount() int {
	items := m.getCurrentItems()
	switch v := items.(type) {
	case []*types.Host:
		return len(v)
	case []*types.Group:
		return len(v)
	default:
		return 0
	}
}

func (m Model) getGridDimensions() (rows, cols int) {
	itemCount := m.getCurrentItemCount()
	if itemCount == 0 {
		return 0, 0
	}

	cols = m.gridCols
	if cols <= 0 {
		cols = 1
	}

	rows = (itemCount + cols - 1) / cols
	return rows, cols
}

func (m Model) getCurrentIndex() int {
	_, cols := m.getGridDimensions()
	if cols <= 0 {
		return 0
	}
	return m.cursorRow*cols + m.cursorCol
}

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filterMode = false
		m.updateFilteredData()
		return m, nil
	case "esc":
		m.filterMode = false
		return m, nil
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.updateFilteredData()
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.filterText += msg.String()
			m.updateFilteredData()
		}
		return m, nil
	}
}

func (m Model) moveUp() (tea.Model, tea.Cmd) {
	if m.cursorRow > 0 {
		m.cursorRow--
	}
	return m, nil
}

func (m Model) moveDown() (tea.Model, tea.Cmd) {
	rows, _ := m.getGridDimensions()
	if m.cursorRow < rows-1 {
		index := m.getCurrentIndex()
		itemCount := m.getCurrentItemCount()
		if index+m.gridCols < itemCount {
			m.cursorRow++
		}
	}
	return m, nil
}

func (m Model) moveLeft() (tea.Model, tea.Cmd) {
	if m.cursorCol > 0 {
		m.cursorCol--
	}
	return m, nil
}

func (m Model) moveRight() (tea.Model, tea.Cmd) {
	_, cols := m.getGridDimensions()
	if m.cursorCol < cols-1 {
		index := m.getCurrentIndex()
		itemCount := m.getCurrentItemCount()
		if index+1 < itemCount {
			m.cursorCol++
		}
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
	currentIndex := m.getCurrentIndex()
	if len(m.filteredGroups) == 0 || currentIndex >= len(m.filteredGroups) {
		return m, nil
	}

	selectedGroup := m.filteredGroups[currentIndex]
	m.currentGroup = selectedGroup
	m.viewMode = HostView
	m.resetCursor()
	m.breadcrumb = append(m.breadcrumb, selectedGroup.Name)

	return m, nil
}

func (m Model) selectHost() (tea.Model, tea.Cmd) {
	currentIndex := m.getCurrentIndex()
	var hosts []*types.Host

	switch m.viewMode {
	case AllHostsView:
		hosts = m.filteredHosts
	case HostView:
		hosts = m.getFilteredGroupHosts()
	}

	if len(hosts) > 0 && currentIndex < len(hosts) {
		m.choice = hosts[currentIndex]
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) switchView() (tea.Model, tea.Cmd) {
	m.resetCursor()

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

	m.resetCursor()
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

	if m.filterMode {
		s += fmt.Sprintf("Filter: %s_\n\n", m.filterText)
	} else if m.filterText != "" {
		s += fmt.Sprintf("Filter: %s (Press Esc to clear)\n\n", m.filterText)
	}

	switch m.viewMode {
	case AllHostsView:
		return m.renderGridView(s, m.filteredHosts, nil)
	case GroupView:
		return m.renderGridView(s, nil, m.filteredGroups)
	case HostView:
		hosts := m.getFilteredGroupHosts()
		return m.renderGridView(s, hosts, nil)
	default:
		return s + "Unknown view mode"
	}
}

func (m Model) renderGridView(header string, hosts []*types.Host, groups []*types.Group) string {
	s := header

	var items []string
	var itemCount int

	if hosts != nil {
		itemCount = len(hosts)
		for _, host := range hosts {
			items = append(items, fmt.Sprintf("%s (%s)", host.Name, host.Hostname))
		}
	} else if groups != nil {
		itemCount = len(groups)
		for _, group := range groups {
			hostCount := len(group.AllHosts())
			items = append(items, fmt.Sprintf("%s (%d hosts)", group.Name, hostCount))
		}
	}

	if itemCount == 0 {
		s += "No items available.\n"
		s += helpStyle.Render(m.getHelpText())
		return s
	}

	rows, cols := m.getGridDimensions()
	colWidths := m.calculateColumnWidths(items, rows, cols)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			index := row*cols + col
			if index >= itemCount {
				break
			}

			isSelected := (row == m.cursorRow && col == m.cursorCol)
			rawText := items[index]

			var displayText string
			if isSelected {
				displayText = "► " + rawText
			} else {
				displayText = "  " + rawText
			}

			contentWidth := len(rawText) + 2
			padding := colWidths[col] - contentWidth
			if padding < 0 {
				padding = 0
			}

			var styledText string
			if isSelected {
				styledText = selectedItemStyle.Render(displayText)
			} else {
				styledText = itemStyle.Render(displayText)
			}

			s += styledText

			if col < cols-1 && index+1 < itemCount {
				s += strings.Repeat(" ", padding+2)
			}
		}
		s += "\n"
	}

	s += "\n" + helpStyle.Render(m.getHelpText())
	return s
}

func (m Model) calculateColumnWidths(items []string, rows, cols int) []int {
	if len(items) == 0 || cols == 0 {
		return nil
	}

	colWidths := make([]int, cols)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			index := row*cols + col
			if index >= len(items) {
				break
			}

			itemWidth := len(items[index]) + 2
			if itemWidth > colWidths[col] {
				colWidths[col] = itemWidth
			}
		}
	}

	return colWidths
}

func (m Model) getHelpText() string {
	baseHelp := "↑↓←→/hjkl: navigate, Enter: select"

	if m.viewMode == HostView && len(m.breadcrumb) > 1 {
		baseHelp += ", Backspace: back"
	}

	baseHelp += ", Tab: switch view, /: filter"

	if m.filterText != "" {
		baseHelp += ", Esc: clear filter"
	}

	baseHelp += ", q: quit"
	return baseHelp
}

func (m Model) Choice() *types.Host {
	return m.choice
}
