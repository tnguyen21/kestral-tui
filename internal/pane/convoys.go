package pane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// ConvoysPane displays a scrollable list of convoys with progress bars
// and an expandable detail view showing tracked issues.
type ConvoysPane struct {
	convoys  []data.ConvoyInfo
	progress map[string][2]int            // convoy ID -> (done, total)
	issues   map[string][]data.IssueDetail // convoy ID -> tracked issues
	cursor   int
	offset   int // viewport scroll offset
	expanded int // -1 = list mode, >= 0 = expanded convoy index
	width    int
	height   int
	err      error
	keys     convoyKeys
}

type convoyKeys struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding
}

// NewConvoysPane creates a new Convoys pane.
func NewConvoysPane() *ConvoysPane {
	return &ConvoysPane{
		progress: make(map[string][2]int),
		issues:   make(map[string][]data.IssueDetail),
		expanded: -1,
		keys: convoyKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
			Enter: key.NewBinding(
				key.WithKeys("enter"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc", "backspace"),
			),
		},
	}
}

func (p *ConvoysPane) ID() PaneID        { return PaneConvoys }
func (p *ConvoysPane) Title() string      { return "Convoys" }
func (p *ConvoysPane) ShortTitle() string { return "\U0001F69A" } // 🚚

// Badge returns the count of open convoys.
func (p *ConvoysPane) Badge() int {
	return len(p.convoys)
}

func (p *ConvoysPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *ConvoysPane) Init() tea.Cmd {
	return nil
}

func (p *ConvoysPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ConvoyUpdateMsg:
		p.convoys = msg.Convoys
		p.progress = msg.Progress
		if p.progress == nil {
			p.progress = make(map[string][2]int)
		}
		if msg.Issues != nil {
			p.issues = msg.Issues
		}
		if p.expanded >= len(p.convoys) {
			p.expanded = -1
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
			maxCursor := p.maxCursor()
			if p.cursor < maxCursor {
				p.cursor++
				p.scrollToCursor()
			}
		case key.Matches(msg, p.keys.Enter):
			if p.expanded == -1 {
				// Expand the selected convoy
				if p.cursor < len(p.convoys) {
					p.expanded = p.cursor
					p.cursor = 0
					p.offset = 0
				}
			}
		case key.Matches(msg, p.keys.Back):
			if p.expanded >= 0 {
				// Collapse back to list
				p.cursor = p.expanded
				p.expanded = -1
				p.offset = 0
				p.scrollToCursor()
			}
		}
	}
	return p, nil
}

func (p *ConvoysPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	if p.expanded >= 0 && p.expanded < len(p.convoys) {
		return p.viewDetail()
	}
	return p.viewList()
}

// viewList renders the convoy list view.
func (p *ConvoysPane) viewList() string {
	var b strings.Builder

	header := fmt.Sprintf("─── CONVOYS (%d open) ───", len(p.convoys))
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.convoys) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No open convoys"))
		return b.String()
	}

	contentHeight := p.height - 2 // header + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	rows := p.renderListRows()
	end := p.offset + contentHeight
	if end > len(rows) {
		end = len(rows)
	}
	visible := rows[p.offset:end]
	for _, row := range visible {
		b.WriteString(row)
		b.WriteString("\n")
	}

	for i := len(visible); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	footer := theme.MutedStyle.Render("j/k scroll  enter expand")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderListRows produces display rows for the convoy list.
func (p *ConvoysPane) renderListRows() []string {
	var rows []string
	for i, c := range p.convoys {
		done, total := 0, 0
		if pr, ok := p.progress[c.ID]; ok {
			done, total = pr[0], pr[1]
		}

		pct := 0
		if total > 0 {
			pct = done * 100 / total
		}

		bar := p.multiColorBar(c.ID, 10)
		fraction := fmt.Sprintf("%d/%d", done, total)
		pctStr := fmt.Sprintf("%3d%%", pct)
		status := convoyStatusLabel(c.Status)

		// Layout: "  <bar> <title>  <pct> <fraction>  <status>"
		titleMaxLen := p.width - 10 - 5 - len(fraction) - len(status) - 8
		if titleMaxLen < 8 {
			titleMaxLen = 8
		}
		title := TruncateWithEllipsis(c.Title, titleMaxLen)

		line := fmt.Sprintf("  %s %s  %s %s  %s",
			bar,
			padOrTruncate(title, titleMaxLen),
			pctStr,
			fraction,
			status,
		)

		selected := i == p.cursor
		if selected {
			rows = append(rows, theme.AccentStyle.Bold(true).Render(line))
		} else {
			rows = append(rows, line)
		}
	}
	return rows
}

// viewDetail renders the expanded detail view for a single convoy.
func (p *ConvoysPane) viewDetail() string {
	var b strings.Builder

	c := p.convoys[p.expanded]
	done, total := 0, 0
	if pr, ok := p.progress[c.ID]; ok {
		done, total = pr[0], pr[1]
	}

	header := fmt.Sprintf("─── CONVOY: %s (%d/%d) ───", c.Title, done, total)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	// Convoy metadata
	statusLine := fmt.Sprintf("  Status: %s    ID: %s",
		convoyStatusLabel(c.Status),
		theme.MutedStyle.Render(c.ID),
	)
	b.WriteString(TruncateWithEllipsis(statusLine, p.width))
	b.WriteString("\n")

	// Progress bar
	bar := p.multiColorBar(c.ID, 20)
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	barLine := fmt.Sprintf("  %s %d%%", bar, pct)
	b.WriteString(barLine)
	b.WriteString("\n")

	b.WriteString(theme.MutedStyle.Render(strings.Repeat("─", p.width)))
	b.WriteString("\n")

	// Issue list
	issues := p.issues[c.ID]
	contentHeight := p.height - 6 // header + status + bar + separator + footer + 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(issues) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No tracked issues"))
		b.WriteString("\n")
	} else {
		rows := p.renderIssueRows(issues)
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

		for i := len(visible); i < contentHeight; i++ {
			b.WriteString("\n")
		}
	}

	footer := theme.MutedStyle.Render("j/k scroll  esc back")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderIssueRows produces display rows for issues in detail view.
func (p *ConvoysPane) renderIssueRows(issues []data.IssueDetail) []string {
	var rows []string
	for i, iss := range issues {
		icon := issueStatusIcon(iss.Status)
		assignee := iss.Assignee
		if assignee == "" {
			assignee = "(unassigned)"
		}
		// Shorten long assignee paths (e.g., "rig/polecats/name" -> "name")
		if parts := strings.Split(assignee, "/"); len(parts) > 1 {
			assignee = parts[len(parts)-1]
		}

		idStr := padOrTruncate(iss.ID, 10)
		statusStr := padOrTruncate(iss.Status, 12)

		titleMaxLen := p.width - 4 - 10 - 12 - len(assignee) - 6
		if titleMaxLen < 8 {
			titleMaxLen = 8
		}
		title := TruncateWithEllipsis(iss.Title, titleMaxLen)

		line := fmt.Sprintf("  %s %s  %-*s  %s  %s",
			icon,
			idStr,
			titleMaxLen, title,
			StatusColor(iss.Status).Render(statusStr),
			theme.MutedStyle.Render(assignee),
		)

		selected := i == p.cursor
		if selected {
			rows = append(rows, theme.AccentStyle.Bold(true).Render(line))
		} else {
			rows = append(rows, line)
		}
	}
	return rows
}

// multiColorBar renders a progress bar with colors by issue status.
func (p *ConvoysPane) multiColorBar(convoyID string, barWidth int) string {
	issues := p.issues[convoyID]
	if len(issues) == 0 {
		// Fall back to simple progress bar
		pr := p.progress[convoyID]
		return progressBar(pr[0], pr[1], barWidth)
	}

	var closed, inProgress, blocked, pending int
	for _, iss := range issues {
		switch iss.Status {
		case "COMPLETED", "CLOSED":
			closed++
		case "IN_PROGRESS":
			inProgress++
		case "BLOCKED":
			blocked++
		default:
			pending++
		}
	}

	total := len(issues)
	if total == 0 || barWidth <= 0 {
		return theme.MutedStyle.Render(strings.Repeat("░", barWidth))
	}

	closedW := closed * barWidth / total
	inProgW := inProgress * barWidth / total
	blockedW := blocked * barWidth / total
	pendingW := barWidth - closedW - inProgW - blockedW
	if pendingW < 0 {
		pendingW = 0
	}

	var bar string
	if closedW > 0 {
		bar += theme.PassStyle.Render(strings.Repeat("█", closedW))
	}
	if inProgW > 0 {
		bar += lipgloss.NewStyle().Foreground(theme.ColorAccent).Render(strings.Repeat("█", inProgW))
	}
	if blockedW > 0 {
		bar += theme.FailStyle.Render(strings.Repeat("█", blockedW))
	}
	if pendingW > 0 {
		bar += theme.MutedStyle.Render(strings.Repeat("░", pendingW))
	}

	return bar
}

// issueStatusIcon returns a colored icon for an issue status.
func issueStatusIcon(status string) string {
	switch status {
	case "COMPLETED", "CLOSED":
		return theme.PassStyle.Render("✓")
	case "IN_PROGRESS":
		return theme.WarnStyle.Render("●")
	case "BLOCKED":
		return theme.FailStyle.Render("✗")
	default: // OPEN, PENDING
		return theme.MutedStyle.Render("○")
	}
}

// convoyStatusLabel returns a styled status string.
func convoyStatusLabel(status string) string {
	switch strings.ToLower(status) {
	case "feeding":
		return theme.WarnStyle.Render("feeding")
	case "landed":
		return theme.PassStyle.Render("landed")
	case "in-progress", "in_progress":
		return lipgloss.NewStyle().Foreground(theme.ColorAccent).Render("in-progress")
	default:
		return theme.MutedStyle.Render(status)
	}
}

// maxCursor returns the maximum valid cursor position.
func (p *ConvoysPane) maxCursor() int {
	if p.expanded >= 0 && p.expanded < len(p.convoys) {
		issues := p.issues[p.convoys[p.expanded].ID]
		if len(issues) == 0 {
			return 0
		}
		return len(issues) - 1
	}
	if len(p.convoys) == 0 {
		return 0
	}
	return len(p.convoys) - 1
}

// scrollToCursor ensures the cursor row is visible.
func (p *ConvoysPane) scrollToCursor() {
	contentHeight := p.contentHeight()

	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+contentHeight {
		p.offset = p.cursor - contentHeight + 1
	}
	p.clampScroll()
}

// contentHeight returns available content rows.
func (p *ConvoysPane) contentHeight() int {
	if p.expanded >= 0 {
		h := p.height - 6 // header + status + bar + separator + footer + 1
		if h < 1 {
			return 1
		}
		return h
	}
	h := p.height - 2 // header + footer
	if h < 1 {
		return 1
	}
	return h
}

// clampScroll keeps offset in valid range.
func (p *ConvoysPane) clampScroll() {
	contentHeight := p.contentHeight()

	var totalRows int
	if p.expanded >= 0 && p.expanded < len(p.convoys) {
		totalRows = len(p.issues[p.convoys[p.expanded].ID])
	} else {
		totalRows = len(p.convoys)
	}

	maxOffset := totalRows - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}

	max := p.maxCursor()
	if p.cursor > max {
		p.cursor = max
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// Ensure ConvoysPane implements Pane at compile time.
var _ Pane = (*ConvoysPane)(nil)
