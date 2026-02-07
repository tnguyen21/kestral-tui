package pane

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// MailInfo holds parsed mail data for display.
type MailInfo struct {
	ID        string
	From      string
	To        string
	Subject   string
	Body      string
	Timestamp time.Time
	Read      bool
	Priority  string
	Type      string
	ThreadID  string
}

// MailUpdateMsg carries fresh mail data to the pane.
type MailUpdateMsg struct {
	Messages []MailInfo
	Err      error
}

// mailView tracks which sub-view is active in the mail pane.
type mailView int

const (
	mailViewInbox   mailView = iota // list of messages
	mailViewMessage                 // reading a single message
)

// MailPane displays a mail client with inbox, message reading, and navigation.
type MailPane struct {
	messages []MailInfo
	cursor   int
	offset   int // viewport scroll offset
	width    int
	height   int
	err      error
	keys     mailKeys
	view     mailView
}

type mailKeys struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Back   key.Binding
}

// NewMailPane creates a new Mail pane.
func NewMailPane() *MailPane {
	return &MailPane{
		keys: mailKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
			Select: key.NewBinding(
				key.WithKeys("enter"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
			),
		},
	}
}

func (p *MailPane) ID() PaneID        { return PaneMail }
func (p *MailPane) Title() string      { return "Mail" }
func (p *MailPane) ShortTitle() string { return "\U0001F4E7" } // 📧

// Badge returns the count of unread messages.
func (p *MailPane) Badge() int {
	count := 0
	for _, m := range p.messages {
		if !m.Read {
			count++
		}
	}
	return count
}

func (p *MailPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *MailPane) Init() tea.Cmd {
	return nil
}

func (p *MailPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MailUpdateMsg:
		p.messages = msg.Messages
		p.err = msg.Err
		p.clampScroll()

	case tea.KeyMsg:
		return p.handleKey(msg)
	}
	return p, nil
}

func (p *MailPane) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch p.view {
	case mailViewInbox:
		return p.handleInboxKey(msg)
	case mailViewMessage:
		return p.handleMessageKey(msg)
	}
	return p, nil
}

func (p *MailPane) handleInboxKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, p.keys.Up):
		if p.cursor > 0 {
			p.cursor--
			p.scrollToCursor()
		}
	case key.Matches(msg, p.keys.Down):
		if p.cursor < len(p.messages)-1 {
			p.cursor++
			p.scrollToCursor()
		}
	case key.Matches(msg, p.keys.Select):
		if p.cursor < len(p.messages) {
			p.view = mailViewMessage
			p.offset = 0 // reset scroll for message view
		}
	}
	return p, nil
}

func (p *MailPane) handleMessageKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, p.keys.Back):
		p.view = mailViewInbox
		p.clampScroll()
	case key.Matches(msg, p.keys.Up):
		if p.offset > 0 {
			p.offset--
		}
	case key.Matches(msg, p.keys.Down):
		p.offset++
	}
	return p, nil
}

func (p *MailPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	switch p.view {
	case mailViewMessage:
		return p.viewMessage()
	default:
		return p.viewInbox()
	}
}

func (p *MailPane) viewInbox() string {
	var b strings.Builder

	// Header line
	unread := p.Badge()
	header := fmt.Sprintf("─── MAIL (%d unread) ───", unread)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.messages) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No messages"))
		return b.String()
	}

	// Content area (height minus header and footer)
	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render visible message rows
	rows := p.renderInboxRows()
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
	footer := theme.MutedStyle.Render("j/k=scroll  enter=read  esc=back")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

func (p *MailPane) renderInboxRows() []string {
	var rows []string
	for i, m := range p.messages {
		selected := i == p.cursor

		// Read/unread icon
		icon := theme.IconRead
		if !m.Read {
			icon = theme.IconUnread
		}

		// Format: "  ● from      subject            age"
		from := shortAddr(m.From)
		age := FormatAge(time.Since(m.Timestamp))

		fromCol := 14
		ageCol := 10
		// Subject gets remaining space
		subjectCol := p.width - 4 - fromCol - ageCol - 2 // 4=indent+icon+space, 2=spacing
		if subjectCol < 8 {
			subjectCol = 8
		}

		fromStr := padOrTruncate(from, fromCol)
		subjectStr := padOrTruncate(m.Subject, subjectCol)
		ageStr := padOrTruncate(age, ageCol)

		line := fmt.Sprintf("  %s %s%s%s", icon, fromStr, subjectStr, ageStr)

		if selected {
			line = theme.AccentStyle.Bold(true).Render(
				fmt.Sprintf("  %s %s%s%s", iconChar(!m.Read), fromStr, subjectStr, ageStr))
		}
		rows = append(rows, line)

		// Priority indicator on second line for high-priority
		if m.Priority == "high" || m.Priority == "urgent" {
			priLine := fmt.Sprintf("      %s", theme.WarnStyle.Render("⚡ "+m.Priority))
			if selected {
				priLine = theme.AccentStyle.Render(priLine)
			}
			rows = append(rows, priLine)
		}
	}
	return rows
}

func (p *MailPane) viewMessage() string {
	if p.cursor >= len(p.messages) {
		p.view = mailViewInbox
		return p.viewInbox()
	}

	m := p.messages[p.cursor]
	var b strings.Builder

	// Header
	header := fmt.Sprintf("─── MESSAGE ───")
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	// Message metadata
	lines := []string{
		fmt.Sprintf("  From:    %s", m.From),
		fmt.Sprintf("  To:      %s", m.To),
		fmt.Sprintf("  Subject: %s", m.Subject),
		fmt.Sprintf("  Date:    %s", m.Timestamp.Format("2006-01-02 15:04")),
	}
	if m.Priority != "" && m.Priority != "normal" {
		lines = append(lines, fmt.Sprintf("  Priority: %s", theme.WarnStyle.Render(m.Priority)))
	}
	lines = append(lines, theme.MutedStyle.Render(strings.Repeat("─", p.width)))

	// Body lines
	bodyLines := wrapText(m.Body, p.width-2)
	for _, bl := range bodyLines {
		lines = append(lines, "  "+bl)
	}

	// Apply scroll offset
	contentHeight := p.height - 2 // header + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Clamp scroll offset for message view
	maxOffset := len(lines) - contentHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}

	end := p.offset + contentHeight
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[p.offset:end]
	for _, line := range visible {
		b.WriteString(TruncateWithEllipsis(line, p.width))
		b.WriteString("\n")
	}

	// Pad
	for i := len(visible); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Footer
	footer := theme.MutedStyle.Render("esc=back  j/k=scroll")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// scrollToCursor ensures the cursor row is visible in the viewport.
func (p *MailPane) scrollToCursor() {
	row := 0
	for i := 0; i < p.cursor && i < len(p.messages); i++ {
		row++
		if p.messages[i].Priority == "high" || p.messages[i].Priority == "urgent" {
			row++
		}
	}

	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	if row < p.offset {
		p.offset = row
	}
	rowEnd := row
	if p.cursor < len(p.messages) {
		m := p.messages[p.cursor]
		if m.Priority == "high" || m.Priority == "urgent" {
			rowEnd++
		}
	}
	if rowEnd >= p.offset+contentHeight {
		p.offset = rowEnd - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset stays in valid range.
func (p *MailPane) clampScroll() {
	if p.view == mailViewMessage {
		return
	}
	rows := p.renderInboxRows()
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
	if p.cursor >= len(p.messages) {
		p.cursor = len(p.messages) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// shortAddr extracts the last component of an agent address for compact display.
// e.g., "kestral_tui/polecats/jasper" -> "jasper"
func shortAddr(addr string) string {
	parts := strings.Split(addr, "/")
	return parts[len(parts)-1]
}

// iconChar returns a plain character for use in styled lines.
func iconChar(unread bool) string {
	if unread {
		return "●"
	}
	return "○"
}

// wrapText wraps text to the given width, splitting on word boundaries.
func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var result []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result = append(result, "")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}
		line := words[0]
		for _, w := range words[1:] {
			if len(line)+1+len(w) > width {
				result = append(result, line)
				line = w
			} else {
				line += " " + w
			}
		}
		result = append(result, line)
	}
	return result
}

// Ensure MailPane implements Pane at compile time.
var _ Pane = (*MailPane)(nil)

// Ensure MailUpdateMsg implements tea.Msg.
var _ tea.Msg = MailUpdateMsg{}
