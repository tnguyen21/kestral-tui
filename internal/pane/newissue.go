package pane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// issueTypes available for selection.
var issueTypes = []string{"bug", "feature", "task"}

// issuePriorities available for selection.
var issuePriorities = []string{"P0 Critical", "P1 High", "P2 Medium", "P3 Low", "P4 Backlog"}

// priorityFlags maps display labels to bd create --priority flags.
var priorityFlags = []string{"0", "1", "2", "3", "4"}

// formField identifies the active field in the form.
type formField int

const (
	fieldTitle formField = iota
	fieldDescription
	fieldType
	fieldPriority
	fieldRig
	fieldCount // sentinel for wrapping
)

// formState tracks whether we're in the form, confirming, or showing a result.
type formState int

const (
	stateForm formState = iota
	stateSubmitting
	stateResult
)

// RigListMsg delivers available rig names to the pane.
type RigListMsg struct {
	Rigs []string
	Err  error
}

// IssueSubmitMsg delivers the result of a bd create call.
type IssueSubmitMsg struct {
	BeadID string
	Err    error
}

// NewIssuePane provides a form for creating bug reports and feature requests.
type NewIssuePane struct {
	width  int
	height int

	// Form state
	state       formState
	activeField formField
	titleInput  textinput.Model
	description []string // lines of multiline text
	descCursor  int      // cursor position in description
	descLine    int      // current line index
	typeIdx     int      // index into issueTypes
	priorityIdx int      // index into issuePriorities
	rigs        []string // populated from RigListMsg
	rigIdx      int      // index into rigs

	// Result
	resultID  string
	resultErr error

	keys newIssueKeys
}

type newIssueKeys struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Submit key.Binding
	Cancel key.Binding
}

// NewNewIssuePane creates a new issue creation pane.
func NewNewIssuePane() *NewIssuePane {
	ti := textinput.New()
	ti.Placeholder = "Issue title (required)"
	ti.CharLimit = 120
	ti.Focus()

	return &NewIssuePane{
		titleInput:  ti,
		description: []string{""},
		priorityIdx: 2, // Default P2 Medium
		keys: newIssueKeys{
			Up: key.NewBinding(
				key.WithKeys("up"),
			),
			Down: key.NewBinding(
				key.WithKeys("down"),
			),
			Left: key.NewBinding(
				key.WithKeys("left", "h"),
			),
			Right: key.NewBinding(
				key.WithKeys("right", "l"),
			),
			Submit: key.NewBinding(
				key.WithKeys("ctrl+s"),
			),
			Cancel: key.NewBinding(
				key.WithKeys("esc"),
			),
		},
	}
}

func (p *NewIssuePane) ID() PaneID        { return PaneNewIssue }
func (p *NewIssuePane) Title() string      { return "New Issue" }
func (p *NewIssuePane) ShortTitle() string { return "\U0001F4DD" } // 📝
func (p *NewIssuePane) Badge() int         { return 0 }

func (p *NewIssuePane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.titleInput.Width = w - 4
	if p.titleInput.Width < 10 {
		p.titleInput.Width = 10
	}
}

func (p *NewIssuePane) Init() tea.Cmd {
	return textinput.Blink
}

func (p *NewIssuePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RigListMsg:
		if msg.Err == nil && len(msg.Rigs) > 0 {
			p.rigs = msg.Rigs
		}
		return p, nil

	case IssueSubmitMsg:
		p.state = stateResult
		p.resultID = msg.BeadID
		p.resultErr = msg.Err
		return p, nil

	case tea.KeyMsg:
		return p.handleKey(msg)
	}

	// Forward to text input if focused on title
	if p.activeField == fieldTitle && p.state == stateForm {
		var cmd tea.Cmd
		p.titleInput, cmd = p.titleInput.Update(msg)
		return p, cmd
	}

	return p, nil
}

func (p *NewIssuePane) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch p.state {
	case stateResult:
		return p.handleResultKey(msg)
	case stateSubmitting:
		return p, nil
	default:
		return p.handleFormKey(msg)
	}
}

func (p *NewIssuePane) handleResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":
		// Create another
		p.resetForm()
		return p, textinput.Blink
	case "esc", "q":
		p.resetForm()
		return p, textinput.Blink
	}
	return p, nil
}

func (p *NewIssuePane) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global form keys
	if key.Matches(msg, p.keys.Submit) {
		return p.submit()
	}

	// Enter submits the form, except in description field where it creates newlines
	if msg.Type == tea.KeyEnter && p.activeField != fieldDescription {
		return p.submit()
	}

	switch p.activeField {
	case fieldTitle:
		return p.handleTitleKey(msg)
	case fieldDescription:
		return p.handleDescriptionKey(msg)
	case fieldType:
		return p.handleToggleKey(msg, &p.typeIdx, len(issueTypes))
	case fieldPriority:
		return p.handleToggleKey(msg, &p.priorityIdx, len(issuePriorities))
	case fieldRig:
		if len(p.rigs) > 0 {
			return p.handleToggleKey(msg, &p.rigIdx, len(p.rigs))
		}
		return p.handleNavKey(msg)
	}
	return p, nil
}

func (p *NewIssuePane) handleTitleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyDown:
		p.titleInput.Blur()
		p.activeField = fieldDescription
		return p, nil
	}

	var cmd tea.Cmd
	p.titleInput, cmd = p.titleInput.Update(msg)
	return p, cmd
}

func (p *NewIssuePane) handleDescriptionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		p.activeField = fieldType
		return p, nil
	case tea.KeyUp:
		if p.descLine == 0 {
			p.activeField = fieldTitle
			p.titleInput.Focus()
			return p, textinput.Blink
		}
		p.descLine--
		p.clampDescCursor()
		return p, nil
	case tea.KeyDown:
		if p.descLine >= len(p.description)-1 {
			p.activeField = fieldType
			return p, nil
		}
		p.descLine++
		p.clampDescCursor()
		return p, nil
	case tea.KeyLeft:
		if p.descCursor > 0 {
			p.descCursor--
		}
		return p, nil
	case tea.KeyRight:
		line := p.description[p.descLine]
		if p.descCursor < len([]rune(line)) {
			p.descCursor++
		}
		return p, nil
	case tea.KeyEnter:
		// Split line at cursor
		line := p.description[p.descLine]
		runes := []rune(line)
		before := string(runes[:p.descCursor])
		after := string(runes[p.descCursor:])
		p.description[p.descLine] = before
		// Insert new line after current
		tail := make([]string, len(p.description[p.descLine+1:]))
		copy(tail, p.description[p.descLine+1:])
		p.description = append(p.description[:p.descLine+1], after)
		p.description = append(p.description, tail...)
		p.descLine++
		p.descCursor = 0
		return p, nil
	case tea.KeyBackspace:
		line := p.description[p.descLine]
		runes := []rune(line)
		if p.descCursor > 0 {
			p.description[p.descLine] = string(runes[:p.descCursor-1]) + string(runes[p.descCursor:])
			p.descCursor--
		} else if p.descLine > 0 {
			// Merge with previous line
			prevLine := p.description[p.descLine-1]
			p.descCursor = len([]rune(prevLine))
			p.description[p.descLine-1] = prevLine + line
			p.description = append(p.description[:p.descLine], p.description[p.descLine+1:]...)
			p.descLine--
		}
		return p, nil
	}

	// Regular character input
	if msg.Type == tea.KeyRunes {
		line := p.description[p.descLine]
		runes := []rune(line)
		newRunes := make([]rune, 0, len(runes)+len(msg.Runes))
		newRunes = append(newRunes, runes[:p.descCursor]...)
		newRunes = append(newRunes, msg.Runes...)
		newRunes = append(newRunes, runes[p.descCursor:]...)
		p.description[p.descLine] = string(newRunes)
		p.descCursor += len(msg.Runes)
		return p, nil
	}

	return p, nil
}

func (p *NewIssuePane) handleToggleKey(msg tea.KeyMsg, idx *int, count int) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, p.keys.Left):
		if *idx > 0 {
			*idx--
		}
		return p, nil
	case key.Matches(msg, p.keys.Right):
		if *idx < count-1 {
			*idx++
		}
		return p, nil
	}
	return p.handleNavKey(msg)
}

func (p *NewIssuePane) handleNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if p.activeField > 0 {
			p.activeField--
			if p.activeField == fieldTitle {
				p.titleInput.Focus()
				return p, textinput.Blink
			}
		}
		return p, nil
	case tea.KeyDown, tea.KeyTab:
		if p.activeField < fieldCount-1 {
			p.activeField++
		}
		return p, nil
	}
	return p, nil
}

func (p *NewIssuePane) clampDescCursor() {
	if p.descLine >= len(p.description) {
		p.descLine = len(p.description) - 1
	}
	if p.descLine < 0 {
		p.descLine = 0
	}
	lineLen := len([]rune(p.description[p.descLine]))
	if p.descCursor > lineLen {
		p.descCursor = lineLen
	}
}

func (p *NewIssuePane) submit() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(p.titleInput.Value())
	if title == "" {
		return p, nil
	}
	p.state = stateSubmitting

	// Build bd create args
	args := []string{"create", "--title", title}
	args = append(args, "--type", issueTypes[p.typeIdx])
	args = append(args, "--priority", priorityFlags[p.priorityIdx])

	desc := strings.TrimSpace(strings.Join(p.description, "\n"))
	if desc != "" {
		args = append(args, "-d", desc)
	}

	if len(p.rigs) > 0 && p.rigIdx < len(p.rigs) {
		args = append(args, "--rig", p.rigs[p.rigIdx])
	}

	return p, submitIssueCmd(args)
}

// submitIssueCmd returns a tea.Cmd that shells out to bd create.
func submitIssueCmd(args []string) tea.Cmd {
	return func() tea.Msg {
		stdout, err := runBdCreate(args)
		if err != nil {
			return IssueSubmitMsg{Err: err}
		}
		// Parse bead ID from output (typically first line contains the ID)
		id := parseBeadID(stdout)
		return IssueSubmitMsg{BeadID: id}
	}
}

func (p *NewIssuePane) resetForm() {
	p.state = stateForm
	p.activeField = fieldTitle
	p.titleInput.SetValue("")
	p.titleInput.Focus()
	p.description = []string{""}
	p.descLine = 0
	p.descCursor = 0
	p.typeIdx = 0
	p.priorityIdx = 2
	p.rigIdx = 0
	p.resultID = ""
	p.resultErr = nil
}

// View renders the new issue form.
func (p *NewIssuePane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	switch p.state {
	case stateResult:
		return p.renderResult()
	case stateSubmitting:
		return p.renderSubmitting()
	default:
		return p.renderForm()
	}
}

func (p *NewIssuePane) renderForm() string {
	var b strings.Builder

	header := "─── NEW ISSUE ───"
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	// Title field
	p.renderField(&b, fieldTitle, "Title", p.titleInput.View())

	// Description field
	descView := p.renderDescription()
	p.renderField(&b, fieldDescription, "Description", descView)

	// Type toggle
	typeView := p.renderToggle(issueTypes, p.typeIdx)
	p.renderField(&b, fieldType, "Type", typeView)

	// Priority toggle
	priorityView := p.renderToggle(issuePriorities, p.priorityIdx)
	p.renderField(&b, fieldPriority, "Priority", priorityView)

	// Rig selector
	rigView := "(loading...)"
	if len(p.rigs) > 0 {
		rigView = p.renderToggle(p.rigs, p.rigIdx)
	}
	p.renderField(&b, fieldRig, "Rig", rigView)

	b.WriteString("\n")

	// Help text
	help := p.fieldHelp()
	b.WriteString(theme.MutedStyle.Render(help))
	b.WriteString("\n")

	// Footer
	footer := theme.MutedStyle.Render("enter/ctrl+s submit  tab next field  ←/→ toggle  esc cancel")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

func (p *NewIssuePane) renderField(b *strings.Builder, field formField, label, content string) {
	active := p.activeField == field
	labelStyle := theme.MutedStyle
	if active {
		labelStyle = theme.AccentStyle
	}

	indicator := "  "
	if active {
		indicator = theme.AccentStyle.Render("> ")
	}

	b.WriteString(indicator)
	b.WriteString(labelStyle.Render(label))
	b.WriteString("\n")
	b.WriteString("    ")
	b.WriteString(content)
	b.WriteString("\n")
}

func (p *NewIssuePane) renderDescription() string {
	maxLines := 4
	maxWidth := p.width - 6
	if maxWidth < 10 {
		maxWidth = 10
	}

	active := p.activeField == fieldDescription
	var lines []string

	for i, line := range p.description {
		if i >= maxLines {
			break
		}
		display := line
		if len([]rune(display)) > maxWidth {
			display = string([]rune(display)[:maxWidth])
		}

		if active && i == p.descLine {
			// Show cursor
			runes := []rune(display)
			pos := p.descCursor
			if pos > len(runes) {
				pos = len(runes)
			}
			before := string(runes[:pos])
			cursor := theme.AccentStyle.Render("│")
			after := ""
			if pos < len(runes) {
				after = string(runes[pos:])
			}
			display = before + cursor + after
		}

		if display == "" && !active {
			display = theme.MutedStyle.Render("(empty)")
		}
		lines = append(lines, display)
	}

	if len(lines) == 0 {
		if active {
			return theme.AccentStyle.Render("│")
		}
		return theme.MutedStyle.Render("(empty)")
	}

	return strings.Join(lines, "\n    ")
}

func (p *NewIssuePane) renderToggle(options []string, selected int) string {
	var parts []string
	for i, opt := range options {
		if i == selected {
			parts = append(parts, theme.AccentStyle.Bold(true).Render("["+opt+"]"))
		} else {
			parts = append(parts, theme.MutedStyle.Render(" "+opt+" "))
		}
	}
	return strings.Join(parts, "")
}

func (p *NewIssuePane) fieldHelp() string {
	switch p.activeField {
	case fieldTitle:
		return "  Enter a short, descriptive title for the issue"
	case fieldDescription:
		return "  Describe the issue in detail. Enter for newline."
	case fieldType:
		return "  bug: something broken  feature: new capability  task: other work"
	case fieldPriority:
		return "  P0: drop everything  P1: today  P2: normal  P3: soon  P4: someday"
	case fieldRig:
		return "  Which rig owns this issue? Use ←/→ to select."
	default:
		return ""
	}
}

func (p *NewIssuePane) renderSubmitting() string {
	var b strings.Builder
	header := "─── NEW ISSUE ───"
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n\n")
	b.WriteString(theme.AccentStyle.Render("  Submitting..."))
	return b.String()
}

func (p *NewIssuePane) renderResult() string {
	var b strings.Builder
	header := "─── NEW ISSUE ───"
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n\n")

	if p.resultErr != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.resultErr.Error()))
		b.WriteString("\n\n")
		b.WriteString(theme.MutedStyle.Render("  n=try again  esc=back"))
	} else {
		b.WriteString(theme.PassStyle.Render(fmt.Sprintf("  Created: %s", p.resultID)))
		b.WriteString("\n\n")
		b.WriteString(theme.MutedStyle.Render("  n=create another  esc=back"))
	}

	return b.String()
}

// Ensure NewIssuePane implements Pane at compile time.
var _ Pane = (*NewIssuePane)(nil)

// Ensure messages implement tea.Msg.
var _ tea.Msg = RigListMsg{}
var _ tea.Msg = IssueSubmitMsg{}
