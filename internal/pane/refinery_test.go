package pane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
)

func TestNewRefineryPane(t *testing.T) {
	p := NewRefineryPane()
	if p.ID() != PaneRefinery {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneRefinery)
	}
	if p.Title() != "Refinery" {
		t.Errorf("Title() = %q, want %q", p.Title(), "Refinery")
	}
	if p.ShortTitle() != "🔧" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "🔧")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestRefineryPaneBadge(t *testing.T) {
	p := NewRefineryPane()
	p.statuses = []data.RefineryStatus{
		{Rig: "rig1", QueueDepth: 3},
		{Rig: "rig2", QueueDepth: 2},
	}
	if got := p.Badge(); got != 5 {
		t.Errorf("Badge() = %d, want 5", got)
	}
}

func TestRefineryPaneBadgeEmpty(t *testing.T) {
	p := NewRefineryPane()
	p.statuses = []data.RefineryStatus{
		{Rig: "rig1", QueueDepth: 0},
	}
	if got := p.Badge(); got != 0 {
		t.Errorf("Badge() = %d, want 0", got)
	}
}

func TestRefineryPaneUpdateWithData(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	statuses := []data.RefineryStatus{
		{
			Rig:        "kestral_tui",
			Running:    true,
			QueueDepth: 2,
			Current: &data.MergeRequest{
				ID:     "mr-1",
				BeadID: "kt-abc",
				Title:  "Fix auth bug",
				Status: "testing",
				Branch: "polecat/quartz/kt-abc",
			},
			Queue: []data.MergeRequest{
				{ID: "mr-2", BeadID: "kt-def", Title: "Add feature"},
			},
			History: []data.MergeRequest{
				{ID: "mr-0", BeadID: "kt-xyz", Title: "Previous merge", Status: "merged"},
			},
			SuccessRate: 95.0,
		},
	}

	updated, _ := p.Update(RefineryUpdateMsg{Statuses: statuses})
	p = updated.(*RefineryPane)

	if len(p.statuses) != 1 {
		t.Fatalf("statuses count = %d, want 1", len(p.statuses))
	}
	if p.Badge() != 2 {
		t.Errorf("Badge() = %d, want 2", p.Badge())
	}
}

func TestRefineryPaneViewEmpty(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No refineries active") {
		t.Error("Empty pane should show 'No refineries active'")
	}
}

func TestRefineryPaneViewZeroSize(t *testing.T) {
	p := NewRefineryPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestRefineryPaneViewError(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	p.Update(RefineryUpdateMsg{Err: errTest})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

func TestRefineryPaneViewWithData(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 30)

	statuses := []data.RefineryStatus{
		{
			Rig:        "kestral_tui",
			Running:    true,
			QueueDepth: 2,
			Current: &data.MergeRequest{
				BeadID: "kt-abc",
				Title:  "Fix auth bug",
				Status: "testing",
				Branch: "polecat/quartz/kt-abc",
			},
			Queue: []data.MergeRequest{
				{BeadID: "kt-def", Title: "Add feature"},
			},
			History: []data.MergeRequest{
				{BeadID: "kt-xyz", Title: "Previous merge", Status: "merged"},
			},
			SuccessRate: 95.0,
		},
	}

	p.Update(RefineryUpdateMsg{Statuses: statuses})
	view := p.View()

	// Header should contain REFINERY
	if !strings.Contains(view, "REFINERY") {
		t.Error("View should contain 'REFINERY' header")
	}

	// Should show current MR
	if !strings.Contains(view, "CURRENT") {
		t.Error("View should contain 'CURRENT' section")
	}
	if !strings.Contains(view, "kt-abc") {
		t.Error("View should contain current MR bead ID")
	}

	// Should show queue
	if !strings.Contains(view, "QUEUE") {
		t.Error("View should contain 'QUEUE' section")
	}
	if !strings.Contains(view, "kt-def") {
		t.Error("View should contain queued MR bead ID")
	}

	// Should show metrics
	if !strings.Contains(view, "METRICS") {
		t.Error("View should contain 'METRICS' section")
	}

	// Should show history
	if !strings.Contains(view, "HISTORY") {
		t.Error("View should contain 'HISTORY' section")
	}

	// Footer
	if !strings.Contains(view, "j/k to scroll") {
		t.Error("View should contain scroll help footer")
	}
}

func TestRefineryPaneViewIdleCurrent(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	statuses := []data.RefineryStatus{
		{Rig: "rig1", Running: true, Current: nil},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})
	view := p.View()

	if !strings.Contains(view, "(idle)") {
		t.Error("Should show '(idle)' when no current MR")
	}
}

func TestRefineryPaneViewEmptyQueue(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	statuses := []data.RefineryStatus{
		{Rig: "rig1", Running: true, Queue: nil},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})
	view := p.View()

	if !strings.Contains(view, "(empty)") {
		t.Error("Should show '(empty)' when queue is empty")
	}
}

func TestRefineryPaneViewNoHistory(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	statuses := []data.RefineryStatus{
		{Rig: "rig1", Running: true, History: nil},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})
	view := p.View()

	if !strings.Contains(view, "(no history)") {
		t.Error("Should show '(no history)' when history is empty")
	}
}

func TestRefineryPaneScrolling(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 5)

	statuses := []data.RefineryStatus{
		{
			Rig:     "rig1",
			Running: true,
			Queue: []data.MergeRequest{
				{BeadID: "mr-1", Title: "MR 1"},
				{BeadID: "mr-2", Title: "MR 2"},
				{BeadID: "mr-3", Title: "MR 3"},
				{BeadID: "mr-4", Title: "MR 4"},
				{BeadID: "mr-5", Title: "MR 5"},
			},
			History: []data.MergeRequest{
				{BeadID: "mr-h1", Title: "Hist 1", Status: "merged"},
				{BeadID: "mr-h2", Title: "Hist 2", Status: "failed"},
			},
		},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})

	if p.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", p.cursor)
	}

	// Move down multiple times
	for i := 0; i < 5; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if p.cursor < 1 {
		t.Errorf("cursor after scrolling down = %d, want > 0", p.cursor)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	curAfterUp := p.cursor

	// Can't go below 0
	p.cursor = 0
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", p.cursor)
	}

	_ = curAfterUp // used for verification above
}

func TestRefineryPaneMultiRig(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 30)

	statuses := []data.RefineryStatus{
		{Rig: "rig1", Running: true, QueueDepth: 1},
		{Rig: "rig2", Running: false, QueueDepth: 3},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})

	// Should show rig tabs when multiple rigs
	view := p.View()
	if !strings.Contains(view, "rig1") {
		t.Error("View should contain first rig name")
	}
	if !strings.Contains(view, "rig2") {
		t.Error("View should contain second rig name")
	}
	if !strings.Contains(view, "h/l switch rig") {
		t.Error("Footer should mention rig switching")
	}

	// Switch to rig2 with 'l' key
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if p.rigIdx != 1 {
		t.Errorf("rigIdx after 'l' = %d, want 1", p.rigIdx)
	}

	// Switch back with 'h' key
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if p.rigIdx != 0 {
		t.Errorf("rigIdx after 'h' = %d, want 0", p.rigIdx)
	}

	// Can't go past bounds
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if p.rigIdx != 0 {
		t.Errorf("rigIdx after 'h' at 0 = %d, want 0", p.rigIdx)
	}
}

func TestRefineryPaneSingleRigNoTabs(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)

	statuses := []data.RefineryStatus{
		{Rig: "onlyrig", Running: true},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})

	view := p.View()
	// Single rig should not show h/l switcher
	if strings.Contains(view, "h/l switch rig") {
		t.Error("Single-rig view should not show rig switching hint")
	}
}

func TestRefineryPaneSetSize(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestTestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		empty  bool // just verify it returns something
	}{
		{"testing", false},
		{"IN_PROGRESS", false},
		{"merged", false},
		{"COMPLETED", false},
		{"failed", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := testStatusIcon(tt.status)
			if got == "" {
				t.Errorf("testStatusIcon(%q) returned empty string", tt.status)
			}
		})
	}
}

func TestHistoryIcon(t *testing.T) {
	tests := []struct {
		status string
		empty  bool
	}{
		{"merged", false},
		{"COMPLETED", false},
		{"failed", false},
		{"skipped", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := historyIcon(tt.status)
			if got == "" {
				t.Errorf("historyIcon(%q) returned empty string", tt.status)
			}
		})
	}
}

func TestQueueWaitEstimate(t *testing.T) {
	tests := []struct {
		name   string
		status data.RefineryStatus
		want   string
	}{
		{"empty queue", data.RefineryStatus{QueueDepth: 0}, "—"},
		{"unknown avg", data.RefineryStatus{QueueDepth: 3, AvgMergeTime: 0}, "unknown"},
		{"minutes", data.RefineryStatus{QueueDepth: 3, AvgMergeTime: 120}, "~6m"},
		{"hours", data.RefineryStatus{QueueDepth: 10, AvgMergeTime: 600}, "~1h"},
		{"seconds", data.RefineryStatus{QueueDepth: 1, AvgMergeTime: 30}, "~30s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queueWaitEstimate(tt.status)
			if got != tt.want {
				t.Errorf("queueWaitEstimate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefineryPaneRigIdxClamped(t *testing.T) {
	p := NewRefineryPane()
	p.SetSize(80, 24)
	p.rigIdx = 5 // out of bounds

	statuses := []data.RefineryStatus{
		{Rig: "rig1", Running: true},
	}
	p.Update(RefineryUpdateMsg{Statuses: statuses})

	if p.rigIdx != 0 {
		t.Errorf("rigIdx should be clamped to 0 when out of bounds, got %d", p.rigIdx)
	}
}
