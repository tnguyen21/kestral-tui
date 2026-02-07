package pane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// RefineryUpdateMsg carries fresh refinery data to the pane.
type RefineryUpdateMsg struct {
	Statuses []data.RefineryStatus
	Err      error
}

// RefineryPane displays merge queue status per rig.
type RefineryPane struct {
	statuses []data.RefineryStatus
	cursor   int
	offset   int // viewport scroll offset
	rigIdx   int // which rig is selected (for multi-rig tab switching)
	width    int
	height   int
	err      error
	keys     refineryKeys
}

type refineryKeys struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
}

// NewRefineryPane creates a new Refinery Status pane.
func NewRefineryPane() *RefineryPane {
	return &RefineryPane{
		keys: refineryKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
			Left: key.NewBinding(
				key.WithKeys("h", "left"),
			),
			Right: key.NewBinding(
				key.WithKeys("l", "right"),
			),
		},
	}
}

func (p *RefineryPane) ID() PaneID        { return PaneRefinery }
func (p *RefineryPane) Title() string      { return "Refinery" }
func (p *RefineryPane) ShortTitle() string { return "🔧" }

// Badge returns the total queue depth across all refineries.
func (p *RefineryPane) Badge() int {
	total := 0
	for _, s := range p.statuses {
		total += s.QueueDepth
	}
	return total
}

func (p *RefineryPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *RefineryPane) Init() tea.Cmd {
	return nil
}

func (p *RefineryPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RefineryUpdateMsg:
		p.statuses = msg.Statuses
		p.err = msg.Err
		if p.rigIdx >= len(p.statuses) {
			p.rigIdx = 0
		}
		p.clampScroll()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Up):
			if p.cursor > 0 {
				p.cursor--
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Down):
			p.cursor++
			p.clampScroll()
			p.scrollToCursor()
		case key.Matches(msg, p.keys.Left):
			if p.rigIdx > 0 {
				p.rigIdx--
				p.cursor = 0
				p.offset = 0
			}
		case key.Matches(msg, p.keys.Right):
			if p.rigIdx < len(p.statuses)-1 {
				p.rigIdx++
				p.cursor = 0
				p.offset = 0
			}
		}
	}
	return p, nil
}

func (p *RefineryPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header
	totalQueue := p.Badge()
	header := fmt.Sprintf("─── REFINERY (%d queued) ───", totalQueue)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.statuses) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No refineries active"))
		return b.String()
	}

	// Rig tabs (if multiple rigs)
	if len(p.statuses) > 1 {
		b.WriteString(p.renderRigTabs())
		b.WriteString("\n")
	}

	// Content area
	contentHeight := p.height - 2 // header + footer
	if len(p.statuses) > 1 {
		contentHeight-- // rig tabs
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	rows := p.renderRows()
	end := p.offset + contentHeight
	if end > len(rows) {
		end = len(rows)
	}
	start := p.offset
	if start > len(rows) {
		start = len(rows)
	}
	visible := rows[start:end]
	for _, row := range visible {
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Pad remaining lines
	for i := len(visible); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Footer
	var footerParts []string
	footerParts = append(footerParts, "j/k to scroll")
	if len(p.statuses) > 1 {
		footerParts = append(footerParts, "h/l switch rig")
	}
	footer := theme.MutedStyle.Render(strings.Join(footerParts, "  "))
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderRigTabs renders the rig selector tabs.
func (p *RefineryPane) renderRigTabs() string {
	var parts []string
	for i, s := range p.statuses {
		label := s.Rig
		if i == p.rigIdx {
			parts = append(parts, theme.AccentStyle.Bold(true).Render("["+label+"]"))
		} else {
			parts = append(parts, theme.MutedStyle.Render(" "+label+" "))
		}
	}
	return "  " + strings.Join(parts, " ")
}

// renderRows produces display rows for the currently selected rig.
func (p *RefineryPane) renderRows() []string {
	if p.rigIdx >= len(p.statuses) {
		return nil
	}

	s := p.statuses[p.rigIdx]
	var rows []string
	rowIdx := 0

	// Status line
	statusIcon := theme.IconIdle
	statusLabel := "stopped"
	if s.Running {
		statusIcon = theme.IconWorking
		statusLabel = "running"
	}
	rows = append(rows, fmt.Sprintf("  %s Refinery: %s", statusIcon, statusLabel))
	rowIdx++

	// Blank separator
	rows = append(rows, "")
	rowIdx++

	// Current MR being processed
	rows = append(rows, theme.AccentStyle.Render("  CURRENT"))
	rowIdx++

	if s.Current != nil {
		mr := s.Current
		icon := testStatusIcon(mr.Status)
		selected := p.cursor == rowIdx
		line := formatMRRow(icon, mr.BeadID, mr.Title, mr.Status, p.width, selected)
		rows = append(rows, line)
		rowIdx++
		if mr.Branch != "" {
			branchLine := formatMRDetail("branch", mr.Branch, p.width, selected)
			rows = append(rows, branchLine)
			rowIdx++
		}
		if mr.PRURL != "" {
			prLine := formatMRDetail("pr", mr.PRURL, p.width, selected)
			rows = append(rows, prLine)
			rowIdx++
		}
	} else {
		rows = append(rows, theme.MutedStyle.Render("  (idle)"))
		rowIdx++
	}

	// Blank separator
	rows = append(rows, "")
	rowIdx++

	// Queue
	queueLabel := fmt.Sprintf("  QUEUE (%d)", len(s.Queue))
	rows = append(rows, theme.AccentStyle.Render(queueLabel))
	rowIdx++

	if len(s.Queue) == 0 {
		rows = append(rows, theme.MutedStyle.Render("  (empty)"))
		rowIdx++
	} else {
		for i, mr := range s.Queue {
			posLabel := fmt.Sprintf("#%d", i+1)
			selected := p.cursor == rowIdx
			line := formatQueueRow(posLabel, mr.BeadID, mr.Title, p.width, selected)
			rows = append(rows, line)
			rowIdx++
		}
	}

	// Blank separator
	rows = append(rows, "")
	rowIdx++

	// Metrics
	rows = append(rows, theme.AccentStyle.Render("  METRICS"))
	rowIdx++

	successStr := fmt.Sprintf("%.0f%%", s.SuccessRate)
	if s.SuccessRate >= 90 {
		successStr = theme.PassStyle.Render(successStr)
	} else if s.SuccessRate >= 70 {
		successStr = theme.WarnStyle.Render(successStr)
	} else if len(s.History) > 0 {
		successStr = theme.FailStyle.Render(successStr)
	} else {
		successStr = theme.MutedStyle.Render("—")
	}
	rows = append(rows, fmt.Sprintf("  Success rate:  %s", successStr))
	rowIdx++

	avgTime := "—"
	if s.AvgMergeTime > 0 {
		avgTime = FormatAge(0) // Will show "just now" for 0
		if s.AvgMergeTime >= 60 {
			avgTime = fmt.Sprintf("%dm", s.AvgMergeTime/60)
		} else {
			avgTime = fmt.Sprintf("%ds", s.AvgMergeTime)
		}
	}
	rows = append(rows, fmt.Sprintf("  Avg merge:     %s", theme.MutedStyle.Render(avgTime)))
	rowIdx++

	rows = append(rows, fmt.Sprintf("  Queue wait:    %s", theme.MutedStyle.Render(queueWaitEstimate(s))))
	rowIdx++

	// Blank separator
	rows = append(rows, "")
	rowIdx++

	// History (last 10)
	histLabel := fmt.Sprintf("  HISTORY (%d)", len(s.History))
	rows = append(rows, theme.AccentStyle.Render(histLabel))
	rowIdx++

	if len(s.History) == 0 {
		rows = append(rows, theme.MutedStyle.Render("  (no history)"))
		rowIdx++
	} else {
		for _, mr := range s.History {
			icon := historyIcon(mr.Status)
			selected := p.cursor == rowIdx
			line := formatMRRow(icon, mr.BeadID, mr.Title, mr.Status, p.width, selected)
			rows = append(rows, line)
			rowIdx++
		}
	}

	return rows
}

func formatMRRow(icon, beadID, title, status string, width int, selected bool) string {
	maxTitle := width - 8 - len(beadID) - len(status) - 4
	if maxTitle < 0 {
		maxTitle = 0
	}
	truncTitle := TruncateWithEllipsis(title, maxTitle)

	line := fmt.Sprintf("  %s %s %s", icon, beadID, truncTitle)

	if selected {
		return theme.AccentStyle.Bold(true).Render(line)
	}
	return line
}

func formatMRDetail(label, value string, width int, selected bool) string {
	maxVal := width - 10 - len(label)
	if maxVal < 0 {
		maxVal = 0
	}
	line := fmt.Sprintf("      %s: %s", label, TruncateWithEllipsis(value, maxVal))
	style := theme.MutedStyle
	if selected {
		style = theme.AccentStyle
	}
	return style.Render(line)
}

func formatQueueRow(pos, beadID, title string, width int, selected bool) string {
	maxTitle := width - 8 - len(pos) - len(beadID) - 2
	if maxTitle < 0 {
		maxTitle = 0
	}
	truncTitle := TruncateWithEllipsis(title, maxTitle)
	line := fmt.Sprintf("  %s %s %s", pos, beadID, truncTitle)

	if selected {
		return theme.AccentStyle.Bold(true).Render(line)
	}
	return theme.MutedStyle.Render(line)
}

func testStatusIcon(status string) string {
	switch status {
	case "testing", "IN_PROGRESS":
		return theme.IconStale // yellow half-circle = in progress
	case "merged", "COMPLETED":
		return theme.IconWorking // green = pass
	case "failed":
		return theme.IconStuck // red = fail
	default:
		return theme.IconIdle // muted
	}
}

func historyIcon(status string) string {
	switch status {
	case "merged", "COMPLETED":
		return theme.PassStyle.Render("✓")
	case "failed":
		return theme.FailStyle.Render("✗")
	case "skipped":
		return theme.MutedStyle.Render("⊘")
	default:
		return theme.MutedStyle.Render("·")
	}
}

func queueWaitEstimate(s data.RefineryStatus) string {
	if s.QueueDepth == 0 {
		return "—"
	}
	if s.AvgMergeTime <= 0 {
		return "unknown"
	}
	est := s.QueueDepth * s.AvgMergeTime
	if est >= 3600 {
		return fmt.Sprintf("~%dh", est/3600)
	}
	if est >= 60 {
		return fmt.Sprintf("~%dm", est/60)
	}
	return fmt.Sprintf("~%ds", est)
}

// scrollToCursor ensures the cursor row is visible.
func (p *RefineryPane) scrollToCursor() {
	contentHeight := p.contentHeight()
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+contentHeight {
		p.offset = p.cursor - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset and cursor stay in valid range.
func (p *RefineryPane) clampScroll() {
	rows := p.renderRows()
	contentHeight := p.contentHeight()

	maxCursor := len(rows) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}
	if p.cursor > maxCursor {
		p.cursor = maxCursor
	}
	if p.cursor < 0 {
		p.cursor = 0
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
}

func (p *RefineryPane) contentHeight() int {
	h := p.height - 2 // header + footer
	if len(p.statuses) > 1 {
		h-- // rig tabs
	}
	if h < 1 {
		h = 1
	}
	return h
}

// Ensure RefineryPane implements Pane at compile time.
var _ Pane = (*RefineryPane)(nil)

// Ensure RefineryUpdateMsg implements tea.Msg.
var _ tea.Msg = RefineryUpdateMsg{}
