package tui

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tech-arch1tect/lssh/internal/provider"
	"github.com/tech-arch1tect/lssh/internal/ssh"
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

	detailsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(1, 2).
				MarginLeft(2)

	detailsHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#5A56E0")).
				Padding(0, 1)

	detailsLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170"))

	detailsValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
)

type ViewMode int

const (
	AllHostsView ViewMode = iota
	GroupView
	HostView
	BulkCommandView
)

type Model struct {
	providers         []provider.Provider
	groups            []*types.Group
	hosts             []*types.Host
	filteredHosts     []*types.Host
	filteredGroups    []*types.Group
	currentGroup      *types.Group
	viewMode          ViewMode
	cursorRow         int
	cursorCol         int
	gridCols          int
	terminalWidth     int
	terminalHeight    int
	selected          map[int]struct{}
	choice            *types.Host
	err               error
	quitting          bool
	loading           bool
	breadcrumb        []string
	filterMode        bool
	filterText        string
	usernameMode      bool
	usernameText      string
	customUsername    string
	bulkSelectionMode bool
	bulkCommandMode   bool
	bulkCommandText   string
	selectedHosts     []*types.Host
	bulkResults       map[string]*BulkCommandResult
	bulkOutputFile    string
}

type BulkCommandResult struct {
	Host   *types.Host
	Output string
	Error  error
	Done   bool
}

type dataLoadedMsg struct {
	groups []*types.Group
	hosts  []*types.Host
	err    error
}

type bulkCommandFinishedMsg struct {
	host   *types.Host
	output string
	err    error
}

func NewModel(providers []provider.Provider) Model {
	return newModelWithError(providers, nil)
}

func NewModelWithError(providers []provider.Provider, err error) Model {
	return newModelWithError(providers, err)
}

func newModelWithError(providers []provider.Provider, err error) Model {
	m := Model{
		providers:         providers,
		selected:          make(map[int]struct{}),
		loading:           true,
		viewMode:          AllHostsView,
		breadcrumb:        []string{"All Hosts"},
		gridCols:          3,
		terminalWidth:     80,
		terminalHeight:    24,
		filterMode:        false,
		filterText:        "",
		usernameMode:      false,
		usernameText:      "",
		bulkSelectionMode: false,
		bulkCommandMode:   false,
		bulkCommandText:   "",
		selectedHosts:     make([]*types.Host, 0),
		bulkResults:       make(map[string]*BulkCommandResult),
		bulkOutputFile:    "",
	}

	if err != nil {
		m.loading = false
		m.err = fmt.Errorf("connection error: %w", err)
	}

	return m
}

func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
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
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.err != nil {
			if msg.String() == "ctrl+c" || msg.String() == "q" {
				m.quitting = true
				return m, tea.Quit
			} else {
				m.err = nil
				m.loading = true
				return m, m.loadData()
			}
		}

		if m.filterMode {
			return m.handleFilterInput(msg)
		}

		if m.usernameMode {
			return m.handleUsernameInput(msg)
		}

		if m.bulkCommandMode {
			return m.handleBulkCommandInput(msg)
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
			if m.viewMode == HostView && m.cursorCol == 0 {
				return m.backToGroups()
			}
			return m.moveLeft()

		case "right", "l":
			return m.moveRight()

		case "enter", " ":
			if m.bulkSelectionMode && m.viewMode != GroupView {
				return m.toggleHostSelection()
			} else if m.viewMode == GroupView {
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

		case "u":
			if m.viewMode != GroupView {
				m.usernameMode = true
				m.usernameText = ""
				return m, nil
			}

		case "s":
			if m.viewMode != GroupView && m.viewMode != BulkCommandView {
				m.bulkSelectionMode = !m.bulkSelectionMode
				if !m.bulkSelectionMode {
					m.selectedHosts = make([]*types.Host, 0)
				}
				return m, nil
			}

		case "c":
			if m.bulkSelectionMode && len(m.selectedHosts) > 0 {
				m.bulkCommandMode = true
				m.bulkCommandText = ""
				return m, nil
			}

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
		if msg.err != nil {
			m.err = msg.err
		}
		m.updateFilteredData()

	case bulkCommandFinishedMsg:
		key := fmt.Sprintf("%s@%s", msg.host.Name, msg.host.Hostname)
		if result, exists := m.bulkResults[key]; exists {
			result.Output = msg.output
			result.Error = msg.err
			result.Done = true

			if err := m.saveBulkResult(msg.host, msg.output, msg.err); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save result to file: %v\n", err)
			}
		}
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

func (m Model) handleUsernameInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.usernameMode = false
		if m.usernameText != "" {
			m.customUsername = m.usernameText
			return m.selectHostWithUsername()
		}
		return m, nil
	case "esc":
		m.usernameMode = false
		m.usernameText = ""
		return m, nil
	case "backspace":
		if len(m.usernameText) > 0 {
			m.usernameText = m.usernameText[:len(m.usernameText)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.usernameText += msg.String()
		}
		return m, nil
	}
}

func (m Model) handleBulkCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.bulkCommandMode = false
		if m.bulkCommandText != "" {
			return m.executeBulkCommand()
		}
		return m, nil
	case "esc":
		m.bulkCommandMode = false
		m.bulkCommandText = ""
		return m, nil
	case "backspace":
		if len(m.bulkCommandText) > 0 {
			m.bulkCommandText = m.bulkCommandText[:len(m.bulkCommandText)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.bulkCommandText += msg.String()
		}
		return m, nil
	}
}

func (m Model) toggleHostSelection() (tea.Model, tea.Cmd) {
	currentIndex := m.getCurrentIndex()
	var hosts []*types.Host

	switch m.viewMode {
	case AllHostsView:
		hosts = m.filteredHosts
	case HostView:
		hosts = m.getFilteredGroupHosts()
	}

	if len(hosts) > 0 && currentIndex < len(hosts) {
		selectedHost := hosts[currentIndex]

		for i, host := range m.selectedHosts {
			if host.Name == selectedHost.Name && host.Hostname == selectedHost.Hostname {
				m.selectedHosts = append(m.selectedHosts[:i], m.selectedHosts[i+1:]...)
				return m, nil
			}
		}

		m.selectedHosts = append(m.selectedHosts, selectedHost)
	}

	return m, nil
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
		m.customUsername = ""
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) selectHostWithUsername() (tea.Model, tea.Cmd) {
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
	case BulkCommandView:
		m.viewMode = AllHostsView
		m.breadcrumb = []string{"All Hosts"}
		m.currentGroup = nil
		m.bulkSelectionMode = false
		m.selectedHosts = make([]*types.Host, 0)
		m.bulkResults = make(map[string]*BulkCommandResult)
		m.bulkOutputFile = ""
	}

	return m, nil
}

func (m Model) executeBulkCommand() (tea.Model, tea.Cmd) {
	if len(m.selectedHosts) == 0 {
		return m, nil
	}

	m.viewMode = BulkCommandView
	m.breadcrumb = []string{fmt.Sprintf("Bulk Command: %s", m.bulkCommandText)}
	m.bulkSelectionMode = false

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("lssh-bulk-%s.log", timestamp)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		m.err = fmt.Errorf("failed to get home directory: %w", err)
		return m, nil
	}

	lsshDir := filepath.Join(homeDir, ".lssh", "logs")
	err = os.MkdirAll(lsshDir, 0755)
	if err != nil {
		m.err = fmt.Errorf("failed to create logs directory: %w", err)
		return m, nil
	}

	m.bulkOutputFile = filepath.Join(lsshDir, filename)

	err = m.initializeBulkOutputFile()
	if err != nil {
		m.err = fmt.Errorf("failed to create output file: %w", err)
		return m, nil
	}

	m.bulkResults = make(map[string]*BulkCommandResult)
	for _, host := range m.selectedHosts {
		key := fmt.Sprintf("%s@%s", host.Name, host.Hostname)
		m.bulkResults[key] = &BulkCommandResult{
			Host: host,
			Done: false,
		}
	}

	commands := make([]tea.Cmd, len(m.selectedHosts))
	for i, host := range m.selectedHosts {
		commands[i] = m.executeBulkCommandOnHost(host, m.bulkCommandText)
	}

	return m, tea.Batch(commands...)
}

func (m Model) executeBulkCommandOnHost(host *types.Host, command string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		output, err := ssh.ExecuteCommand(context.Background(), host, command)
		return bulkCommandFinishedMsg{
			host:   host,
			output: output,
			err:    err,
		}
	})
}

func (m Model) isHostSelected(host *types.Host) bool {
	for _, selectedHost := range m.selectedHosts {
		if selectedHost.Name == host.Name && selectedHost.Hostname == host.Hostname {
			return true
		}
	}
	return false
}

func (m Model) initializeBulkOutputFile() error {
	file, err := os.Create(m.bulkOutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	header := fmt.Sprintf("LSSH Bulk Command Execution Log\n")
	header += fmt.Sprintf("================================\n")
	header += fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	header += fmt.Sprintf("Command: %s\n", m.bulkCommandText)
	header += fmt.Sprintf("Hosts: %d\n", len(m.selectedHosts))
	header += fmt.Sprintf("--------------------------------\n\n")

	_, err = file.WriteString(header)
	return err
}

func (m Model) saveBulkResult(host *types.Host, output string, execErr error) error {
	if m.bulkOutputFile == "" {
		return nil
	}

	file, err := os.OpenFile(m.bulkOutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().Format("15:04:05")
	hostHeader := fmt.Sprintf("[%s] %s (%s)\n", timestamp, host.Name, host.Hostname)
	result := hostHeader

	if execErr != nil {
		result += fmt.Sprintf("ERROR: %v\n", execErr)
	}

	if output != "" {
		result += fmt.Sprintf("OUTPUT:\n%s\n", output)
	}

	result += fmt.Sprintf("---\n\n")

	_, err = file.WriteString(result)
	return err
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
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196"))

		errorMsg := fmt.Sprintf("⚠ %v", m.err)

		return fmt.Sprintf("%s\n\n%s\n\n%s\n",
			titleStyle.Render("LSSH - SSH Host Manager"),
			errorStyle.Render(errorMsg),
			helpStyle.Render("Press any key to continue, q to quit"))
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

	if m.usernameMode {
		s += fmt.Sprintf("Enter username: %s_\n\n", m.usernameText)
	} else if m.bulkCommandMode {
		s += fmt.Sprintf("Enter command: %s_\n\n", m.bulkCommandText)
	} else if m.bulkSelectionMode {
		s += fmt.Sprintf("Bulk Selection Mode - %d hosts selected (Space: toggle, c: command)\n\n", len(m.selectedHosts))
	}

	switch m.viewMode {
	case AllHostsView:
		return m.renderGridView(s, m.filteredHosts, nil)
	case GroupView:
		return m.renderGridView(s, nil, m.filteredGroups)
	case HostView:
		hosts := m.getFilteredGroupHosts()
		return m.renderGridView(s, hosts, nil)
	case BulkCommandView:
		return m.renderBulkCommandView(s)
	default:
		return s + "Unknown view mode"
	}
}

func (m Model) renderGridView(header string, hosts []*types.Host, groups []*types.Group) string {
	detailsPanelWidth := 40
	availableWidth := m.terminalWidth - detailsPanelWidth - 6

	s := header

	var items []string
	var itemCount int

	if hosts != nil {
		itemCount = len(hosts)
		for _, host := range hosts {
			prefix := ""
			if m.bulkSelectionMode {
				if m.isHostSelected(host) {
					prefix = "[✓] "
				} else {
					prefix = "[ ] "
				}
			}
			items = append(items, fmt.Sprintf("%s%s (%s)", prefix, host.Name, host.Hostname))
		}
	} else if groups != nil {
		itemCount = len(groups)
		for _, group := range groups {
			hostCount := len(group.AllHosts())
			items = append(items, fmt.Sprintf("%s (%d hosts)", group.Name, hostCount))
		}
	}

	var gridContent string
	if itemCount == 0 {
		gridContent = "No items available.\n"
	} else {
		gridContent = m.renderGrid(items, itemCount, availableWidth)
	}

	detailsContent := m.renderHostDetails(m.getCurrentHost())

	gridLines := strings.Split(strings.TrimRight(gridContent, "\n"), "\n")
	detailsLines := strings.Split(strings.TrimRight(detailsContent, "\n"), "\n")

	maxLines := len(gridLines)
	if len(detailsLines) > maxLines {
		maxLines = len(detailsLines)
	}

	for i := 0; i < maxLines; i++ {
		var gridLine, detailsLine string

		if i < len(gridLines) {
			gridLine = gridLines[i]
		}
		if i < len(detailsLines) {
			detailsLine = detailsLines[i]
		}

		gridLineWidth := lipgloss.Width(gridLine)
		padding := availableWidth - gridLineWidth
		if padding < 0 {
			padding = 0
		}

		s += gridLine + strings.Repeat(" ", padding) + "  " + detailsLine + "\n"
	}

	s += "\n" + helpStyle.Render(m.getHelpText())
	return s
}

func (m Model) renderGrid(items []string, itemCount, availableWidth int) string {
	adjustedGridCols := m.gridCols
	if availableWidth < 120 {
		adjustedGridCols = 2
	}
	if availableWidth < 80 {
		adjustedGridCols = 1
	}

	rows := (itemCount + adjustedGridCols - 1) / adjustedGridCols
	colWidths := m.calculateColumnWidths(items, rows, adjustedGridCols, availableWidth)

	var content string
	for row := 0; row < rows; row++ {
		for col := 0; col < adjustedGridCols; col++ {
			index := row*adjustedGridCols + col
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

			content += styledText

			if col < adjustedGridCols-1 && index+1 < itemCount {
				content += strings.Repeat(" ", padding+2)
			}
		}
		content += "\n"
	}

	return content
}

func (m Model) calculateColumnWidths(items []string, rows, cols, maxWidth int) []int {
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

	totalWidth := 0
	for _, width := range colWidths {
		totalWidth += width
	}

	if totalWidth > maxWidth && cols > 1 {
		availablePerCol := maxWidth / cols
		for i := range colWidths {
			if colWidths[i] > availablePerCol {
				colWidths[i] = availablePerCol
			}
		}
	}

	return colWidths
}

func (m Model) getHelpText() string {
	if m.viewMode == BulkCommandView {
		return "Tab: back to hosts, q: quit"
	}

	baseHelp := "↑↓←→/hjkl: navigate"

	if m.bulkSelectionMode {
		baseHelp += ", Space: toggle selection"
		if len(m.selectedHosts) > 0 {
			baseHelp += ", c: enter command"
		}
	} else {
		baseHelp += ", Enter: select"
	}

	if m.viewMode != GroupView && m.viewMode != BulkCommandView {
		baseHelp += ", u: custom user, s: bulk mode"
	}

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

func (m Model) CustomUsername() string {
	return m.customUsername
}

func (m Model) getCurrentHost() *types.Host {
	currentIndex := m.getCurrentIndex()
	var hosts []*types.Host

	switch m.viewMode {
	case AllHostsView:
		hosts = m.filteredHosts
	case HostView:
		hosts = m.getFilteredGroupHosts()
	case GroupView:
		return nil
	}

	if len(hosts) > 0 && currentIndex < len(hosts) {
		return hosts[currentIndex]
	}
	return nil
}

func (m Model) renderHostDetails(host *types.Host) string {
	if host == nil {
		return detailsPanelStyle.Render("No host selected")
	}

	content := detailsHeaderStyle.Render("Connection Details") + "\n\n"

	content += detailsLabelStyle.Render("Name: ") + detailsValueStyle.Render(host.Name) + "\n"
	content += detailsLabelStyle.Render("Hostname: ") + detailsValueStyle.Render(host.Hostname) + "\n"

	port := "22"
	if host.Port > 0 {
		port = fmt.Sprintf("%d", host.Port)
	}
	content += detailsLabelStyle.Render("Port: ") + detailsValueStyle.Render(port) + "\n"

	username := host.User
	if username == "" {
		if currentUser, err := user.Current(); err == nil {
			username = currentUser.Username + " (current user)"
		} else {
			username = "(current user)"
		}
	}
	content += detailsLabelStyle.Render("User: ") + detailsValueStyle.Render(username) + "\n\n"

	content += detailsLabelStyle.Render("SSH Command:") + "\n"
	content += detailsValueStyle.Render(host.SSHCommand())

	return detailsPanelStyle.Render(content)
}

func (m Model) renderBulkCommandView(header string) string {
	s := header
	s += fmt.Sprintf("Command: %s\n", m.bulkCommandText)
	s += fmt.Sprintf("Hosts: %d\n", len(m.selectedHosts))

	if m.bulkOutputFile != "" {
		outputFileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
		s += fmt.Sprintf("Output: %s\n", outputFileStyle.Render(m.bulkOutputFile))
	}
	s += "\n"

	completedCount := 0
	for _, result := range m.bulkResults {
		if result.Done {
			completedCount++
		}
	}
	s += fmt.Sprintf("Progress: %d/%d completed\n\n", completedCount, len(m.bulkResults))

	for _, host := range m.selectedHosts {
		key := fmt.Sprintf("%s@%s", host.Name, host.Hostname)
		result, exists := m.bulkResults[key]

		hostHeader := fmt.Sprintf("=== %s ===", host.Name)
		s += lipgloss.NewStyle().Bold(true).Render(hostHeader) + "\n"

		if !exists {
			s += "Initializing...\n\n"
			continue
		}

		if !result.Done {
			s += "Running...\n\n"
			continue
		}

		if result.Error != nil {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			s += errorStyle.Render(fmt.Sprintf("Error: %v", result.Error)) + "\n"
		}

		if result.Output != "" {
			outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
			lines := strings.Split(result.Output, "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					s += outputStyle.Render(line) + "\n"
				}
			}
		}
		s += "\n"
	}

	s += "\n" + helpStyle.Render("Tab: back to hosts, q: quit")
	return s
}
