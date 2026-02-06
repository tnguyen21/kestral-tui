package pane

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/theme"
)

// StatusUpdateMsg delivers town status data to the dashboard.
type StatusUpdateMsg struct {
	Status   *data.TownStatus
	Sessions []data.SessionInfo
	FetchedAt time.Time
}

// ConvoyUpdateMsg delivers convoy data to the dashboard.
type ConvoyUpdateMsg struct {
	Convoys []data.ConvoyInfo
	// Progress maps convoy ID to (done, total) counts.
	Progress map[string][2]int
}

// Dashboard is the home screen pane showing system health at a glance.
type Dashboard struct {
	width    int
	height   int
	viewport viewport.Model

	// Cached data
	status   *data.TownStatus
	sessions []data.SessionInfo
	convoys  []data.ConvoyInfo
	progress map[string][2]int // convoy ID -> (done, total)
	lastUpdate time.Time
}

// NewDashboard creates a new Dashboard pane.
func NewDashboard() *Dashboard {
	vp := viewport.New(0, 0)
	return &Dashboard{
		viewport: vp,
		progress: make(map[string][2]int),
	}
}

func (d *Dashboard) ID() PaneID        { return PaneDashboard }
func (d *Dashboard) Title() string      { return "Dashboard" }
func (d *Dashboard) ShortTitle() string { return "\U0001F3E0" } // 🏠
func (d *Dashboard) Badge() int         { return 0 }

func (d *Dashboard) SetSize(w, h int) {
	d.width = w
	d.height = h
	d.viewport.Width = w
	d.viewport.Height = h
	d.viewport.SetContent(d.renderContent())
}

func (d *Dashboard) Init() tea.Cmd {
	return nil
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusUpdateMsg:
		d.status = msg.Status
		d.sessions = msg.Sessions
		d.lastUpdate = msg.FetchedAt
		d.viewport.SetContent(d.renderContent())
		return d, nil

	case ConvoyUpdateMsg:
		d.convoys = msg.Convoys
		d.progress = msg.Progress
		if d.progress == nil {
			d.progress = make(map[string][2]int)
		}
		d.viewport.SetContent(d.renderContent())
		return d, nil
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *Dashboard) View() string {
	return d.viewport.View()
}

// renderContent builds the full dashboard text.
func (d *Dashboard) renderContent() string {
	if d.width == 0 {
		return ""
	}

	var b strings.Builder

	d.renderHeader(&b)
	d.renderAgents(&b)
	d.renderConvoys(&b)
	d.renderSessions(&b)

	return b.String()
}

// renderHeader renders the title and health banner.
func (d *Dashboard) renderHeader(b *strings.Builder) {
	titleLine := theme.PaneHeaderStyle.Render(centerPad("KESTRAL", d.width))
	b.WriteString(titleLine)
	b.WriteByte('\n')
	b.WriteString(d.separator())
	b.WriteByte('\n')

	// Health banner
	healthLabel := "HEALTHY"
	healthStyle := theme.PassStyle
	if d.status != nil {
		stopped := 0
		for _, a := range d.status.Agents {
			if !a.Running {
				stopped++
			}
		}
		if stopped > 0 {
			healthLabel = "DEGRADED"
			healthStyle = theme.WarnStyle
		}
	} else {
		healthLabel = "LOADING"
		healthStyle = theme.MutedStyle
	}

	age := "…"
	if !d.lastUpdate.IsZero() {
		age = FormatAge(time.Since(d.lastUpdate))
	}

	healthLine := fmt.Sprintf("%s Town: %s    %s %s",
		theme.PassStyle.Render("⚡"),
		healthStyle.Render(healthLabel),
		theme.MutedStyle.Render("↻"),
		theme.MutedStyle.Render(age),
	)
	b.WriteString(TruncateWithEllipsis(healthLine, d.width))
	b.WriteByte('\n')
	b.WriteString(d.separator())
	b.WriteByte('\n')
}

// renderAgents renders the agent summary section.
func (d *Dashboard) renderAgents(b *strings.Builder) {
	if d.status == nil {
		b.WriteString(theme.MutedStyle.Render("AGENTS        …"))
		b.WriteByte('\n')
		b.WriteString(d.separator())
		b.WriteByte('\n')
		return
	}

	running := 0
	for _, a := range d.status.Agents {
		if a.Running {
			running++
		}
	}

	header := fmt.Sprintf("AGENTS        %d running", running)
	b.WriteString(theme.AccentStyle.Render(header))
	b.WriteByte('\n')

	for _, a := range d.status.Agents {
		icon := agentIcon(a)
		state := a.State
		if state == "" {
			if a.Running {
				state = "active"
			} else {
				state = "stopped"
			}
		}
		name := TruncateWithEllipsis(a.Name, d.width-6)
		line := fmt.Sprintf("  %s %s", icon, name)

		// Pad and add state right-aligned if there's room
		stateStr := theme.MutedStyle.Render(state)
		nameLen := 4 + len([]rune(a.Name)) // "  X "
		stateLen := len([]rune(state))
		if nameLen+stateLen+2 <= d.width {
			padding := d.width - nameLen - stateLen
			if padding < 1 {
				padding = 1
			}
			line = fmt.Sprintf("  %s %-*s%s", icon, d.width-4-stateLen, name, stateStr)
		}

		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(d.separator())
	b.WriteByte('\n')
}

// renderConvoys renders the convoy summary section with progress bars.
func (d *Dashboard) renderConvoys(b *strings.Builder) {
	if d.convoys == nil {
		b.WriteString(theme.MutedStyle.Render("CONVOYS       …"))
		b.WriteByte('\n')
		b.WriteString(d.separator())
		b.WriteByte('\n')
		return
	}

	openCount := len(d.convoys)
	header := fmt.Sprintf("CONVOYS       %d open", openCount)
	b.WriteString(theme.AccentStyle.Render(header))
	b.WriteByte('\n')

	if openCount == 0 {
		b.WriteString(theme.MutedStyle.Render("  (none)"))
		b.WriteByte('\n')
	}

	for _, c := range d.convoys {
		title := TruncateWithEllipsis(c.Title, d.width-16)
		done, total := 0, 0
		if p, ok := d.progress[c.ID]; ok {
			done, total = p[0], p[1]
		}

		bar := progressBar(done, total, 6)
		fraction := fmt.Sprintf("%d/%d", done, total)
		line := fmt.Sprintf("  %s %s %s",
			title,
			fraction,
			bar,
		)
		b.WriteString(TruncateWithEllipsis(line, d.width))
		b.WriteByte('\n')
	}

	b.WriteString(d.separator())
	b.WriteByte('\n')
}

// renderSessions renders the tmux session count.
func (d *Dashboard) renderSessions(b *strings.Builder) {
	count := len(d.sessions)
	label := "SESSIONS"
	if count > 0 {
		b.WriteString(fmt.Sprintf("%s      %d tmux",
			theme.AccentStyle.Render(label),
			count,
		))
	} else if d.status == nil {
		b.WriteString(theme.MutedStyle.Render(label + "      …"))
	} else {
		b.WriteString(fmt.Sprintf("%s      0 tmux",
			theme.AccentStyle.Render(label),
		))
	}
	b.WriteByte('\n')
}

// separator returns a dim horizontal rule.
func (d *Dashboard) separator() string {
	if d.width <= 0 {
		return ""
	}
	return theme.MutedStyle.Render(strings.Repeat("─", d.width))
}

// agentIcon returns the styled status icon for an agent.
func agentIcon(a data.AgentInfo) string {
	switch {
	case !a.Running:
		return theme.IconStuck
	case a.State == "idle":
		return theme.IconIdle
	default:
		return theme.IconWorking
	}
}

// progressBar renders a fixed-width progress bar using block characters.
func progressBar(done, total, barWidth int) string {
	if total <= 0 || barWidth <= 0 {
		return theme.MutedStyle.Render(strings.Repeat("░", barWidth))
	}

	filled := done * barWidth / total
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	if done >= total {
		return theme.PassStyle.Render(bar)
	}
	return lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Render(bar)
}

// centerPad centers text within width using spaces.
func centerPad(s string, width int) string {
	sLen := len([]rune(s))
	if sLen >= width {
		return s
	}
	left := (width - sLen) / 2
	right := width - sLen - left
	return strings.Repeat("─", left) + " " + s + " " + strings.Repeat("─", right)
}
