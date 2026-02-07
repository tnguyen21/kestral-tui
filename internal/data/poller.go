package data

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tick messages trigger fetches on the bubbletea event loop.
type StatusTickMsg time.Time
type AgentTickMsg time.Time
type ConvoyTickMsg time.Time
type ResourceTickMsg time.Time

// Result messages carry fetched data back to the model.
type StatusUpdateMsg struct {
	Status *TownStatus
	Err    error
}

type AgentUpdateMsg struct {
	Sessions []SessionInfo
	Err      error
}

type ConvoyUpdateMsg struct {
	Convoys []ConvoyInfo
	Err     error
}

// ScheduleStatusPoll returns a tea.Tick command for the next status poll.
func ScheduleStatusPoll(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return StatusTickMsg(t)
	})
}

// FetchStatusCmd returns a tea.Cmd that fetches status in the background.
func FetchStatusCmd(fetcher *Fetcher) tea.Cmd {
	return func() tea.Msg {
		status, err := fetcher.FetchStatus()
		return StatusUpdateMsg{Status: status, Err: err}
	}
}

// ScheduleAgentPoll returns a tea.Tick command for the next agent poll.
func ScheduleAgentPoll(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return AgentTickMsg(t)
	})
}

// FetchAgentsCmd returns a tea.Cmd that fetches agent sessions in the background.
func FetchAgentsCmd(fetcher *Fetcher) tea.Cmd {
	return func() tea.Msg {
		sessions, err := fetcher.FetchSessions()
		return AgentUpdateMsg{Sessions: sessions, Err: err}
	}
}

// ScheduleConvoyPoll returns a tea.Tick command for the next convoy poll.
func ScheduleConvoyPoll(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return ConvoyTickMsg(t)
	})
}

// FetchConvoysCmd returns a tea.Cmd that fetches convoys in the background.
func FetchConvoysCmd(fetcher *Fetcher) tea.Cmd {
	return func() tea.Msg {
		convoys, err := fetcher.FetchConvoys()
		return ConvoyUpdateMsg{Convoys: convoys, Err: err}
	}
}

// ScheduleResourcePoll returns a tea.Tick command for the next resource poll.
func ScheduleResourcePoll(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return ResourceTickMsg(t)
	})
}
