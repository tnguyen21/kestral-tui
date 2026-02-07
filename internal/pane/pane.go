package pane

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaneID identifies each TUI pane.
type PaneID int

const (
	PaneDashboard PaneID = iota
	PaneAgents
	PaneRefinery
	PanePRs
	PaneConvoys
	PaneMayor
	PaneMail
	PaneHistory
	PaneCI
	PaneLogs
	PaneResources
	PaneNewIssue
	PaneWitness
)

// Pane is the interface that all TUI panes implement.
type Pane interface {
	tea.Model
	ID() PaneID
	Title() string      // full title for wide mode (e.g., "Dashboard")
	ShortTitle() string // emoji/icon for narrow mode (e.g., "🏠")
	Badge() int         // notification count (0 = hidden)
	SetSize(w, h int)   // called on resize
}

// TruncateWithEllipsis truncates s to maxLen, appending "…" if truncated.
// If maxLen < 1, returns an empty string.
func TruncateWithEllipsis(s string, maxLen int) string {
	if maxLen < 1 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// FormatAge formats a duration as a human-readable age string.
func FormatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// StatusColor maps a work status string to a lipgloss style with the
// appropriate foreground color.
func StatusColor(status string) lipgloss.Style {
	base := lipgloss.NewStyle()
	switch status {
	case "OPEN", "PENDING":
		return base.Foreground(lipgloss.Color("12")) // blue
	case "IN_PROGRESS":
		return base.Foreground(lipgloss.Color("11")) // yellow
	case "COMPLETED", "CLOSED":
		return base.Foreground(lipgloss.Color("10")) // green
	case "BLOCKED":
		return base.Foreground(lipgloss.Color("9")) // red
	case "ESCALATED":
		return base.Foreground(lipgloss.Color("13")) // magenta
	default:
		return base.Foreground(lipgloss.Color("7")) // white/default
	}
}
