package pane

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// WitnessInfo holds parsed witness data for display.
type WitnessInfo struct {
	Rig           string
	Status        string        // alive, stale, dead
	LastHeartbeat time.Duration // time since last heartbeat (0 if no session)
	PolecatCount  int
	Uptime        time.Duration // session uptime (0 if no session)
	HasSession    bool
}

// WitnessUpdateMsg carries fresh witness data to the pane.
type WitnessUpdateMsg struct {
	Witnesses []WitnessInfo
	Err       error
}

// WitnessPane displays witness heartbeat status per rig.
type WitnessPane struct {
	witnesses []WitnessInfo
	cursor    int
	offset    int // viewport scroll offset
	width     int
	height    int
	err       error
	keys      witnessKeys
}

type witnessKeys struct {
	Up   key.Binding
	Down key.Binding
}

// NewWitnessPane creates a new Witness Heartbeat pane.
func NewWitnessPane() *WitnessPane {
	return &WitnessPane{
		keys: witnessKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
		},
	}
}

func (p *WitnessPane) ID() PaneID        { return PaneWitness }
func (p *WitnessPane) Title() string      { return "Witnesses" }
func (p *WitnessPane) ShortTitle() string { return "👁" }

// Badge returns the count of unhealthy (stale or dead) witnesses.
func (p *WitnessPane) Badge() int {
	count := 0
	for _, w := range p.witnesses {
		if w.Status != "alive" {
			count++
		}
	}
	return count
}

func (p *WitnessPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *WitnessPane) Init() tea.Cmd {
	return nil
}

func (p *WitnessPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case WitnessUpdateMsg:
		p.witnesses = msg.Witnesses
		p.err = msg.Err
		p.clampScroll()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Up):
			if p.cursor > 0 {
				p.cursor--
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Down):
			if p.cursor < len(p.witnesses)-1 {
				p.cursor++
				p.scrollToCursor()
			}
		}
	}
	return p, nil
}

func (p *WitnessPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header line
	total := len(p.witnesses)
	header := fmt.Sprintf("─── WITNESS HEARTBEAT (%d rigs) ───", total)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.witnesses) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No witness sessions detected"))
		return b.String()
	}

	// Content area (height minus header and footer)
	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render visible witness rows
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
	footer := theme.MutedStyle.Render("j/k to scroll")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderRows produces display rows for each witness with a detail line for the selected one.
func (p *WitnessPane) renderRows() []string {
	var rows []string
	for i, w := range p.witnesses {
		selected := i == p.cursor
		icon := witnessStatusIcon(w.Status)

		// Format heartbeat age
		heartbeat := "no session"
		if w.HasSession && w.LastHeartbeat > 0 {
			heartbeat = FormatAge(w.LastHeartbeat)
		}

		// Format polecat count
		polecats := fmt.Sprintf("%d polecats", w.PolecatCount)
		if w.PolecatCount == 1 {
			polecats = "1 polecat"
		}

		line := formatWitnessRow(icon, w.Rig, w.Status, heartbeat, polecats, p.width, selected)
		rows = append(rows, line)

		// Show detail line for selected witness
		if selected && w.HasSession {
			detail := formatWitnessDetail(w, p.width)
			rows = append(rows, detail)
		}
	}
	return rows
}

func formatWitnessRow(icon, rig, status, heartbeat, polecats string, width int, selected bool) string {
	rigCol := 14
	statusCol := 8
	heartbeatCol := 12

	rigStr := padOrTruncate(rig, rigCol)
	statusStr := padOrTruncate(status, statusCol)
	heartbeatStr := padOrTruncate(heartbeat, heartbeatCol)

	line := fmt.Sprintf("  %s %s%s%s%s", icon, rigStr, statusStr, heartbeatStr, polecats)

	if selected {
		return theme.AccentStyle.Bold(true).Render(line)
	}

	// Bold dead witnesses for visual alert
	if status == "dead" {
		return theme.FailStyle.Bold(true).Render(line)
	}

	return line
}

func formatWitnessDetail(w WitnessInfo, width int) string {
	var parts []string
	if w.Uptime > 0 {
		parts = append(parts, fmt.Sprintf("uptime: %s", formatUptime(w.Uptime)))
	}
	if len(parts) == 0 {
		return ""
	}
	line := "      " + strings.Join(parts, "  ")
	return theme.MutedStyle.Render(TruncateWithEllipsis(line, width))
}

func formatUptime(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%dh", int(d.Hours()/24), int(d.Hours())%24)
	}
}

func witnessStatusIcon(status string) string {
	switch status {
	case "alive":
		return theme.IconWorking
	case "stale":
		return theme.IconStale
	default: // dead
		return theme.IconStuck
	}
}

// WitnessStatusFromAge returns "alive", "stale", or "dead" based on heartbeat age.
// <5m = alive (green), <15m = stale (yellow), >15m = dead (red).
func WitnessStatusFromAge(age time.Duration) string {
	switch {
	case age < 5*time.Minute:
		return "alive"
	case age < 15*time.Minute:
		return "stale"
	default:
		return "dead"
	}
}

// scrollToCursor ensures the cursor row is visible in the viewport.
func (p *WitnessPane) scrollToCursor() {
	row := 0
	for i := 0; i < p.cursor && i < len(p.witnesses); i++ {
		row++ // witness row
		if i == p.cursor-1 {
			// don't count detail lines for previous cursor position
			break
		}
	}

	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	if row < p.offset {
		p.offset = row
	}
	// Ensure the witness row plus its potential detail line are visible
	rowEnd := row
	if p.cursor < len(p.witnesses) && p.witnesses[p.cursor].HasSession {
		rowEnd++
	}
	if rowEnd >= p.offset+contentHeight {
		p.offset = rowEnd - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset stays in valid range.
func (p *WitnessPane) clampScroll() {
	rows := p.renderRows()
	contentHeight := p.height - 2
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
	if p.cursor >= len(p.witnesses) {
		p.cursor = len(p.witnesses) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// Ensure WitnessPane implements Pane at compile time.
var _ Pane = (*WitnessPane)(nil)

// Ensure WitnessUpdateMsg implements tea.Msg.
var _ tea.Msg = WitnessUpdateMsg{}
