package pane

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// PRUpdateMsg carries fresh PR data to the pane.
type PRUpdateMsg struct {
	PRs []data.PRInfo
	Err error
}

// PRsPane displays open PRs with status checks, review state, and merge info.
type PRsPane struct {
	prs    []data.PRInfo
	cursor int
	offset int // viewport scroll offset
	width  int
	height int
	err    error
	detail bool // showing detail view
	keys   prKeys
}

type prKeys struct {
	Up   key.Binding
	Down key.Binding
}

// NewPRsPane creates a new PRs pane.
func NewPRsPane() *PRsPane {
	return &PRsPane{
		keys: prKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
		},
	}
}

func (p *PRsPane) ID() PaneID        { return PanePRs }
func (p *PRsPane) Title() string      { return "PRs" }
func (p *PRsPane) ShortTitle() string { return "📋" }

// Badge returns the count of open PRs.
func (p *PRsPane) Badge() int {
	return len(p.prs)
}

func (p *PRsPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *PRsPane) Init() tea.Cmd {
	return nil
}

func (p *PRsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PRUpdateMsg:
		p.prs = msg.PRs
		p.err = msg.Err
		p.clampScroll()

	case tea.KeyMsg:
		if p.detail {
			return p.updateDetail(msg)
		}
		return p.updateList(msg)
	}
	return p, nil
}

func (p *PRsPane) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, p.keys.Up):
		if p.cursor > 0 {
			p.cursor--
			p.scrollToCursor()
		}
	case key.Matches(msg, p.keys.Down):
		if p.cursor < len(p.prs)-1 {
			p.cursor++
			p.scrollToCursor()
		}
	case msg.String() == "enter":
		if p.cursor < len(p.prs) {
			p.detail = true
		}
	}
	return p, nil
}

func (p *PRsPane) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		p.detail = false
	}
	return p, nil
}

func (p *PRsPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	if p.detail && p.cursor < len(p.prs) {
		return p.renderDetail()
	}
	return p.renderList()
}

func (p *PRsPane) renderList() string {
	var b strings.Builder

	// Header
	header := fmt.Sprintf("─── PRs (%d open) ───", len(p.prs))
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.prs) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No open PRs"))
		return b.String()
	}

	// Content area (height minus header and footer)
	contentHeight := p.height - 2
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

	// Pad remaining lines
	for i := len(visible); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Footer
	footer := theme.MutedStyle.Render("j/k scroll  enter detail  r refresh")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderRows produces display rows for the PR list.
func (p *PRsPane) renderRows() []string {
	var rows []string
	for i, pr := range p.prs {
		selected := i == p.cursor
		icon := prStatusIcon(pr)
		line := formatPRRow(icon, pr, p.width, selected)
		rows = append(rows, line)

		// Second line: branch + review + merge info
		detail := formatPRDetailLine(pr, p.width, selected)
		rows = append(rows, detail)
	}
	return rows
}

func formatPRRow(icon string, pr data.PRInfo, width int, selected bool) string {
	numStr := fmt.Sprintf("#%d", pr.Number)
	author := pr.Author.Login
	if author == "" {
		author = "unknown"
	}

	age := prAge(pr.CreatedAt)
	title := pr.Title

	// Layout: "  <icon> #N  title  author  age"
	fixedLen := 2 + 3 + 1 + len(numStr) + 2 + len(author) + 2 + len(age)
	titleMax := width - fixedLen
	if titleMax < 4 {
		titleMax = 4
	}
	title = TruncateWithEllipsis(title, titleMax)

	line := fmt.Sprintf("  %s %s  %s  %s  %s",
		icon, numStr,
		padOrTruncate(title, titleMax),
		theme.MutedStyle.Render(author),
		theme.MutedStyle.Render(age),
	)

	if selected {
		return theme.AccentStyle.Bold(true).Render(
			fmt.Sprintf("  %s %s  %s  %s  %s", icon, numStr, padOrTruncate(title, titleMax), author, age),
		)
	}
	return line
}

func formatPRDetailLine(pr data.PRInfo, width int, selected bool) string {
	parts := []string{}

	// Review state
	review := prReviewLabel(pr.ReviewDecision)
	parts = append(parts, review)

	// Checks summary
	checks := prChecksLabel(pr)
	parts = append(parts, checks)

	// Merge state
	merge := prMergeLabel(pr)
	parts = append(parts, merge)

	// Diff stats
	stats := fmt.Sprintf("+%d/-%d %df", pr.Additions, pr.Deletions, pr.ChangedFiles)
	parts = append(parts, stats)

	line := "      " + strings.Join(parts, "  ")

	style := theme.MutedStyle
	if selected {
		style = theme.AccentStyle
	}
	return style.Render(TruncateWithEllipsis(line, width))
}

func (p *PRsPane) renderDetail() string {
	var b strings.Builder
	pr := p.prs[p.cursor]

	// Header
	header := fmt.Sprintf("─── PR #%d ───", pr.Number)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n\n")

	// Title
	b.WriteString("  ")
	b.WriteString(theme.AccentStyle.Bold(true).Render(TruncateWithEllipsis(pr.Title, p.width-2)))
	b.WriteString("\n\n")

	// Meta
	author := pr.Author.Login
	if author == "" {
		author = "unknown"
	}
	b.WriteString(fmt.Sprintf("  Author:   %s\n", author))
	b.WriteString(fmt.Sprintf("  Branch:   %s\n", TruncateWithEllipsis(pr.HeadRefName, p.width-12)))
	b.WriteString(fmt.Sprintf("  Created:  %s\n", prAge(pr.CreatedAt)))
	if pr.IsDraft {
		b.WriteString(fmt.Sprintf("  Draft:    %s\n", theme.MutedStyle.Render("yes")))
	}
	b.WriteString("\n")

	// Status checks
	b.WriteString("  ")
	b.WriteString(theme.PaneHeaderStyle.Render("Status Checks"))
	b.WriteString("\n")
	if len(pr.StatusChecks) == 0 {
		b.WriteString(theme.MutedStyle.Render("    (none)"))
		b.WriteString("\n")
	} else {
		for _, check := range pr.StatusChecks {
			icon := checkIcon(check)
			name := TruncateWithEllipsis(check.Name, p.width-8)
			b.WriteString(fmt.Sprintf("    %s %s\n", icon, name))
		}
	}
	b.WriteString("\n")

	// Review state
	b.WriteString(fmt.Sprintf("  Review:     %s\n", prReviewLabel(pr.ReviewDecision)))
	b.WriteString(fmt.Sprintf("  Mergeable:  %s\n", prMergeLabel(pr)))
	b.WriteString(fmt.Sprintf("  Changes:    +%d -%d (%d files)\n", pr.Additions, pr.Deletions, pr.ChangedFiles))
	b.WriteString("\n")

	// Footer
	footer := theme.MutedStyle.Render("esc back")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// prStatusIcon returns a colored icon summarizing the PR's overall status.
func prStatusIcon(pr data.PRInfo) string {
	if pr.IsDraft {
		return theme.MutedStyle.Render("○")
	}

	checksOK := prAllChecksPassed(pr)
	approved := pr.ReviewDecision == "APPROVED"

	switch {
	case checksOK && approved:
		return theme.PassStyle.Render("✓")
	case pr.Mergeable == "CONFLICTING" || pr.ReviewDecision == "CHANGES_REQUESTED":
		return theme.FailStyle.Render("✗")
	case !checksOK && len(pr.StatusChecks) > 0:
		// Check if any failed vs still pending
		for _, c := range pr.StatusChecks {
			if c.Conclusion == "FAILURE" || c.Conclusion == "CANCELLED" || c.Conclusion == "TIMED_OUT" {
				return theme.FailStyle.Render("✗")
			}
		}
		return theme.WarnStyle.Render("◐")
	default:
		return theme.WarnStyle.Render("◐")
	}
}

// prAllChecksPassed returns true if all status checks have passed.
func prAllChecksPassed(pr data.PRInfo) bool {
	if len(pr.StatusChecks) == 0 {
		return false
	}
	for _, c := range pr.StatusChecks {
		if c.Conclusion != "SUCCESS" && c.Conclusion != "NEUTRAL" && c.Conclusion != "SKIPPED" {
			return false
		}
	}
	return true
}

func prReviewLabel(decision string) string {
	switch decision {
	case "APPROVED":
		return theme.PassStyle.Render("approved")
	case "CHANGES_REQUESTED":
		return theme.FailStyle.Render("changes requested")
	case "REVIEW_REQUIRED":
		return theme.WarnStyle.Render("review required")
	default:
		return theme.MutedStyle.Render("no reviews")
	}
}

func prChecksLabel(pr data.PRInfo) string {
	if len(pr.StatusChecks) == 0 {
		return theme.MutedStyle.Render("no checks")
	}

	passed, failed, pending := 0, 0, 0
	for _, c := range pr.StatusChecks {
		switch c.Conclusion {
		case "SUCCESS", "NEUTRAL", "SKIPPED":
			passed++
		case "FAILURE", "CANCELLED", "TIMED_OUT":
			failed++
		default:
			pending++
		}
	}

	total := len(pr.StatusChecks)
	if failed > 0 {
		return theme.FailStyle.Render(fmt.Sprintf("%d/%d checks", passed, total))
	}
	if pending > 0 {
		return theme.WarnStyle.Render(fmt.Sprintf("%d/%d checks", passed, total))
	}
	return theme.PassStyle.Render(fmt.Sprintf("%d/%d checks", passed, total))
}

func prMergeLabel(pr data.PRInfo) string {
	switch pr.Mergeable {
	case "MERGEABLE":
		return theme.PassStyle.Render("mergeable")
	case "CONFLICTING":
		return theme.FailStyle.Render("conflicts")
	default:
		return theme.MutedStyle.Render("unknown")
	}
}

func checkIcon(c data.PRStatusCheck) string {
	switch c.Conclusion {
	case "SUCCESS":
		return theme.PassStyle.Render("✓")
	case "FAILURE", "CANCELLED", "TIMED_OUT":
		return theme.FailStyle.Render("✗")
	case "NEUTRAL", "SKIPPED":
		return theme.MutedStyle.Render("–")
	default:
		return theme.WarnStyle.Render("◐")
	}
}

func prAge(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return "?"
	}
	return FormatAge(time.Since(t))
}

// scrollToCursor ensures the cursor row is visible in the viewport.
func (p *PRsPane) scrollToCursor() {
	// Each PR takes 2 rows (main + detail)
	row := p.cursor * 2

	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	if row < p.offset {
		p.offset = row
	}
	// Ensure both lines of the PR are visible
	rowEnd := row + 1
	if rowEnd >= p.offset+contentHeight {
		p.offset = rowEnd - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset stays in valid range.
func (p *PRsPane) clampScroll() {
	totalRows := len(p.prs) * 2 // 2 rows per PR
	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
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
	if p.cursor >= len(p.prs) {
		p.cursor = len(p.prs) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// Ensure PRsPane implements Pane at compile time.
var _ Pane = (*PRsPane)(nil)

// Ensure PRUpdateMsg implements tea.Msg.
var _ tea.Msg = PRUpdateMsg{}
