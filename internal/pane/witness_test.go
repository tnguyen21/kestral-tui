package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewWitnessPane(t *testing.T) {
	p := NewWitnessPane()
	if p.ID() != PaneWitness {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneWitness)
	}
	if p.Title() != "Witnesses" {
		t.Errorf("Title() = %q, want %q", p.Title(), "Witnesses")
	}
	if p.ShortTitle() != "👁" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "👁")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestWitnessPaneBadge(t *testing.T) {
	p := NewWitnessPane()
	p.witnesses = []WitnessInfo{
		{Rig: "gastown", Status: "alive"},
		{Rig: "longeye", Status: "stale"},
		{Rig: "beads", Status: "dead"},
	}
	// Badge counts non-alive witnesses
	if got := p.Badge(); got != 2 {
		t.Errorf("Badge() = %d, want 2 (stale + dead)", got)
	}
}

func TestWitnessPaneUpdateWithData(t *testing.T) {
	p := NewWitnessPane()
	p.SetSize(80, 24)

	witnesses := []WitnessInfo{
		{Rig: "gastown", Status: "alive", LastHeartbeat: 2 * time.Minute, PolecatCount: 3, HasSession: true, Uptime: time.Hour},
		{Rig: "longeye", Status: "stale", LastHeartbeat: 8 * time.Minute, PolecatCount: 1, HasSession: true, Uptime: 2 * time.Hour},
		{Rig: "beads", Status: "dead", PolecatCount: 0, HasSession: false},
	}

	updated, _ := p.Update(WitnessUpdateMsg{Witnesses: witnesses})
	p = updated.(*WitnessPane)

	if len(p.witnesses) != 3 {
		t.Fatalf("witnesses count = %d, want 3", len(p.witnesses))
	}
	if p.Badge() != 2 {
		t.Errorf("Badge() = %d, want 2", p.Badge())
	}
}

func TestWitnessPaneView(t *testing.T) {
	p := NewWitnessPane()
	p.SetSize(80, 24)

	witnesses := []WitnessInfo{
		{Rig: "gastown", Status: "alive", LastHeartbeat: 2 * time.Minute, PolecatCount: 3, HasSession: true, Uptime: time.Hour},
		{Rig: "longeye", Status: "dead", PolecatCount: 0, HasSession: false},
	}
	p.Update(WitnessUpdateMsg{Witnesses: witnesses})

	view := p.View()

	if !strings.Contains(view, "WITNESS HEARTBEAT") {
		t.Error("View should contain 'WITNESS HEARTBEAT' header")
	}
	if !strings.Contains(view, "gastown") {
		t.Error("View should contain rig name 'gastown'")
	}
	if !strings.Contains(view, "longeye") {
		t.Error("View should contain rig name 'longeye'")
	}
	if !strings.Contains(view, "no session") {
		t.Error("View should show 'no session' for dead witness without session")
	}
	if !strings.Contains(view, "j/k to scroll") {
		t.Error("View should contain scroll help footer")
	}
}

func TestWitnessPaneViewEmpty(t *testing.T) {
	p := NewWitnessPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No witness sessions detected") {
		t.Error("Empty pane should show 'No witness sessions detected'")
	}
}

func TestWitnessPaneViewError(t *testing.T) {
	p := NewWitnessPane()
	p.SetSize(80, 24)

	p.Update(WitnessUpdateMsg{Err: &testWitnessError{}})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

type testWitnessError struct{}

func (e *testWitnessError) Error() string { return "witness test error" }

func TestWitnessPaneScrolling(t *testing.T) {
	p := NewWitnessPane()
	p.SetSize(80, 5)

	var witnesses []WitnessInfo
	for i := 0; i < 20; i++ {
		witnesses = append(witnesses, WitnessInfo{
			Rig:    "rig" + string(rune('a'+i)),
			Status: "alive",
		})
	}
	p.Update(WitnessUpdateMsg{Witnesses: witnesses})

	if p.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", p.cursor)
	}

	for i := 0; i < 5; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if p.cursor != 5 {
		t.Errorf("cursor after 5x down = %d, want 5", p.cursor)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 4 {
		t.Errorf("cursor after up = %d, want 4", p.cursor)
	}

	p.cursor = 0
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", p.cursor)
	}
}

func TestWitnessStatusFromAge(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"alive", 2 * time.Minute, "alive"},
		{"boundary alive", 4*time.Minute + 59*time.Second, "alive"},
		{"stale", 8 * time.Minute, "stale"},
		{"boundary stale", 14*time.Minute + 59*time.Second, "stale"},
		{"dead", 15 * time.Minute, "dead"},
		{"very dead", 2 * time.Hour, "dead"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WitnessStatusFromAge(tt.age)
			if got != tt.want {
				t.Errorf("WitnessStatusFromAge(%v) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func TestWitnessPaneSetSize(t *testing.T) {
	p := NewWitnessPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestWitnessPaneViewZeroSize(t *testing.T) {
	p := NewWitnessPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"minutes", 30 * time.Minute, "30m"},
		{"hours", 2*time.Hour + 15*time.Minute, "2h15m"},
		{"days", 26*time.Hour + 30*time.Minute, "1d2h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.d)
			if got != tt.want {
				t.Errorf("formatUptime(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
