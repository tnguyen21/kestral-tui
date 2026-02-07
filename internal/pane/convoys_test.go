package pane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
)

func TestNewConvoysPane(t *testing.T) {
	p := NewConvoysPane()
	if p.ID() != PaneConvoys {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneConvoys)
	}
	if p.Title() != "Convoys" {
		t.Errorf("Title() = %q, want %q", p.Title(), "Convoys")
	}
	if p.ShortTitle() != "\U0001F69A" {
		t.Errorf("ShortTitle() = %q, want 🚚", p.ShortTitle())
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
	if p.expanded != -1 {
		t.Errorf("expanded = %d, want -1", p.expanded)
	}
}

func TestConvoysPaneBadge(t *testing.T) {
	p := NewConvoysPane()
	p.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Deploy v2"},
		{ID: "c2", Title: "Bug sweep"},
	}
	if got := p.Badge(); got != 2 {
		t.Errorf("Badge() = %d, want 2", got)
	}
}

func TestConvoysPaneUpdateWithData(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 24)

	msg := ConvoyUpdateMsg{
		Convoys: []data.ConvoyInfo{
			{ID: "c1", Title: "Deploy v2", Status: "feeding"},
			{ID: "c2", Title: "Bug sweep", Status: "in-progress"},
		},
		Progress: map[string][2]int{
			"c1": {3, 5},
			"c2": {1, 3},
		},
		Issues: map[string][]data.IssueDetail{
			"c1": {
				{ID: "gt-1", Title: "Fix auth", Status: "COMPLETED", Assignee: "dag"},
				{ID: "gt-2", Title: "Add tests", Status: "IN_PROGRESS", Assignee: "nux"},
			},
		},
	}

	updated, _ := p.Update(msg)
	p = updated.(*ConvoysPane)

	if len(p.convoys) != 2 {
		t.Fatalf("convoys count = %d, want 2", len(p.convoys))
	}
	if p.progress["c1"] != [2]int{3, 5} {
		t.Errorf("progress c1 = %v, want {3, 5}", p.progress["c1"])
	}
	if len(p.issues["c1"]) != 2 {
		t.Errorf("issues c1 count = %d, want 2", len(p.issues["c1"]))
	}
}

func TestConvoysPaneViewList(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 24)

	p.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Deploy v2", Status: "feeding"},
	}
	p.progress = map[string][2]int{
		"c1": {3, 5},
	}

	view := p.View()

	if !strings.Contains(view, "CONVOYS") {
		t.Error("View should contain 'CONVOYS' header")
	}
	if !strings.Contains(view, "Deploy v2") {
		t.Error("View should contain convoy title")
	}
	if !strings.Contains(view, "3/5") {
		t.Error("View should contain progress fraction")
	}
	if !strings.Contains(view, "j/k scroll") {
		t.Error("View should contain footer")
	}
}

func TestConvoysPaneViewEmpty(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No open convoys") {
		t.Error("Empty pane should show 'No open convoys'")
	}
}

func TestConvoysPaneViewZeroSize(t *testing.T) {
	p := NewConvoysPane()
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestConvoysPaneExpand(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 24)

	p.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Deploy v2", Status: "feeding"},
	}
	p.progress = map[string][2]int{
		"c1": {1, 3},
	}
	p.issues = map[string][]data.IssueDetail{
		"c1": {
			{ID: "gt-1", Title: "Fix auth", Status: "COMPLETED", Assignee: "dag"},
			{ID: "gt-2", Title: "Add tests", Status: "IN_PROGRESS", Assignee: "nux"},
			{ID: "gt-3", Title: "Deploy", Status: "OPEN"},
		},
	}

	// Press Enter to expand
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.expanded != 0 {
		t.Errorf("expanded = %d, want 0 after enter", p.expanded)
	}

	// Detail view should show issues
	view := p.View()
	if !strings.Contains(view, "CONVOY:") {
		t.Error("Detail view should contain 'CONVOY:' header")
	}
	if !strings.Contains(view, "gt-1") {
		t.Error("Detail view should contain issue ID")
	}
	if !strings.Contains(view, "Fix auth") {
		t.Error("Detail view should contain issue title")
	}
	if !strings.Contains(view, "esc back") {
		t.Error("Detail view should contain back footer")
	}
}

func TestConvoysPaneCollapse(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 24)

	p.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Deploy v2"},
		{ID: "c2", Title: "Bug sweep"},
	}
	p.progress = map[string][2]int{"c1": {1, 2}, "c2": {0, 3}}
	p.issues = map[string][]data.IssueDetail{
		"c1": {{ID: "i1", Title: "T1", Status: "OPEN"}},
	}

	// Select second convoy
	p.cursor = 1
	// Expand
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.expanded != 1 {
		t.Fatalf("expanded = %d, want 1", p.expanded)
	}

	// Press Esc to collapse
	p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if p.expanded != -1 {
		t.Errorf("expanded = %d, want -1 after esc", p.expanded)
	}
	if p.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (restored to convoy index)", p.cursor)
	}
}

func TestConvoysPaneScrolling(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 5) // Tiny viewport

	var convoys []data.ConvoyInfo
	for i := 0; i < 20; i++ {
		convoys = append(convoys, data.ConvoyInfo{
			ID:    "c" + string(rune('a'+i)),
			Title: "Convoy " + string(rune('A'+i)),
		})
	}
	p.Update(ConvoyUpdateMsg{
		Convoys:  convoys,
		Progress: map[string][2]int{},
	})

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

func TestConvoysPaneSetSize(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestConvoysPaneNilProgress(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(40, 20)

	msg := ConvoyUpdateMsg{
		Convoys:  []data.ConvoyInfo{{ID: "c1", Title: "Test"}},
		Progress: nil,
	}

	model, _ := p.Update(msg)
	updated := model.(*ConvoysPane)
	if updated.progress == nil {
		t.Error("progress map should be initialized even when nil is passed")
	}
}

func TestConvoysPaneExpandedBeyondRange(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 24)
	p.expanded = 5 // out of range

	p.convoys = []data.ConvoyInfo{{ID: "c1", Title: "Test"}}
	p.progress = map[string][2]int{}

	// Update with data should reset expanded
	p.Update(ConvoyUpdateMsg{
		Convoys:  p.convoys,
		Progress: p.progress,
	})

	if p.expanded != -1 {
		t.Errorf("expanded = %d, want -1 (reset when out of range)", p.expanded)
	}
}

func TestIssueStatusIcon(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"COMPLETED"},
		{"CLOSED"},
		{"IN_PROGRESS"},
		{"BLOCKED"},
		{"OPEN"},
		{"PENDING"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := issueStatusIcon(tt.status)
			if icon == "" {
				t.Errorf("issueStatusIcon(%q) should not be empty", tt.status)
			}
		})
	}
}

func TestConvoyStatusLabel(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"feeding"},
		{"landed"},
		{"in-progress"},
		{"unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			label := convoyStatusLabel(tt.status)
			if label == "" {
				t.Errorf("convoyStatusLabel(%q) should not be empty", tt.status)
			}
		})
	}
}

func TestMultiColorBar(t *testing.T) {
	p := NewConvoysPane()
	p.issues = map[string][]data.IssueDetail{
		"c1": {
			{ID: "i1", Status: "COMPLETED"},
			{ID: "i2", Status: "IN_PROGRESS"},
			{ID: "i3", Status: "BLOCKED"},
			{ID: "i4", Status: "OPEN"},
		},
	}

	bar := p.multiColorBar("c1", 8)
	if bar == "" {
		t.Error("multiColorBar should not be empty")
	}
}

func TestMultiColorBarNoIssues(t *testing.T) {
	p := NewConvoysPane()
	p.progress = map[string][2]int{
		"c1": {3, 5},
	}

	bar := p.multiColorBar("c1", 6)
	if bar == "" {
		t.Error("multiColorBar with no issues should fall back to simple bar")
	}
}

func TestConvoysPaneDetailScroll(t *testing.T) {
	p := NewConvoysPane()
	p.SetSize(80, 10) // Small viewport

	p.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Deploy v2"},
	}
	p.progress = map[string][2]int{"c1": {0, 20}}

	var issues []data.IssueDetail
	for i := 0; i < 20; i++ {
		issues = append(issues, data.IssueDetail{
			ID:     "i" + string(rune('a'+i)),
			Title:  "Issue " + string(rune('A'+i)),
			Status: "OPEN",
		})
	}
	p.issues = map[string][]data.IssueDetail{"c1": issues}

	// Expand
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Scroll down
	for i := 0; i < 5; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if p.cursor != 5 {
		t.Errorf("cursor in detail view = %d, want 5", p.cursor)
	}

	// View should render without panic
	view := p.View()
	if view == "" {
		t.Error("detail view should not be empty")
	}
}
