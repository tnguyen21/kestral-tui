package pane

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// ResourceInfo holds per-session resource data for display.
type ResourceInfo struct {
	SessionName  string
	CPU          float64
	MemoryMB     float64
	Uptime       time.Duration
	ProcessCount int
	LastActivity time.Time
}

// ResourceUpdateMsg carries fresh resource data to the pane.
type ResourceUpdateMsg struct {
	Resources []ResourceInfo
	Err       error
}

// SortField determines the sort order for resource rows.
type SortField int

const (
	SortByName SortField = iota
	SortByCPU
	SortByMem
)

const maxCPUHistory = 10

// ResourceMonitorPane displays CPU/memory metrics per tmux session.
type ResourceMonitorPane struct {
	resources []ResourceInfo
	cursor    int
	offset    int
	width     int
	height    int
	err       error
	sortBy    SortField
	keys      resourceKeys

	// CPU history: session name -> last N CPU samples (for sparkline)
	cpuHistory map[string][]float64
}

type resourceKeys struct {
	Up   key.Binding
	Down key.Binding
	Sort key.Binding
}

// NewResourceMonitorPane creates a new Resource Monitor pane.
func NewResourceMonitorPane() *ResourceMonitorPane {
	return &ResourceMonitorPane{
		sortBy:     SortByCPU,
		cpuHistory: make(map[string][]float64),
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

func (p *ResourceMonitorPane) ID() PaneID        { return PaneResources }
func (p *ResourceMonitorPane) Title() string      { return "Resources" }
func (p *ResourceMonitorPane) ShortTitle() string { return "📊" }

// Badge returns the count of sessions with warnings or alerts.
func (p *ResourceMonitorPane) Badge() int {
	count := 0
	for _, r := range p.resources {
		s := p.sessionStatus(r)
		if s == "warning" || s == "alert" || s == "stale" {
			count++
		}
	}
	return count
}

func (p *ResourceMonitorPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *ResourceMonitorPane) Init() tea.Cmd {
	return nil
}

func (p *ResourceMonitorPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ResourceUpdateMsg:
		p.err = msg.Err
		p.resources = msg.Resources
		// Update CPU history
		for _, r := range p.resources {
			hist := p.cpuHistory[r.SessionName]
			hist = append(hist, r.CPU)
			if len(hist) > maxCPUHistory {
				hist = hist[len(hist)-maxCPUHistory:]
			}
			p.cpuHistory[r.SessionName] = hist
		}
		p.sortResources()
		p.clampScroll()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Up):
			if p.cursor > 0 {
				p.cursor--
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Down):
			if p.cursor < len(p.resources)-1 {
				p.cursor++
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Sort):
			p.sortBy = (p.sortBy + 1) % 3
			p.sortResources()
		}
	}
	return p, nil
}

func (p *ResourceMonitorPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header
	alertCount := p.Badge()
	header := fmt.Sprintf("─── RESOURCES (%d sessions", len(p.resources))
	if alertCount > 0 {
		header += fmt.Sprintf(", %d alert", alertCount)
	}
	header += ") ───"
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.resources) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No tmux sessions found"))
		return b.String()
	}

	// Column header
	colHeader := p.renderColumnHeader()
	b.WriteString(colHeader)
	b.WriteString("\n")

	// Content area (height minus header, column header, and footer)
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

	// Footer with sort indicator
	sortLabel := "name"
	switch p.sortBy {
	case SortByCPU:
		sortLabel = "cpu"
	case SortByMem:
		sortLabel = "mem"
	}
	footer := theme.MutedStyle.Render(fmt.Sprintf("j/k=scroll  s=sort(%s)", sortLabel))
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

func (p *ResourceMonitorPane) renderColumnHeader() string {
	if p.width >= 70 {
		return theme.MutedStyle.Render(fmt.Sprintf("  %-16s %6s %7s %6s %5s %-9s %s",
			"SESSION", "CPU", "MEM", "UPTIME", "PROCS", "STATUS", "CPU HISTORY"))
	}
	return theme.MutedStyle.Render(fmt.Sprintf("  %-12s %6s %7s %-9s",
		"SESSION", "CPU", "MEM", "STATUS"))
}

func (p *ResourceMonitorPane) renderRows() []string {
	var rows []string
	for i, r := range p.resources {
		selected := i == p.cursor
		rows = append(rows, p.renderResourceRow(r, selected))
	}
	return rows
}

func (p *ResourceMonitorPane) renderResourceRow(r ResourceInfo, selected bool) string {
	status := p.sessionStatus(r)
	icon := resourceStatusIcon(status)

	cpuStr := fmt.Sprintf("%5.1f%%", r.CPU)
	memStr := formatMemory(r.MemoryMB)
	name := r.SessionName

	var line string
	if p.width >= 70 {
		uptimeStr := formatUptime(r.Uptime)
		procsStr := fmt.Sprintf("%5d", r.ProcessCount)
		statusStr := fmt.Sprintf("%s %-7s", icon, status)
		spark := p.sparkline(r.SessionName)

		nameStr := padOrTruncate(name, 16)
		line = fmt.Sprintf("  %s %s %7s %6s %s %s %s",
			nameStr, cpuStr, memStr, uptimeStr, procsStr, statusStr, spark)
	} else {
		nameStr := padOrTruncate(name, 12)
		statusStr := fmt.Sprintf("%s %-7s", icon, status)
		line = fmt.Sprintf("  %s %s %7s %s",
			nameStr, cpuStr, memStr, statusStr)
	}

	if selected {
		return theme.AccentStyle.Bold(true).Render(TruncateWithEllipsis(line, p.width))
	}

	// Color the line based on status
	switch status {
	case "alert":
		return theme.FailStyle.Render(TruncateWithEllipsis(line, p.width))
	case "warning":
		return theme.WarnStyle.Render(TruncateWithEllipsis(line, p.width))
	case "stale":
		return theme.MutedStyle.Render(TruncateWithEllipsis(line, p.width))
	default:
		return TruncateWithEllipsis(line, p.width)
	}
}

// sessionStatus computes the alert status for a session.
func (p *ResourceMonitorPane) sessionStatus(r ResourceInfo) string {
	// Check stale: no activity for >15 minutes
	if !r.LastActivity.IsZero() {
		if time.Since(r.LastActivity) > 15*time.Minute {
			return "stale"
		}
	}

	history := p.cpuHistory[r.SessionName]

	// Very high CPU (>95%) sustained for 10+ samples (~5 min at 30s intervals)
	if countTrailingAbove(history, 95) >= 10 {
		return "alert"
	}

	// High CPU (>80%) sustained for 4+ samples (~2 min at 30s intervals)
	if countTrailingAbove(history, 80) >= 4 {
		return "warning"
	}

	return "healthy"
}

// countTrailingAbove returns how many consecutive trailing samples exceed the threshold.
func countTrailingAbove(samples []float64, threshold float64) int {
	count := 0
	for i := len(samples) - 1; i >= 0; i-- {
		if samples[i] > threshold {
			count++
		} else {
			break
		}
	}
	return count
}

// sparkline renders a mini CPU history chart using block characters.
func (p *ResourceMonitorPane) sparkline(sessionName string) string {
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	history := p.cpuHistory[sessionName]

	if len(history) == 0 {
		return theme.MutedStyle.Render("──────────")
	}

	var sb strings.Builder
	for _, v := range history {
		idx := int(v / 100.0 * 7)
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		sb.WriteRune(blocks[idx])
	}

	// Pad to 10 chars if fewer samples
	for i := len(history); i < maxCPUHistory; i++ {
		sb.WriteRune('─')
	}

	spark := sb.String()

	// Color based on latest value
	if len(history) > 0 {
		latest := history[len(history)-1]
		switch {
		case latest > 95:
			return theme.FailStyle.Render(spark)
		case latest > 80:
			return theme.WarnStyle.Render(spark)
		default:
			return theme.PassStyle.Render(spark)
		}
	}
	return theme.MutedStyle.Render(spark)
}

func (p *ResourceMonitorPane) sortResources() {
	switch p.sortBy {
	case SortByName:
		sort.Slice(p.resources, func(i, j int) bool {
			return p.resources[i].SessionName < p.resources[j].SessionName
		})
	case SortByCPU:
		sort.Slice(p.resources, func(i, j int) bool {
			return p.resources[i].CPU > p.resources[j].CPU
		})
	case SortByMem:
		sort.Slice(p.resources, func(i, j int) bool {
			return p.resources[i].MemoryMB > p.resources[j].MemoryMB
		})
	}
}

func (p *ResourceMonitorPane) scrollToCursor() {
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

func (p *ResourceMonitorPane) clampScroll() {
	contentHeight := p.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	maxOffset := len(p.resources) - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	if p.cursor >= len(p.resources) {
		p.cursor = len(p.resources) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func resourceStatusIcon(status string) string {
	switch status {
	case "healthy":
		return theme.IconWorking
	case "warning":
		return theme.IconStale
	case "alert":
		return theme.IconStuck
	case "stale":
		return theme.IconIdle
	default:
		return theme.IconIdle
	}
}

func formatMemory(mb float64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGB", mb/1024)
	}
	return fmt.Sprintf("%.0fMB", mb)
}

func formatUptime(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// Ensure ResourceMonitorPane implements Pane at compile time.
var _ Pane = (*ResourceMonitorPane)(nil)

// Ensure ResourceUpdateMsg implements tea.Msg.
var _ tea.Msg = ResourceUpdateMsg{}
