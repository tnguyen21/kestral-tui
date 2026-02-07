package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tnguyen21/kestral-tui/internal/config"
	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/pane"
	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// Model is the root bubbletea Model that orchestrates panes, tab bar,
// status bar, and background polling.
type Model struct {
	panes       []pane.Pane
	activePane  int
	width       int
	height      int
	layoutMode  LayoutMode
	keys        KeyMap
	fetcher     *data.Fetcher
	config      *config.Config
	help        help.Model
	showHelp    bool
	lastRefresh time.Time
}

// New creates a root Model with the given config.
func New(cfg config.Config) Model {
	fetcher := &data.Fetcher{TownRoot: cfg.TownRoot}
	panes := []pane.Pane{
		pane.NewDashboard(),
		pane.NewAgentsPane(),
		pane.NewPRsPane(),
	}

	return Model{
		panes:   panes,
		keys:    DefaultKeyMap(),
		fetcher: fetcher,
		config:  &cfg,
		help:    help.New(),
	}
}

// ShortHelp implements help.KeyMap for the application key bindings.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Tab, k.Help}
}

// FullHelp implements help.KeyMap for the application key bindings.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Tab, k.ShiftTab},
		{k.Pane1, k.Pane2, k.Pane3, k.Pane4},
		{k.Up, k.Down, k.Select, k.Back},
		{k.Refresh, k.Help},
	}
}

// Ensure KeyMap satisfies help.KeyMap at compile time.
var _ help.KeyMap = KeyMap{}

// Init starts the initial data fetches.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchStatusCmd(m.fetcher),
		fetchAgentsCmd(m.fetcher),
		fetchConvoysCmd(m.fetcher),
		fetchPRsCmd(m.fetcher),
	)
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutMode = GetLayoutMode(msg.Width)
		m.help.Width = msg.Width
		contentH := ContentHeight(msg.Height)
		for i, p := range m.panes {
			p.SetSize(msg.Width, contentH)
			m.panes[i] = p
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	// Tick messages trigger background fetches.
	case data.StatusTickMsg:
		return m, fetchStatusCmd(m.fetcher)
	case data.AgentTickMsg:
		return m, fetchAgentsCmd(m.fetcher)
	case data.ConvoyTickMsg:
		return m, fetchConvoysCmd(m.fetcher)
	case data.PRTickMsg:
		return m, fetchPRsCmd(m.fetcher)

	// Data update messages — forward to all panes and schedule next poll.
	case pane.StatusUpdateMsg:
		m.lastRefresh = msg.FetchedAt
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleStatusPoll(
			time.Duration(m.config.PollInterval.Status)*time.Second))
		return m, tea.Batch(cmds...)

	case pane.AgentUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleAgentPoll(
			time.Duration(m.config.PollInterval.Agents)*time.Second))
		return m, tea.Batch(cmds...)

	case pane.ConvoyUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleConvoyPoll(
			time.Duration(m.config.PollInterval.Convoys)*time.Second))
		return m, tea.Batch(cmds...)

	case pane.PRUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.SchedulePRPoll(
			time.Duration(m.config.PollInterval.PRs)*time.Second))
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// View renders the full UI: tab bar, active pane content, and status bar.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	tabBar := m.renderTabBar()
	statusBar := m.renderStatusBar()

	var content string
	if m.showHelp {
		content = m.help.View(m.keys)
	} else if m.activePane < len(m.panes) {
		content = m.panes[m.activePane].View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, statusBar)
}

// handleKey processes global key bindings, forwarding unhandled keys
// to the active pane.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		m.activePane = (m.activePane + 1) % len(m.panes)
		return m, nil

	case key.Matches(msg, m.keys.ShiftTab):
		m.activePane = (m.activePane - 1 + len(m.panes)) % len(m.panes)
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, tea.Batch(
			fetchStatusCmd(m.fetcher),
			fetchAgentsCmd(m.fetcher),
			fetchConvoysCmd(m.fetcher),
			fetchPRsCmd(m.fetcher),
		)
	}

	// Number keys for direct pane switching.
	if idx, ok := m.paneKeyIndex(msg); ok {
		m.activePane = idx
		return m, nil
	}

	// Forward to active pane.
	return m.updateActivePane(msg)
}

// paneKeyIndex returns the pane index if msg matches a pane number key.
func (m Model) paneKeyIndex(msg tea.KeyMsg) (int, bool) {
	paneKeys := []key.Binding{
		m.keys.Pane1, m.keys.Pane2, m.keys.Pane3,
		m.keys.Pane4, m.keys.Pane5, m.keys.Pane6, m.keys.Pane7,
	}
	for i, k := range paneKeys {
		if key.Matches(msg, k) && i < len(m.panes) {
			return i, true
		}
	}
	return 0, false
}

// handleMouse processes mouse events, detecting tab bar clicks.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		if msg.Y == 0 { // Tab bar row
			if idx := m.tabAtX(msg.X); idx >= 0 {
				m.activePane = idx
				return m, nil
			}
		}
	}

	// Forward to active pane.
	return m.updateActivePane(msg)
}

// updateActivePane sends a message to the active pane and stores the result.
func (m Model) updateActivePane(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.activePane >= len(m.panes) {
		return m, nil
	}
	newModel, cmd := m.panes[m.activePane].Update(msg)
	if newPane, ok := newModel.(pane.Pane); ok {
		m.panes[m.activePane] = newPane
	}
	return m, cmd
}

// forwardToAllPanes sends a message to every pane and collects commands.
func (m *Model) forwardToAllPanes(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	for i, p := range m.panes {
		newModel, cmd := p.Update(msg)
		if newPane, ok := newModel.(pane.Pane); ok {
			m.panes[i] = newPane
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// ---------------------------------------------------------------------------
// Tab bar
// ---------------------------------------------------------------------------

// renderTabBar renders the tab bar across the top of the screen.
func (m Model) renderTabBar() string {
	var parts []string

	for i, p := range m.panes {
		label := m.tabLabel(p)
		style := theme.TabInactiveStyle
		if i == m.activePane {
			style = theme.TabActiveStyle.Underline(true)
		}
		parts = append(parts, style.Render(label))
	}

	if m.layoutMode == LayoutWide {
		sep := theme.MutedStyle.Render("|")
		return strings.Join(parts, " "+sep+" ")
	}
	return strings.Join(parts, "")
}

// tabLabel returns the display label for a pane tab in the current layout mode.
func (m Model) tabLabel(p pane.Pane) string {
	var label string
	switch m.layoutMode {
	case LayoutNarrow:
		label = p.ShortTitle()
	default:
		label = p.Title()
	}
	if badge := p.Badge(); badge > 0 {
		label += fmt.Sprintf("(%d)", badge)
	}
	return label
}

// tabAtX returns the pane index whose tab contains column x, or -1.
func (m Model) tabAtX(x int) int {
	pos := 0
	for i, p := range m.panes {
		// Account for separator in wide mode.
		if m.layoutMode == LayoutWide && i > 0 {
			sepWidth := 1 + lipgloss.Width(theme.MutedStyle.Render("|")) + 1 // " | "
			pos += sepWidth
		}

		label := m.tabLabel(p)
		style := theme.TabInactiveStyle
		if i == m.activePane {
			style = theme.TabActiveStyle.Underline(true)
		}
		tabWidth := lipgloss.Width(style.Render(label))

		if x >= pos && x < pos+tabWidth {
			return i
		}
		pos += tabWidth
	}
	return -1
}

// ---------------------------------------------------------------------------
// Status bar
// ---------------------------------------------------------------------------

// renderStatusBar renders the bottom status bar.
func (m Model) renderStatusBar() string {
	health := theme.PassStyle.Render("⚡ healthy")

	age := "…"
	if !m.lastRefresh.IsZero() {
		age = pane.FormatAge(time.Since(m.lastRefresh))
	}
	refresh := theme.MutedStyle.Render("↻ " + age)

	keys := theme.MutedStyle.Render("?=help  q=quit")

	bar := strings.Join([]string{health, refresh, keys}, "  |  ")
	return theme.StatusBarStyle.Width(m.width).Render(bar)
}

// ---------------------------------------------------------------------------
// Fetch commands — bridge between poller ticks and pane-level messages
// ---------------------------------------------------------------------------

// fetchStatusCmd fetches town status + sessions and returns a pane.StatusUpdateMsg.
func fetchStatusCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		status, _ := f.FetchStatus()
		sessions, _ := f.FetchSessions()
		return pane.StatusUpdateMsg{
			Status:    status,
			Sessions:  sessions,
			FetchedAt: time.Now(),
		}
	}
}

// fetchAgentsCmd fetches enriched agent details and returns a pane.AgentUpdateMsg.
func fetchAgentsCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		details, err := f.FetchAgents()
		agents := make([]pane.AgentInfo, len(details))
		for i, d := range details {
			agents[i] = pane.AgentInfo{
				Name:       d.Name,
				Rig:        d.Rig,
				Role:       d.Role,
				Status:     d.Status,
				Age:        time.Duration(d.AgeSecs) * time.Second,
				IssueID:    d.IssueID,
				IssueTitle: d.IssueTitle,
			}
		}
		return pane.AgentUpdateMsg{Agents: agents, Err: err}
	}
}

// fetchConvoysCmd fetches convoy data with progress and returns a pane.ConvoyUpdateMsg.
func fetchConvoysCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		convoys, err := f.FetchConvoys()
		if err != nil {
			return pane.ConvoyUpdateMsg{}
		}
		progress := make(map[string][2]int)
		for _, c := range convoys {
			issues, err := f.FetchTrackedIssues(c.ID)
			if err != nil {
				continue
			}
			done := 0
			for _, iss := range issues {
				if iss.Status == "COMPLETED" || iss.Status == "CLOSED" {
					done++
				}
			}
			progress[c.ID] = [2]int{done, len(issues)}
		}
		return pane.ConvoyUpdateMsg{Convoys: convoys, Progress: progress}
	}
}

// fetchPRsCmd fetches open PRs and returns a pane.PRUpdateMsg.
func fetchPRsCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		prs, err := f.FetchPullRequests()
		return pane.PRUpdateMsg{PRs: prs, Err: err}
	}
}
