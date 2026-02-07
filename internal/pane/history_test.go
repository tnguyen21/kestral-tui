package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
)

func TestNewHistoryPane(t *testing.T) {
	p := NewHistoryPane()
	if p.ID() != PaneHistory {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneHistory)
	}
	if p.Title() != "History" {
		t.Errorf("Title() = %q, want %q", p.Title(), "History")
	}
	if p.ShortTitle() != "📜" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "📜")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestHistoryPaneUpdateWithData(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-abc", Title: "Fix bug", Status: "closed", IssueType: "bug",
			Assignee: "rig/polecats/amber", CreatedAt: "2026-02-07T00:00:00Z", ClosedAt: "2026-02-07T01:00:00Z"},
		{ID: "kt-def", Title: "Add feature", Status: "closed", IssueType: "feature",
			Assignee: "rig/polecats/ruby", CreatedAt: "2026-02-06T10:00:00Z", ClosedAt: "2026-02-06T12:00:00Z"},
	}

	updated, _ := p.Update(HistoryUpdateMsg{ClosedBeads: beads})
	p = updated.(*HistoryPane)

	if len(p.entries) != 2 {
		t.Fatalf("entries count = %d, want 2", len(p.entries))
	}
	if p.Badge() != 2 {
		t.Errorf("Badge() = %d, want 2", p.Badge())
	}
}

func TestHistoryPaneSortsMostRecentFirst(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-old", Title: "Old", ClosedAt: "2026-02-05T01:00:00Z", CreatedAt: "2026-02-05T00:00:00Z"},
		{ID: "kt-new", Title: "New", ClosedAt: "2026-02-07T01:00:00Z", CreatedAt: "2026-02-07T00:00:00Z"},
	}

	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	if len(p.entries) != 2 {
		t.Fatalf("entries count = %d, want 2", len(p.entries))
	}
	if p.entries[0].ID != "kt-new" {
		t.Errorf("first entry = %q, want kt-new (most recent)", p.entries[0].ID)
	}
	if p.entries[1].ID != "kt-old" {
		t.Errorf("second entry = %q, want kt-old", p.entries[1].ID)
	}
}

func TestHistoryPaneView(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-abc", Title: "Fix auth bug", Status: "closed", IssueType: "bug",
			Assignee: "rig/polecats/amber", CreatedAt: "2026-02-07T00:00:00Z", ClosedAt: "2026-02-07T01:00:00Z"},
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	view := p.View()

	if !strings.Contains(view, "HISTORY") {
		t.Error("View should contain 'HISTORY' header")
	}
	if !strings.Contains(view, "kt-abc") {
		t.Error("View should contain bead ID")
	}
	if !strings.Contains(view, "Fix auth bug") {
		t.Error("View should contain bead title")
	}
	if !strings.Contains(view, "amber") {
		t.Error("View should contain short assignee name")
	}
}

func TestHistoryPaneViewEmpty(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No completed work") {
		t.Error("Empty pane should show 'No completed work'")
	}
}

func TestHistoryPaneViewError(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	p.Update(HistoryUpdateMsg{Err: errTest})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

func TestHistoryPaneViewZeroSize(t *testing.T) {
	p := NewHistoryPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestHistoryPaneDateFilter(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	now := time.Now()
	todayStr := now.Format(time.RFC3339)
	oldStr := now.AddDate(0, 0, -10).Format(time.RFC3339)
	veryOldStr := now.AddDate(0, -2, 0).Format(time.RFC3339)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-today", Title: "Today", ClosedAt: todayStr, CreatedAt: todayStr},
		{ID: "kt-old", Title: "Old", ClosedAt: oldStr, CreatedAt: oldStr},
		{ID: "kt-vold", Title: "Very Old", ClosedAt: veryOldStr, CreatedAt: veryOldStr},
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	// Default: all
	if len(p.entries) != 3 {
		t.Errorf("all filter: entries = %d, want 3", len(p.entries))
	}

	// Cycle to "today"
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if p.filter.dateRange != "today" {
		t.Errorf("after f: filter = %q, want today", p.filter.dateRange)
	}
	if len(p.entries) != 1 {
		t.Errorf("today filter: entries = %d, want 1", len(p.entries))
	}

	// Cycle to "7d"
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if p.filter.dateRange != "7d" {
		t.Errorf("after f: filter = %q, want 7d", p.filter.dateRange)
	}
	if len(p.entries) != 1 {
		t.Errorf("7d filter: entries = %d, want 1", len(p.entries))
	}

	// Cycle to "30d"
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if p.filter.dateRange != "30d" {
		t.Errorf("after f: filter = %q, want 30d", p.filter.dateRange)
	}
	if len(p.entries) != 2 {
		t.Errorf("30d filter: entries = %d, want 2", len(p.entries))
	}

	// Cycle back to "all"
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if p.filter.dateRange != "all" {
		t.Errorf("after f: filter = %q, want all", p.filter.dateRange)
	}
	if len(p.entries) != 3 {
		t.Errorf("all filter: entries = %d, want 3", len(p.entries))
	}
}

func TestHistoryPaneAgentFilter(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-1", Title: "T1", Assignee: "rig/polecats/amber", ClosedAt: "2026-02-07T01:00:00Z", CreatedAt: "2026-02-07T00:00:00Z"},
		{ID: "kt-2", Title: "T2", Assignee: "rig/polecats/ruby", ClosedAt: "2026-02-07T02:00:00Z", CreatedAt: "2026-02-07T01:00:00Z"},
		{ID: "kt-3", Title: "T3", Assignee: "rig/polecats/amber", ClosedAt: "2026-02-07T03:00:00Z", CreatedAt: "2026-02-07T02:00:00Z"},
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	if len(p.entries) != 3 {
		t.Fatalf("all: entries = %d, want 3", len(p.entries))
	}

	// Cycle to first agent (alphabetical: amber)
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if p.filter.agent != "amber" {
		t.Errorf("after a: agent filter = %q, want amber", p.filter.agent)
	}
	if len(p.entries) != 2 {
		t.Errorf("amber filter: entries = %d, want 2", len(p.entries))
	}

	// Cycle to next agent (ruby)
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if p.filter.agent != "ruby" {
		t.Errorf("after a: agent filter = %q, want ruby", p.filter.agent)
	}
	if len(p.entries) != 1 {
		t.Errorf("ruby filter: entries = %d, want 1", len(p.entries))
	}

	// Cycle back to all
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if p.filter.agent != "" {
		t.Errorf("after a: agent filter = %q, want empty", p.filter.agent)
	}
}

func TestHistoryPaneTypeFilter(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-1", Title: "T1", IssueType: "bug", ClosedAt: "2026-02-07T01:00:00Z", CreatedAt: "2026-02-07T00:00:00Z"},
		{ID: "kt-2", Title: "T2", IssueType: "feature", ClosedAt: "2026-02-07T02:00:00Z", CreatedAt: "2026-02-07T01:00:00Z"},
		{ID: "kt-3", Title: "T3", IssueType: "bug", ClosedAt: "2026-02-07T03:00:00Z", CreatedAt: "2026-02-07T02:00:00Z"},
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	// Cycle to first type (alphabetical: bug)
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.filter.issueType != "bug" {
		t.Errorf("after t: type filter = %q, want bug", p.filter.issueType)
	}
	if len(p.entries) != 2 {
		t.Errorf("bug filter: entries = %d, want 2", len(p.entries))
	}

	// Cycle to feature
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.filter.issueType != "feature" {
		t.Errorf("after t: type filter = %q, want feature", p.filter.issueType)
	}
	if len(p.entries) != 1 {
		t.Errorf("feature filter: entries = %d, want 1", len(p.entries))
	}

	// Cycle back to all
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.filter.issueType != "" {
		t.Errorf("after t: type filter = %q, want empty", p.filter.issueType)
	}
}

func TestHistoryPaneScrolling(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 6) // Tiny viewport

	var beads []data.ClosedBeadInfo
	for i := 0; i < 20; i++ {
		ts := time.Date(2026, 2, 7, i, 0, 0, 0, time.UTC).Format(time.RFC3339)
		beads = append(beads, data.ClosedBeadInfo{
			ID: "kt-" + string(rune('a'+i)), Title: "Task " + string(rune('A'+i)),
			ClosedAt: ts, CreatedAt: ts,
		})
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

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

func TestHistoryPaneSetSize(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestHistoryPaneDayGrouping(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(100, 40)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-1", Title: "T1", ClosedAt: "2026-02-07T10:00:00Z", CreatedAt: "2026-02-07T09:00:00Z"},
		{ID: "kt-2", Title: "T2", ClosedAt: "2026-02-07T08:00:00Z", CreatedAt: "2026-02-07T07:00:00Z"},
		{ID: "kt-3", Title: "T3", ClosedAt: "2026-02-06T05:00:00Z", CreatedAt: "2026-02-06T04:00:00Z"},
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	rows := p.renderRows()

	// Should have 2 day headers (2026-02-07 and 2026-02-06) + 3 entry rows = 5 rows
	if len(rows) != 5 {
		t.Errorf("renderRows() = %d rows, want 5 (2 day headers + 3 entries)", len(rows))
	}

	// First row should be day header for most recent day
	if !strings.Contains(rows[0], "2026-02-07") {
		t.Errorf("first row should be day header for 2026-02-07, got %q", rows[0])
	}
}

func TestShortAssignee(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"rig/polecats/amber", "amber"},
		{"kestral_tui/polecats/ruby", "ruby"},
		{"witness", "witness"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shortAssignee(tt.input)
		if got != tt.want {
			t.Errorf("shortAssignee(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"2026-02-07T01:00:00Z", true},
		{"2026-02-07T01:00:00.123456789Z", true},
		{"2026-02-07", true},
		{"", false},
		{"not-a-date", false},
	}
	for _, tt := range tests {
		got := parseTime(tt.input)
		if tt.valid && got.IsZero() {
			t.Errorf("parseTime(%q) returned zero, want valid time", tt.input)
		}
		if !tt.valid && !got.IsZero() {
			t.Errorf("parseTime(%q) returned non-zero, want zero", tt.input)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{0, "—"},
		{-1 * time.Hour, "—"},
		{30 * time.Second, "<1m"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{2 * time.Hour, "2h"},
		{25 * time.Hour, "1d"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.dur)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.dur, got, tt.want)
		}
	}
}

func TestHistoryPaneFilterBar(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	bar := p.renderFilterBar()
	if !strings.Contains(bar, "date:all") {
		t.Errorf("filter bar should show 'date:all', got %q", bar)
	}

	// Set agent filter
	p.closedBeads = []data.ClosedBeadInfo{
		{ID: "kt-1", Assignee: "rig/polecats/amber", ClosedAt: "2026-02-07T01:00:00Z", CreatedAt: "2026-02-07T00:00:00Z"},
	}
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	bar = p.renderFilterBar()
	if !strings.Contains(bar, "agent:amber") {
		t.Errorf("filter bar should show 'agent:amber', got %q", bar)
	}
}

func TestHistoryPaneSkipsInvalidClosedAt(t *testing.T) {
	p := NewHistoryPane()
	p.SetSize(80, 24)

	beads := []data.ClosedBeadInfo{
		{ID: "kt-valid", Title: "Valid", ClosedAt: "2026-02-07T01:00:00Z", CreatedAt: "2026-02-07T00:00:00Z"},
		{ID: "kt-empty", Title: "No closed time", ClosedAt: "", CreatedAt: "2026-02-07T00:00:00Z"},
	}
	p.Update(HistoryUpdateMsg{ClosedBeads: beads})

	if len(p.entries) != 1 {
		t.Errorf("entries = %d, want 1 (should skip entry with empty ClosedAt)", len(p.entries))
	}
}

func TestHistoryPaneUniqueAgentsAndTypes(t *testing.T) {
	p := NewHistoryPane()
	p.closedBeads = []data.ClosedBeadInfo{
		{Assignee: "rig/polecats/amber", IssueType: "bug"},
		{Assignee: "rig/polecats/ruby", IssueType: "feature"},
		{Assignee: "rig/polecats/amber", IssueType: "bug"},
		{Assignee: "", IssueType: ""},
	}

	agents := p.uniqueAgents()
	if len(agents) != 2 {
		t.Errorf("uniqueAgents = %d, want 2", len(agents))
	}
	// Should be sorted
	if agents[0] != "amber" || agents[1] != "ruby" {
		t.Errorf("uniqueAgents = %v, want [amber ruby]", agents)
	}

	types := p.uniqueTypes()
	if len(types) != 2 {
		t.Errorf("uniqueTypes = %d, want 2", len(types))
	}
	if types[0] != "bug" || types[1] != "feature" {
		t.Errorf("uniqueTypes = %v, want [bug feature]", types)
	}
}
