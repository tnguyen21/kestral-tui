package pane

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// AgentInfo holds parsed agent data for display.
type AgentInfo struct {
	Name       string
	Rig        string
	Role       string // witness, refinery, polecat
	Status     string // working, stale, stuck, idle
	Age        time.Duration
	IssueID    string
	IssueTitle string
}

// AgentUpdateMsg carries fresh agent data to the pane.
type AgentUpdateMsg struct {
	Agents []AgentInfo
	Err    error
}

// AgentsPane displays a scrollable list of running agents with live status.
type AgentsPane struct {
	agents   []AgentInfo
	cursor   int
	offset   int // viewport scroll offset
	width    int
	height   int
	err      error
	keys     agentKeys
}

type agentKeys struct {
	Up   key.Binding
	Down key.Binding
}

// NewAgentsPane creates a new Agents pane.
func NewAgentsPane() *AgentsPane {
	return &AgentsPane{
		keys: agentKeys{
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
			),
		},
	}
}

func (p *AgentsPane) ID() PaneID        { return PaneAgents }
func (p *AgentsPane) Title() string      { return "Agents" }
func (p *AgentsPane) ShortTitle() string { return "🤖" }

// Badge returns the count of active (non-idle) agents.
func (p *AgentsPane) Badge() int {
	count := 0
	for _, a := range p.agents {
		if a.Status != "idle" {
			count++
		}
	}
	return count
}

func (p *AgentsPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.clampScroll()
}

func (p *AgentsPane) Init() tea.Cmd {
	return nil
}

func (p *AgentsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AgentUpdateMsg:
		p.agents = msg.Agents
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
			if p.cursor < len(p.agents)-1 {
				p.cursor++
				p.scrollToCursor()
			}
		}
	}
	return p, nil
}

func (p *AgentsPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header line
	running := p.Badge()
	header := fmt.Sprintf("─── AGENTS (%d running) ───", running)
	b.WriteString(theme.PaneHeaderStyle.Render(TruncateWithEllipsis(header, p.width)))
	b.WriteString("\n")

	if p.err != nil {
		b.WriteString(theme.FailStyle.Render("  Error: " + p.err.Error()))
		return b.String()
	}

	if len(p.agents) == 0 {
		b.WriteString(theme.MutedStyle.Render("  No agents running"))
		return b.String()
	}

	// Content area (height minus header and footer)
	contentHeight := p.height - 2 // header + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render visible agent rows
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
	footer := theme.MutedStyle.Render("j/k to scroll  enter for detail")
	b.WriteString(TruncateWithEllipsis(footer, p.width))

	return b.String()
}

// renderRows produces one display row per agent (with an optional second line for current work).
func (p *AgentsPane) renderRows() []string {
	var rows []string
	for i, a := range p.agents {
		icon := statusIcon(a.Status)
		name := a.Name
		role := a.Role
		age := FormatAge(a.Age)

		// Build the main line: "  ● name     role    age"
		selected := i == p.cursor
		line := formatAgentRow(icon, name, role, age, p.width, selected)
		rows = append(rows, line)

		// If agent has current work, show it on a second line
		if a.IssueID != "" {
			workLine := formatWorkLine(a.IssueID, a.IssueTitle, p.width, selected)
			rows = append(rows, workLine)
		}
	}
	return rows
}

func formatAgentRow(icon, name, role, age string, width int, selected bool) string {
	// Layout: "  <icon> <name>  <role>  <age>"
	// Minimum: 2 + icon(1-3) + 1 + name + 2 + role + 2 + age
	nameCol := 12
	roleCol := 10
	ageCol := 8

	nameStr := padOrTruncate(name, nameCol)
	roleStr := padOrTruncate(role, roleCol)
	ageStr := padOrTruncate(age, ageCol)

	line := fmt.Sprintf("  %s %s%s%s", icon, nameStr, roleStr, ageStr)

	if selected {
		return theme.AccentStyle.Bold(true).Render(line)
	}
	return line
}

func formatWorkLine(issueID, issueTitle string, width int, selected bool) string {
	// Indent to align with name column: "      <issueID>: <title>"
	maxTitleLen := width - 8 - len(issueID) - 2
	if maxTitleLen < 0 {
		maxTitleLen = 0
	}
	title := TruncateWithEllipsis(issueTitle, maxTitleLen)
	line := fmt.Sprintf("      %s: %s", issueID, title)

	style := theme.MutedStyle
	if selected {
		style = theme.AccentStyle
	}
	return style.Render(line)
}

func statusIcon(status string) string {
	switch status {
	case "working":
		return theme.IconWorking
	case "stale":
		return theme.IconStale
	case "stuck":
		return theme.IconStuck
	default: // idle
		return theme.IconIdle
	}
}

func padOrTruncate(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

// scrollToCursor ensures the cursor row is visible in the viewport.
func (p *AgentsPane) scrollToCursor() {
	// Compute the row index for the cursor agent (accounting for work lines above it)
	row := 0
	for i := 0; i < p.cursor && i < len(p.agents); i++ {
		row++ // agent row
		if p.agents[i].IssueID != "" {
			row++ // work line
		}
	}

	contentHeight := p.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	if row < p.offset {
		p.offset = row
	}
	// Ensure the agent row plus its potential work line are visible
	rowEnd := row
	if p.cursor < len(p.agents) && p.agents[p.cursor].IssueID != "" {
		rowEnd++
	}
	if rowEnd >= p.offset+contentHeight {
		p.offset = rowEnd - contentHeight + 1
	}
	p.clampScroll()
}

// clampScroll ensures offset stays in valid range.
func (p *AgentsPane) clampScroll() {
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
	// Also clamp cursor
	if p.cursor >= len(p.agents) {
		p.cursor = len(p.agents) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// RoleIcon returns the emoji icon for a given agent role.
func RoleIcon(role string) string {
	switch role {
	case "witness":
		return theme.RoleWitness
	case "refinery":
		return theme.RoleRefinery
	case "polecat":
		return theme.RolePolecat
	case "crew":
		return theme.RoleCrew
	case "mayor":
		return theme.RoleMayor
	default:
		return ""
	}
}

// AgentStatusFromAge returns "working", "stale", or "stuck" based on activity age.
// <5m = working (green), <30m = stale (yellow), >30m = stuck (red).
func AgentStatusFromAge(age time.Duration) string {
	switch {
	case age < 5*time.Minute:
		return "working"
	case age < 30*time.Minute:
		return "stale"
	default:
		return "stuck"
	}
}

// Ensure AgentsPane implements Pane at compile time.
var _ Pane = (*AgentsPane)(nil)

// Ensure AgentUpdateMsg implements tea.Msg.
var _ tea.Msg = AgentUpdateMsg{}

// DetectRole determines the agent role from a tmux session name.
// Session format: gt-{rig}-{name}
// Known roles: witness, refinery; everything else is a polecat.
func DetectRole(name string) string {
	switch name {
	case "witness":
		return "witness"
	case "refinery":
		return "refinery"
	case "mayor":
		return "mayor"
	default:
		return "polecat"
	}
}

