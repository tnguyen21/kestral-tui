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

const (
	// Number of historical samples to keep per session for sparklines.
	maxSamples = 10

	// Alert thresholds.
	cpuWarnThreshold  = 80.0
	cpuAlertThreshold = 95.0
	// Number of consecutive samples required for sustained alerts.
	cpuWarnSamples  = 4  // 4 * 30s = 2 min
	cpuAlertSamples = 10 // 10 * 30s = 5 min
	// Stale indicator: no activity change for 15 minutes.
	staleThreshold = 15 * time.Minute
)

// ResourceUpdateMsg carries resource data to the pane.
type ResourceUpdateMsg struct {
	Sessions []data.SessionResource
	Err      error
}

// sortField controls which column to sort by in the resources table.
type sortField int

const (
	sortByCPU    sortField = iota
	sortByMem
	sortByName
	sortFieldCount // sentinel for cycling
)

// sessionHistory tracks CPU samples for sparkline and alert detection.
type sessionHistory struct {
	cpuSamples []float64 // circular buffer, most recent last
}

func (h *sessionHistory) addSample(cpu float64) {
	h.cpuSamples = append(h.cpuSamples, cpu)
	if len(h.cpuSamples) > maxSamples {
		h.cpuSamples = h.cpuSamples[len(h.cpuSamples)-maxSamples:]
	}
}

// sustainedAbove checks if the last n samples are all above threshold.
func (h *sessionHistory) sustainedAbove(threshold float64, n int) bool {
	if len(h.cpuSamples) < n {
		return false
	}
	start := len(h.cpuSamples) - n
	for _, v := range h.cpuSamples[start:] {
		if v < threshold {
			return false
		}
	}
	return true
}

// ResourcesPane displays per-session CPU/memory usage with alerts.
type ResourcesPane struct {
	sessions []data.SessionResource
	history  map[string]*sessionHistory // keyed by session name
	cursor   int
	offset   int
	width    int
	height   int
	err      error
	sortBy   sortField
	keys     resourceKeys
}

type resourceKeys struct {
	Up   key.Binding
	Down key.Binding
	Sort key.Binding
}

// NewResourcesPane creates a new Resources pane.
func NewResourcesPane() *ResourcesPane {
	return &ResourcesPane{
		history: make(map[string]*sessionHistory),
		keys: resourceKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
			Sort: key.NewBinding(
				key.WithKeys("s"),
			),
		},
	}
}

func (p *ResourcesPane) ID() PaneID        { return PaneResources }
func (p *ResourcesPane) Title() string      { return "Resources" }
func (p *ResourcesPane) ShortTitle() string { return "\U0001F4CA" } // 📊

// Badge returns the count of sessions in warning or alert state.
func (p *ResourcesPane) Badge() int {
	count := 0
	for _, s := range p.sessions {
		status := p.sessionStatus(s.Name, s.CPUPercent, s.ActivityTS)
		if status == "warning" || status == "alert" || status == "stale" {
			count++
		}
	}
	return count
}

func (p *ResourcesPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *ResourcesPane) Init() tea.Cmd {
	return nil
}

func (p *ResourcesPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ResourceUpdateMsg:
		p.sessions = msg.Sessions
		p.err = msg.Err
		// Record CPU samples in history
		for _, s := range p.sessions {
			h, ok := p.history[s.Name]
			if !ok {
				h = &sessionHistory{}
				p.history[s.Name] = h
			}
			h.addSample(s.CPUPercent)
		}
		p.sortSessions()
		p.clampScroll()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Up):
			if p.cursor > 0 {
				p.cursor--
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Down):
			if p.cursor < len(p.sessions)-1 {
				p.cursor++
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Sort):
			p.sortBy = (p.sortBy + 1) % sortFieldCount
			p.sortSessions()
		}
	}
	return p, nil
}

func (p *ResourcesPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header
	alerts := p.Badge()
	sortLabel := [...]string{"cpu", "mem", "name"}[p.sortBy]
	header := fmt.Sprintf("─── RESOURCES (%d alerts, sort:%s) ───", alerts, sortLabel)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.sessions) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No tmux sessions"))
		return b.String()
	}

	// Column header
	colHeader := p.formatColumnHeader()
	b.WriteString(theme.MutedStyle.Render(TruncateWithEllipsis(colHeader, p.width)))
	b.WriteString("\n")

	// Content area
	contentHeight := p.height - 3 // header + col header + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

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

	// Pad
	for i := len(visible); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Footer
	footer := theme.MutedStyle.Render("j/k=scroll  s=sort  auto-refreshes every 30s")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

func (p *ResourcesPane) formatColumnHeader() string {
	// Columns: NAME  CPU  MEM  PROCS  UPTIME  STATUS  HISTORY
	return fmt.Sprintf("  %-14s %6s %8s %5s %8s %7s %s",
		"SESSION", "CPU%", "MEM", "PROCS", "UPTIME", "STATUS", "HISTORY")
}

func (p *ResourcesPane) renderRows() []string {
	var rows []string
	for i, s := range p.sessions {
		selected := i == p.cursor
		status := p.sessionStatus(s.Name, s.CPUPercent, s.ActivityTS)

		// Format each column
		name := padOrTruncate(s.Name, 14)
		cpu := fmt.Sprintf("%6.1f", s.CPUPercent)
		mem := fmt.Sprintf("%8s", formatBytes(s.MemRSS))
		procs := fmt.Sprintf("%5d", s.ProcessCount)
		uptime := fmt.Sprintf("%8s", formatUptime(time.Duration(s.UptimeSecs)*time.Second))
		statusStr := padOrTruncate(status, 7)

		// Sparkline
		spark := p.renderSparkline(s.Name)

		line := fmt.Sprintf("  %s %s %s %s %s %s %s",
			name, cpu, mem, procs, uptime, statusStr, spark)

		// Apply status coloring
		switch {
		case selected:
			line = theme.AccentStyle.Bold(true).Render(
				fmt.Sprintf("  %s %s %s %s %s %s %s",
					name, cpu, mem, procs, uptime, statusStr, spark))
		case status == "alert":
			// Color the status portion red
			line = fmt.Sprintf("  %s %s %s %s %s %s %s",
				name,
				theme.FailStyle.Render(cpu),
				mem, procs, uptime,
				theme.FailStyle.Render(statusStr),
				spark)
		case status == "warning" || status == "stale":
			line = fmt.Sprintf("  %s %s %s %s %s %s %s",
				name,
				theme.WarnStyle.Render(cpu),
				mem, procs, uptime,
				theme.WarnStyle.Render(statusStr),
				spark)
		case status == "healthy":
			line = fmt.Sprintf("  %s %s %s %s %s %s %s",
				name,
				theme.PassStyle.Render(cpu),
				mem, procs, uptime,
				theme.PassStyle.Render(statusStr),
				spark)
		}

		rows = append(rows, TruncateWithEllipsis(line, p.width))
	}
	return rows
}

// sessionStatus determines the health status of a session.
func (p *ResourcesPane) sessionStatus(name string, cpu float64, activityTS int64) string {
	h := p.history[name]

	// Check for stale (no activity change for >15 minutes)
	if activityTS > 0 {
		lastActivity := time.Unix(activityTS, 0)
		if time.Since(lastActivity) > staleThreshold {
			return "stale"
		}
	}

	// Check for sustained high CPU alerts
	if h != nil {
		if h.sustainedAbove(cpuAlertThreshold, cpuAlertSamples) {
			return "alert"
		}
		if h.sustainedAbove(cpuWarnThreshold, cpuWarnSamples) {
			return "warning"
		}
	}

	// Instantaneous threshold checks (for first few samples before sustained detection)
	if cpu > cpuAlertThreshold {
		return "warning" // not yet sustained, show as warning
	}

	return "healthy"
}

// renderSparkline renders a mini bar chart of the last N CPU samples.
func (p *ResourcesPane) renderSparkline(name string) string {
	h, ok := p.history[name]
	if !ok || len(h.cpuSamples) == 0 {
		return theme.MutedStyle.Render(strings.Repeat("░", maxSamples))
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var sb strings.Builder
	// Pad with empty slots if fewer than maxSamples
	for i := 0; i < maxSamples-len(h.cpuSamples); i++ {
		sb.WriteRune('░')
	}

	for _, v := range h.cpuSamples {
		// Map 0-100% to block index 0-7
		idx := int(v / 100.0 * float64(len(blocks)))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		if idx < 0 {
			idx = 0
		}

		ch := string(blocks[idx])
		switch {
		case v >= cpuAlertThreshold:
			sb.WriteString(theme.FailStyle.Render(ch))
		case v >= cpuWarnThreshold:
			sb.WriteString(theme.WarnStyle.Render(ch))
		default:
			sb.WriteString(theme.PassStyle.Render(ch))
		}
	}
	return sb.String()
}

func (p *ResourcesPane) sortSessions() {
	switch p.sortBy {
	case sortByCPU:
		sort.Slice(p.sessions, func(i, j int) bool {
			return p.sessions[i].CPUPercent > p.sessions[j].CPUPercent
		})
	case sortByMem:
		sort.Slice(p.sessions, func(i, j int) bool {
			return p.sessions[i].MemRSS > p.sessions[j].MemRSS
		})
	case sortByName:
		sort.Slice(p.sessions, func(i, j int) bool {
			return p.sessions[i].Name < p.sessions[j].Name
		})
	}
}

// scrollToCursor ensures the cursor row is visible in the viewport.
func (p *ResourcesPane) scrollToCursor() {
	contentHeight := p.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+contentHeight {
		p.offset = p.cursor - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset stays in valid range.
func (p *ResourcesPane) clampScroll() {
	rows := len(p.sessions)
	contentHeight := p.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	maxOffset := rows - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	if p.cursor >= len(p.sessions) {
		p.cursor = len(p.sessions) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// formatBytes formats bytes into human-readable form (KB, MB, GB).
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// formatUptime formats a duration as a compact uptime string.
func formatUptime(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%02dm", h, m)
	default:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, h)
	}
}

// Ensure ResourcesPane implements Pane at compile time.
var _ Pane = (*ResourcesPane)(nil)

// Ensure ResourceUpdateMsg implements tea.Msg.
var _ tea.Msg = ResourceUpdateMsg{}
