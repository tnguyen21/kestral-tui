package pane

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// HistoryUpdateMsg carries fresh history data to the pane.
type HistoryUpdateMsg struct {
	ClosedBeads []data.ClosedBeadInfo
	Convoys     []data.AllConvoyInfo
	Err         error
}

// historyFilter tracks the current filter state.
type historyFilter struct {
	dateRange string // "all", "today", "7d", "30d"
	agent     string // empty = all agents
	issueType string // empty = all types
}

// historyEntry is a display-ready row for the history list.
type historyEntry struct {
	ID       string
	Title    string
	Assignee string
	Type     string
	ClosedAt time.Time
	Duration time.Duration
}

// HistoryPane displays a chronological log of completed work.
type HistoryPane struct {
	closedBeads []data.ClosedBeadInfo
	convoys     []data.AllConvoyInfo
	entries     []historyEntry // filtered + sorted for display
	cursor      int
	offset      int // viewport scroll offset
	width       int
	height      int
	err         error
	filter      historyFilter
	keys        historyKeys
}

type historyKeys struct {
	Up     key.Binding
	Down   key.Binding
	Filter key.Binding // cycle date filter
	Agent  key.Binding // cycle agent filter
	Type   key.Binding // cycle type filter
}

// NewHistoryPane creates a new History pane.
func NewHistoryPane() *HistoryPane {
	return &HistoryPane{
		filter: historyFilter{dateRange: "all"},
		keys: historyKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
			Filter: key.NewBinding(
				key.WithKeys("f"),
			),
			Agent: key.NewBinding(
				key.WithKeys("a"),
			),
			Type: key.NewBinding(
				key.WithKeys("t"),
			),
		},
	}
}

func (p *HistoryPane) ID() PaneID        { return PaneHistory }
func (p *HistoryPane) Title() string      { return "History" }
func (p *HistoryPane) ShortTitle() string { return "📜" }

// Badge returns the count of completed beads.
func (p *HistoryPane) Badge() int {
	return len(p.entries)
}

func (p *HistoryPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *HistoryPane) Init() tea.Cmd {
	return nil
}

func (p *HistoryPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HistoryUpdateMsg:
		p.closedBeads = msg.ClosedBeads
		p.convoys = msg.Convoys
		p.err = msg.Err
		p.rebuildEntries()
		p.clampScroll()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Up):
			if p.cursor > 0 {
				p.cursor--
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Down):
			if p.cursor < len(p.entries)-1 {
				p.cursor++
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Filter):
			p.cycleDateFilter()
			p.rebuildEntries()
			p.clampScroll()
		case key.Matches(msg, p.keys.Agent):
			p.cycleAgentFilter()
			p.rebuildEntries()
			p.clampScroll()
		case key.Matches(msg, p.keys.Type):
			p.cycleTypeFilter()
			p.rebuildEntries()
			p.clampScroll()
		}
	}
	return p, nil
}

func (p *HistoryPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("─── HISTORY (%d completed) ───", len(p.entries))
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	// Filter bar
	filterLine := p.renderFilterBar()
	b.WriteString(filterLine)
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.entries) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No completed work"))
		return b.String()
	}

	// Content area (height minus header, filter bar, and footer)
	contentHeight := p.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render visible rows
	rows := p.renderRows()
	end := p.offset + contentHeight
	if end > len(rows) {
		end = len(rows)
	}
	visible := rows[p.offset:end]
	for _, row := range visible {
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Pad remaining lines
	for i := len(visible); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Footer
	footer := theme.MutedStyle.Render("j/k scroll  f=date  a=agent  t=type")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderFilterBar shows the active filters.
func (p *HistoryPane) renderFilterBar() string {
	parts := []string{}

	dateLabel := "date:" + p.filter.dateRange
	parts = append(parts, theme.AccentStyle.Render(dateLabel))

	if p.filter.agent != "" {
		parts = append(parts, theme.AccentStyle.Render("agent:"+p.filter.agent))
	}

	if p.filter.issueType != "" {
		parts = append(parts, theme.AccentStyle.Render("type:"+p.filter.issueType))
	}

	return "  " + strings.Join(parts, "  ")
}

// renderRows builds display rows grouped by day.
func (p *HistoryPane) renderRows() []string {
	var rows []string
	var lastDay string

	for i, e := range p.entries {
		day := e.ClosedAt.Format("2006-01-02")
		if day != lastDay {
			// Day header
			dayHeader := theme.AccentStyle.Render("  ── " + day + " ──")
			rows = append(rows, dayHeader)
			lastDay = day
		}

		selected := i == p.cursor
		rows = append(rows, p.formatEntryRow(e, selected))
	}
	return rows
}

// formatEntryRow formats a single history entry.
func (p *HistoryPane) formatEntryRow(e historyEntry, selected bool) string {
	// Layout: "  ✓ <id>  <title>  <assignee>  <duration>"
	icon := theme.PassStyle.Render("✓")

	idCol := 12
	assigneeCol := 14
	durCol := 10

	// Calculate remaining width for title
	fixedWidth := 2 + 2 + idCol + 2 + assigneeCol + 2 + durCol
	titleCol := p.width - fixedWidth
	if titleCol < 10 {
		titleCol = 10
	}

	idStr := padOrTruncate(e.ID, idCol)
	titleStr := padOrTruncate(e.Title, titleCol)

	// Extract short assignee name (last path component)
	assigneeName := shortAssignee(e.Assignee)
	assigneeStr := padOrTruncate(assigneeName, assigneeCol)

	durStr := formatDuration(e.Duration)
	durStr = padOrTruncate(durStr, durCol)

	line := fmt.Sprintf("  %s %s%s%s%s", icon, idStr, titleStr, assigneeStr, durStr)

	if selected {
		return theme.AccentStyle.Bold(true).Render(line)
	}
	return line
}

// rebuildEntries converts raw data to filtered, sorted display entries.
func (p *HistoryPane) rebuildEntries() {
	p.entries = nil

	now := time.Now()
	cutoff := time.Time{}
	switch p.filter.dateRange {
	case "today":
		y, m, d := now.Date()
		cutoff = time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	case "7d":
		cutoff = now.AddDate(0, 0, -7)
	case "30d":
		cutoff = now.AddDate(0, 0, -30)
	}

	for _, bead := range p.closedBeads {
		closedAt := parseTime(bead.ClosedAt)
		if closedAt.IsZero() {
			continue
		}

		// Apply date filter
		if !cutoff.IsZero() && closedAt.Before(cutoff) {
			continue
		}

		// Apply agent filter
		if p.filter.agent != "" {
			name := shortAssignee(bead.Assignee)
			if name != p.filter.agent {
				continue
			}
		}

		// Apply type filter
		if p.filter.issueType != "" && bead.IssueType != p.filter.issueType {
			continue
		}

		createdAt := parseTime(bead.CreatedAt)
		dur := time.Duration(0)
		if !createdAt.IsZero() {
			dur = closedAt.Sub(createdAt)
		}

		p.entries = append(p.entries, historyEntry{
			ID:       bead.ID,
			Title:    bead.Title,
			Assignee: bead.Assignee,
			Type:     bead.IssueType,
			ClosedAt: closedAt,
			Duration: dur,
		})
	}

	// Sort by closed time, most recent first
	sort.Slice(p.entries, func(i, j int) bool {
		return p.entries[i].ClosedAt.After(p.entries[j].ClosedAt)
	})
}

// cycleDateFilter cycles through date range options.
func (p *HistoryPane) cycleDateFilter() {
	switch p.filter.dateRange {
	case "all":
		p.filter.dateRange = "today"
	case "today":
		p.filter.dateRange = "7d"
	case "7d":
		p.filter.dateRange = "30d"
	case "30d":
		p.filter.dateRange = "all"
	}
	p.cursor = 0
}

// cycleAgentFilter cycles through unique agents in the data.
func (p *HistoryPane) cycleAgentFilter() {
	agents := p.uniqueAgents()
	if len(agents) == 0 {
		p.filter.agent = ""
		return
	}

	if p.filter.agent == "" {
		p.filter.agent = agents[0]
		p.cursor = 0
		return
	}

	for i, a := range agents {
		if a == p.filter.agent {
			if i+1 < len(agents) {
				p.filter.agent = agents[i+1]
			} else {
				p.filter.agent = "" // back to "all"
			}
			p.cursor = 0
			return
		}
	}
	p.filter.agent = ""
	p.cursor = 0
}

// cycleTypeFilter cycles through unique issue types.
func (p *HistoryPane) cycleTypeFilter() {
	types := p.uniqueTypes()
	if len(types) == 0 {
		p.filter.issueType = ""
		return
	}

	if p.filter.issueType == "" {
		p.filter.issueType = types[0]
		p.cursor = 0
		return
	}

	for i, t := range types {
		if t == p.filter.issueType {
			if i+1 < len(types) {
				p.filter.issueType = types[i+1]
			} else {
				p.filter.issueType = ""
			}
			p.cursor = 0
			return
		}
	}
	p.filter.issueType = ""
	p.cursor = 0
}

// uniqueAgents returns sorted unique short assignee names.
func (p *HistoryPane) uniqueAgents() []string {
	seen := make(map[string]bool)
	for _, b := range p.closedBeads {
		name := shortAssignee(b.Assignee)
		if name != "" {
			seen[name] = true
		}
	}
	var out []string
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// uniqueTypes returns sorted unique issue types.
func (p *HistoryPane) uniqueTypes() []string {
	seen := make(map[string]bool)
	for _, b := range p.closedBeads {
		if b.IssueType != "" {
			seen[b.IssueType] = true
		}
	}
	var out []string
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// scrollToCursor ensures the cursor row is visible.
func (p *HistoryPane) scrollToCursor() {
	// Account for day headers: count rows up to cursor
	row := 0
	var lastDay string
	for i := 0; i <= p.cursor && i < len(p.entries); i++ {
		day := p.entries[i].ClosedAt.Format("2006-01-02")
		if day != lastDay {
			row++ // day header
			lastDay = day
		}
		if i < p.cursor {
			row++ // entry row
		}
	}

	contentHeight := p.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	if row < p.offset {
		p.offset = row
	}
	if row >= p.offset+contentHeight {
		p.offset = row - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset stays in valid range.
func (p *HistoryPane) clampScroll() {
	rows := p.renderRows()
	contentHeight := p.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	maxOffset := len(rows) - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	if p.cursor >= len(p.entries) {
		p.cursor = len(p.entries) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// shortAssignee extracts the last path component from an assignee path.
// "kestral_tui/polecats/amber" -> "amber"
func shortAssignee(assignee string) string {
	if assignee == "" {
		return ""
	}
	parts := strings.Split(assignee, "/")
	return parts[len(parts)-1]
}

// parseTime tries to parse a timestamp string in common formats.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// Ensure HistoryPane implements Pane at compile time.
var _ Pane = (*HistoryPane)(nil)

// Ensure HistoryUpdateMsg implements tea.Msg.
var _ tea.Msg = HistoryUpdateMsg{}
