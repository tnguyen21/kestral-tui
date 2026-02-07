package pane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewNewIssuePane(t *testing.T) {
	p := NewNewIssuePane()
	if p.ID() != PaneNewIssue {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneNewIssue)
	}
	if p.Title() != "New Issue" {
		t.Errorf("Title() = %q, want %q", p.Title(), "New Issue")
	}
	if p.ShortTitle() != "\U0001F4DD" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "\U0001F4DD")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0", p.Badge())
	}
}

func TestNewIssuePaneViewEmpty(t *testing.T) {
	p := NewNewIssuePane()
	// Don't set size — should return empty
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestNewIssuePaneViewForm(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	view := p.View()

	if !strings.Contains(view, "NEW ISSUE") {
		t.Error("View should contain 'NEW ISSUE' header")
	}
	if !strings.Contains(view, "Title") {
		t.Error("View should contain 'Title' field label")
	}
	if !strings.Contains(view, "Description") {
		t.Error("View should contain 'Description' field label")
	}
	if !strings.Contains(view, "Type") {
		t.Error("View should contain 'Type' field label")
	}
	if !strings.Contains(view, "Priority") {
		t.Error("View should contain 'Priority' field label")
	}
	if !strings.Contains(view, "Rig") {
		t.Error("View should contain 'Rig' field label")
	}
	if !strings.Contains(view, "ctrl+s") {
		t.Error("View should contain submit help")
	}
}

func TestNewIssuePaneRigListMsg(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	rigs := []string{"gastown", "kestral_tui", "beads"}
	updated, _ := p.Update(RigListMsg{Rigs: rigs})
	p = updated.(*NewIssuePane)

	if len(p.rigs) != 3 {
		t.Errorf("rigs count = %d, want 3", len(p.rigs))
	}

	view := p.View()
	if !strings.Contains(view, "gastown") {
		t.Error("View should contain rig name 'gastown'")
	}
}

func TestNewIssuePaneFieldNavigation(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Start at title field
	if p.activeField != fieldTitle {
		t.Errorf("initial field = %d, want %d (title)", p.activeField, fieldTitle)
	}

	// Tab to description
	p.Update(tea.KeyMsg{Type: tea.KeyTab})
	if p.activeField != fieldDescription {
		t.Errorf("after tab, field = %d, want %d (description)", p.activeField, fieldDescription)
	}

	// Tab to type
	p.Update(tea.KeyMsg{Type: tea.KeyTab})
	if p.activeField != fieldType {
		t.Errorf("after 2nd tab, field = %d, want %d (type)", p.activeField, fieldType)
	}

	// Down to priority
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.activeField != fieldPriority {
		t.Errorf("after down, field = %d, want %d (priority)", p.activeField, fieldPriority)
	}

	// Up back to type
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.activeField != fieldType {
		t.Errorf("after up, field = %d, want %d (type)", p.activeField, fieldType)
	}
}

func TestNewIssuePaneTypeToggle(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Navigate to type field
	p.activeField = fieldType

	// Default is index 0 (bug)
	if p.typeIdx != 0 {
		t.Errorf("initial typeIdx = %d, want 0", p.typeIdx)
	}

	// Right arrow to feature
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if p.typeIdx != 1 {
		t.Errorf("after right, typeIdx = %d, want 1", p.typeIdx)
	}

	// Right arrow to task
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if p.typeIdx != 2 {
		t.Errorf("after 2nd right, typeIdx = %d, want 2", p.typeIdx)
	}

	// Right arrow at end — stays at 2
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if p.typeIdx != 2 {
		t.Errorf("right at end, typeIdx = %d, want 2", p.typeIdx)
	}

	// Left arrow back
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if p.typeIdx != 1 {
		t.Errorf("after left, typeIdx = %d, want 1", p.typeIdx)
	}
}

func TestNewIssuePanePriorityDefault(t *testing.T) {
	p := NewNewIssuePane()

	// Default priority should be P2 Medium (index 2)
	if p.priorityIdx != 2 {
		t.Errorf("default priorityIdx = %d, want 2 (P2 Medium)", p.priorityIdx)
	}
}

func TestNewIssuePaneSubmitEmpty(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Try to submit with empty title — should not change state
	p.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if p.state != stateForm {
		t.Errorf("submit with empty title should stay in form state, got %d", p.state)
	}
}

func TestNewIssuePaneSetSize(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestNewIssuePaneResultView(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Simulate successful submission
	p.Update(IssueSubmitMsg{BeadID: "kt-abc1"})

	view := p.View()
	if !strings.Contains(view, "kt-abc1") {
		t.Error("Result view should contain the created bead ID")
	}
	if !strings.Contains(view, "Created") {
		t.Error("Result view should contain 'Created' message")
	}
	if !strings.Contains(view, "n=create another") {
		t.Error("Result view should show create another option")
	}
}

func TestNewIssuePaneResultError(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Simulate failed submission
	p.Update(IssueSubmitMsg{Err: errTest})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error result view should contain 'Error'")
	}
	if !strings.Contains(view, "n=try again") {
		t.Error("Error result view should show try again option")
	}
}

func TestNewIssuePaneResetFromResult(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Go to result state
	p.Update(IssueSubmitMsg{BeadID: "kt-abc1"})
	if p.state != stateResult {
		t.Fatal("should be in result state")
	}

	// Press 'n' to create another
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if p.state != stateForm {
		t.Errorf("after 'n', state = %d, want %d (form)", p.state, stateForm)
	}
	if p.titleInput.Value() != "" {
		t.Error("title should be cleared after reset")
	}
}

func TestNewIssuePaneDescriptionInput(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)

	// Navigate to description
	p.activeField = fieldDescription

	// Type some text
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")})
	if p.description[0] != "hello" {
		t.Errorf("description = %q, want %q", p.description[0], "hello")
	}

	// Enter to create new line
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(p.description) != 2 {
		t.Errorf("description lines = %d, want 2", len(p.description))
	}
	if p.descLine != 1 {
		t.Errorf("descLine = %d, want 1", p.descLine)
	}

	// Type on second line
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("world")})
	if p.description[1] != "world" {
		t.Errorf("description[1] = %q, want %q", p.description[1], "world")
	}
}

func TestNewIssuePaneDescriptionBackspace(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)
	p.activeField = fieldDescription

	// Type and backspace
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.description[0] != "ab" {
		t.Errorf("after backspace, description = %q, want %q", p.description[0], "ab")
	}

	// Create second line and backspace at start to merge
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(p.description) != 1 {
		t.Errorf("after merge backspace, lines = %d, want 1", len(p.description))
	}
	if p.description[0] != "ab" {
		t.Errorf("after merge, description = %q, want %q", p.description[0], "ab")
	}
}

func TestNewIssuePaneSubmittingView(t *testing.T) {
	p := NewNewIssuePane()
	p.SetSize(80, 24)
	p.state = stateSubmitting

	view := p.View()
	if !strings.Contains(view, "Submitting") {
		t.Error("Submitting view should contain 'Submitting'")
	}
}

func TestParseBeadID(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"standard", "Created: kt-abc1\n", "kt-abc1"},
		{"with prefix", "✓ Created issue: gt-xyz9\nDone.", "gt-xyz9"},
		{"no match", "Error occurred", "Error occurred"},
		{"empty", "", "(unknown)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBeadID(tt.output)
			if got != tt.want {
				t.Errorf("parseBeadID(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

// Ensure NewIssuePane implements Pane at compile time.
var _ Pane = (*NewIssuePane)(nil)
