package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
)

func TestNewResourcesPane(t *testing.T) {
	p := NewResourcesPane()
	if p.ID() != PaneResources {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneResources)
	}
	if p.Title() != "Resources" {
		t.Errorf("Title() = %q, want %q", p.Title(), "Resources")
	}
	if p.ShortTitle() != "📊" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "📊")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestResourcesPaneUpdateWithData(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 24)

	now := time.Now()
	sessions := []data.SessionResource{
		{Name: "gt-kestral-witness", CPUPercent: 12.5, MemRSS: 50 << 20, ProcessCount: 3, UptimeSecs: 3600, ActivityTS: now.Unix()},
		{Name: "gt-kestral-obsidian", CPUPercent: 45.0, MemRSS: 200 << 20, ProcessCount: 8, UptimeSecs: 7200, ActivityTS: now.Unix()},
	}

	updated, _ := p.Update(ResourceUpdateMsg{Sessions: sessions})
	p = updated.(*ResourcesPane)

	if len(p.sessions) != 2 {
		t.Fatalf("sessions count = %d, want 2", len(p.sessions))
	}
}

func TestResourcesPaneView(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 24)

	now := time.Now()
	sessions := []data.SessionResource{
		{Name: "gt-kestral-witness", CPUPercent: 12.5, MemRSS: 50 << 20, ProcessCount: 3, UptimeSecs: 3600, ActivityTS: now.Unix()},
		{Name: "gt-kestral-obsidian", CPUPercent: 45.0, MemRSS: 200 << 20, ProcessCount: 8, UptimeSecs: 7200, ActivityTS: now.Unix()},
	}
	p.Update(ResourceUpdateMsg{Sessions: sessions})

	view := p.View()

	if !strings.Contains(view, "RESOURCES") {
		t.Error("View should contain 'RESOURCES' header")
	}
	if !strings.Contains(view, "SESSION") {
		t.Error("View should contain 'SESSION' column header")
	}
	if !strings.Contains(view, "CPU%") {
		t.Error("View should contain 'CPU%' column header")
	}
}

func TestResourcesPaneViewEmpty(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 24)

	view := p.View()
	if !strings.Contains(view, "No tmux sessions") {
		t.Error("Empty pane should show 'No tmux sessions'")
	}
}

func TestResourcesPaneViewZeroSize(t *testing.T) {
	p := NewResourcesPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestResourcesPaneViewError(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 24)

	p.Update(ResourceUpdateMsg{Err: &testError{}})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

func TestResourcesPaneBadgeAlerts(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 24)

	// Session with stale activity (>15 min ago)
	staleTS := time.Now().Add(-20 * time.Minute).Unix()
	sessions := []data.SessionResource{
		{Name: "healthy-session", CPUPercent: 10, MemRSS: 100 << 20, ProcessCount: 2, UptimeSecs: 600, ActivityTS: time.Now().Unix()},
		{Name: "stale-session", CPUPercent: 5, MemRSS: 50 << 20, ProcessCount: 1, UptimeSecs: 1200, ActivityTS: staleTS},
	}
	p.Update(ResourceUpdateMsg{Sessions: sessions})

	if got := p.Badge(); got != 1 {
		t.Errorf("Badge() = %d, want 1 (stale session)", got)
	}
}

func TestResourcesPaneScrolling(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 8) // Tiny viewport

	var sessions []data.SessionResource
	for i := 0; i < 20; i++ {
		sessions = append(sessions, data.SessionResource{
			Name:         "session-" + string(rune('a'+i)),
			CPUPercent:   float64(i * 5),
			MemRSS:       int64(i) * (1 << 20),
			ProcessCount: i + 1,
			UptimeSecs:   int64(i) * 60,
			ActivityTS:   time.Now().Unix(),
		})
	}
	p.Update(ResourceUpdateMsg{Sessions: sessions})

	if p.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", p.cursor)
	}

	// Move down
	for i := 0; i < 5; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if p.cursor != 5 {
		t.Errorf("cursor after 5x down = %d, want 5", p.cursor)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 4 {
		t.Errorf("cursor after up = %d, want 4", p.cursor)
	}

	// Can't go below 0
	p.cursor = 0
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", p.cursor)
	}
}

func TestResourcesPaneSortCycling(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 24)

	sessions := []data.SessionResource{
		{Name: "beta", CPUPercent: 20, MemRSS: 300 << 20, ActivityTS: time.Now().Unix()},
		{Name: "alpha", CPUPercent: 50, MemRSS: 100 << 20, ActivityTS: time.Now().Unix()},
		{Name: "gamma", CPUPercent: 10, MemRSS: 200 << 20, ActivityTS: time.Now().Unix()},
	}
	p.Update(ResourceUpdateMsg{Sessions: sessions})

	// Default sort is CPU (descending)
	if p.sortBy != sortByCPU {
		t.Errorf("default sortBy = %d, want sortByCPU", p.sortBy)
	}
	if p.sessions[0].Name != "alpha" {
		t.Errorf("CPU sort: first = %q, want 'alpha' (highest CPU)", p.sessions[0].Name)
	}

	// Press 's' to cycle to memory sort
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if p.sortBy != sortByMem {
		t.Errorf("after s: sortBy = %d, want sortByMem", p.sortBy)
	}
	if p.sessions[0].Name != "beta" {
		t.Errorf("MEM sort: first = %q, want 'beta' (highest MEM)", p.sessions[0].Name)
	}

	// Press 's' to cycle to name sort
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if p.sortBy != sortByName {
		t.Errorf("after 2nd s: sortBy = %d, want sortByName", p.sortBy)
	}
	if p.sessions[0].Name != "alpha" {
		t.Errorf("name sort: first = %q, want 'alpha' (alphabetical)", p.sessions[0].Name)
	}

	// Press 's' to wrap back to CPU sort
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if p.sortBy != sortByCPU {
		t.Errorf("after 3rd s: sortBy = %d, want sortByCPU (wrapped)", p.sortBy)
	}
}

func TestResourcesPaneSetSize(t *testing.T) {
	p := NewResourcesPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestSessionHistory(t *testing.T) {
	h := &sessionHistory{}

	// Add samples
	for i := 0; i < 15; i++ {
		h.addSample(float64(i * 10))
	}

	// Should only keep last 10
	if len(h.cpuSamples) != maxSamples {
		t.Errorf("samples count = %d, want %d", len(h.cpuSamples), maxSamples)
	}

	// Most recent should be 140 (14*10)
	last := h.cpuSamples[len(h.cpuSamples)-1]
	if last != 140 {
		t.Errorf("last sample = %f, want 140", last)
	}
}

func TestSessionHistorySustainedAbove(t *testing.T) {
	h := &sessionHistory{}

	// Not enough samples
	h.addSample(90)
	if h.sustainedAbove(80, 4) {
		t.Error("should return false with insufficient samples")
	}

	// Add more high samples
	for i := 0; i < 5; i++ {
		h.addSample(85)
	}
	if !h.sustainedAbove(80, 4) {
		t.Error("should return true with 4+ samples above 80")
	}

	// Add a low sample to break the streak
	h.addSample(50)
	if h.sustainedAbove(80, 4) {
		t.Error("should return false after low sample breaks streak")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{500, "500B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1 << 20, "1.0M"},
		{150 << 20, "150.0M"},
		{1 << 30, "1.0G"},
		{int64(2.5 * float64(1<<30)), "2.5G"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{25 * time.Hour, "1d1h"},
		{48 * time.Hour, "2d0h"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatUptime(tt.dur)
			if got != tt.want {
				t.Errorf("formatUptime(%v) = %q, want %q", tt.dur, got, tt.want)
			}
		})
	}
}

func TestSessionStatus(t *testing.T) {
	p := NewResourcesPane()
	now := time.Now()

	// Healthy session
	status := p.sessionStatus("healthy", 10.0, now.Unix())
	if status != "healthy" {
		t.Errorf("sessionStatus for low CPU = %q, want 'healthy'", status)
	}

	// Stale session (no activity for >15 min)
	staleTS := now.Add(-20 * time.Minute).Unix()
	status = p.sessionStatus("stale", 10.0, staleTS)
	if status != "stale" {
		t.Errorf("sessionStatus for stale activity = %q, want 'stale'", status)
	}

	// High instantaneous CPU (above alert threshold but not sustained)
	status = p.sessionStatus("high-cpu", 96.0, now.Unix())
	if status != "warning" {
		t.Errorf("sessionStatus for high instant CPU = %q, want 'warning'", status)
	}
}

func TestResourcesPaneSparkline(t *testing.T) {
	p := NewResourcesPane()

	// Empty history should show placeholder
	spark := p.renderSparkline("unknown")
	if !strings.Contains(spark, "░") {
		t.Error("empty sparkline should contain placeholder chars")
	}

	// Add some history
	h := &sessionHistory{}
	for i := 0; i < 5; i++ {
		h.addSample(float64(i * 20))
	}
	p.history["test"] = h

	spark = p.renderSparkline("test")
	if spark == "" {
		t.Error("sparkline should not be empty with history data")
	}
}

// Ensure ResourcesPane implements Pane at compile time.
var _ Pane = (*ResourcesPane)(nil)
