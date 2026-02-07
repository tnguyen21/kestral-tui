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
	detailAgent *pane.AgentInfo // agent currently viewed in detail mode
}

// New creates a root Model with the given config.
func New(cfg config.Config) Model {
	fetcher := &data.Fetcher{TownRoot: cfg.TownRoot}
	panes := []pane.Pane{
		pane.NewDashboard(),
		pane.NewAgentsPane(),
		pane.NewRefineryPane(),
		pane.NewPRsPane(),
		pane.NewConvoysPane(),
		pane.NewResourcesPane(),
		pane.NewNewIssuePane(),
		pane.NewMailPane(),
		pane.NewWitnessPane(),
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
		fetchRigsCmd(m.fetcher),
		fetchMailCmd(m.fetcher),
		fetchRefineryCmd(m.fetcher),
		fetchResourcesCmd(m.fetcher),
		fetchWitnessesCmd(m.fetcher),
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
	case data.MailTickMsg:
		return m, fetchMailCmd(m.fetcher)
	case data.RefineryTickMsg:
		return m, fetchRefineryCmd(m.fetcher)
	case data.ResourceTickMsg:
		return m, fetchResourcesCmd(m.fetcher)
	case data.WitnessTickMsg:
		return m, fetchWitnessesCmd(m.fetcher)
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

	// Rig list and issue submission — forward to all panes.
	case pane.RigListMsg:
		cmds := m.forwardToAllPanes(msg)
		return m, tea.Batch(cmds...)

	case pane.IssueSubmitMsg:
		cmds := m.forwardToAllPanes(msg)
		return m, tea.Batch(cmds...)

	case pane.MailUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleMailPoll(
			time.Duration(m.config.PollInterval.Mail)*time.Second))
		return m, tea.Batch(cmds...)

	case pane.RefineryUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleRefineryPoll(
			time.Duration(m.config.PollInterval.Refinery)*time.Second))
		return m, tea.Batch(cmds...)

	// Agent detail view messages
	case pane.AgentSelectedMsg:
		agent := msg.Agent
		m.detailAgent = &agent
		return m, fetchAgentDetailCmd(m.fetcher, agent.Rig, agent.Name)

	case pane.AgentDeselectedMsg:
		m.detailAgent = nil
		return m, nil

	case pane.AgentDetailDataMsg:
		cmds := m.forwardToAllPanes(msg)
		if m.detailAgent != nil {
			cmds = append(cmds, data.ScheduleAgentDetailPoll(10*time.Second))
		}
		return m, tea.Batch(cmds...)

	case data.AgentDetailTickMsg:
		if m.detailAgent != nil {
			return m, fetchAgentDetailCmd(m.fetcher, m.detailAgent.Rig, m.detailAgent.Name)
		}
		return m, nil

	case pane.ResourceUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleResourcePoll(
			time.Duration(m.config.PollInterval.Resources)*time.Second))
		return m, tea.Batch(cmds...)

	case pane.WitnessUpdateMsg:
		cmds := m.forwardToAllPanes(msg)
		cmds = append(cmds, data.ScheduleWitnessPoll(
			time.Duration(m.config.PollInterval.Witnesses)*time.Second))
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

// inputPane returns true if the active pane captures keyboard input
// (e.g., a form) and most global keys should be forwarded instead.
func (m Model) inputPane() bool {
	if m.activePane < len(m.panes) {
		return m.panes[m.activePane].ID() == pane.PaneNewIssue
	}
	return false
}

// handleKey processes global key bindings, forwarding unhandled keys
// to the active pane.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When an input-capturing pane is active, only handle ctrl+c for quit.
	// All other keys go to the pane for text input.
	if m.inputPane() {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.updateActivePane(msg)
	}

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
			fetchMailCmd(m.fetcher),
			fetchRefineryCmd(m.fetcher),
			fetchResourcesCmd(m.fetcher),
			fetchWitnessesCmd(m.fetcher),
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

// fetchRigsCmd fetches available rig names and returns a pane.RigListMsg.
func fetchRigsCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		rigs, err := f.FetchRigs()
		return pane.RigListMsg{Rigs: rigs, Err: err}
	}
}

// fetchMailCmd fetches mail messages and returns a pane.MailUpdateMsg.
func fetchMailCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		messages, err := f.FetchMail()
		infos := make([]pane.MailInfo, len(messages))
		for i, m := range messages {
			ts, _ := time.Parse(time.RFC3339Nano, m.Timestamp)
			infos[i] = pane.MailInfo{
				ID:        m.ID,
				From:      m.From,
				To:        m.To,
				Subject:   m.Subject,
				Body:      m.Body,
				Timestamp: ts,
				Read:      m.Read,
				Priority:  m.Priority,
				Type:      m.Type,
				ThreadID:  m.ThreadID,
			}
		}
		return pane.MailUpdateMsg{Messages: infos, Err: err}
	}
}

// fetchRefineryCmd fetches refinery merge queue status and returns a pane.RefineryUpdateMsg.
func fetchRefineryCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		statuses, err := f.FetchRefineryStatus()
		return pane.RefineryUpdateMsg{Statuses: statuses, Err: err}
	}
}

// fetchAgentDetailCmd fetches git branch, commits, and tmux output for a specific agent.
func fetchAgentDetailCmd(f *data.Fetcher, rig, name string) tea.Cmd {
	return func() tea.Msg {
		branch := f.FetchAgentBranch(rig, name)
		commits := f.FetchAgentCommits(rig, name, 5)
		output := f.FetchAgentOutput(rig, name, 15)
		return pane.AgentDetailDataMsg{
			Name:    name,
			Branch:  branch,
			Commits: commits,
			Output:  output,
		}
	}
}

// fetchResourcesCmd fetches session resource data and returns a pane.ResourceUpdateMsg.
func fetchResourcesCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		resources, err := f.FetchResources()
		return pane.ResourceUpdateMsg{
			Sessions: resources,
			Err:      err,
		}
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
		issueMap := make(map[string][]data.IssueDetail)
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
			issueMap[c.ID] = issues
		}
		return pane.ConvoyUpdateMsg{
			Convoys:  convoys,
			Progress: progress,
			Issues:   issueMap,
		}
	}
}

// fetchWitnessesCmd fetches witness heartbeat data and returns a pane.WitnessUpdateMsg.
func fetchWitnessesCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		details, err := f.FetchWitnesses()
		now := time.Now()
		witnesses := make([]pane.WitnessInfo, len(details))
		for i, d := range details {
			var lastHeartbeat time.Duration
			if d.LastHeartbeat > 0 {
				lastHeartbeat = now.Sub(time.Unix(d.LastHeartbeat, 0))
			}
			var uptime time.Duration
			if d.SessionCreated > 0 {
				uptime = now.Sub(time.Unix(d.SessionCreated, 0))
			}
			witnesses[i] = pane.WitnessInfo{
				Rig:           d.Rig,
				Status:        d.Status,
				LastHeartbeat: lastHeartbeat,
				PolecatCount:  d.PolecatCount,
				Uptime:        uptime,
				HasSession:    d.HasSession,
			}
		}
		return pane.WitnessUpdateMsg{Witnesses: witnesses, Err: err}
	}
}

// fetchPRsCmd fetches open PRs and returns a pane.PRUpdateMsg.
func fetchPRsCmd(f *data.Fetcher) tea.Cmd {
	return func() tea.Msg {
		prs, err := f.FetchPullRequests()
		return pane.PRUpdateMsg{PRs: prs, Err: err}
	}
}
