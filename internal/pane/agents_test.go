package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewAgentsPane(t *testing.T) {
	p := NewAgentsPane()
	if p.ID() != PaneAgents {
		t.Errorf("ID() = %d, want %d", p.ID(), PaneAgents)
	}
	if p.Title() != "Agents" {
		t.Errorf("Title() = %q, want %q", p.Title(), "Agents")
	}
	if p.ShortTitle() != "🤖" {
		t.Errorf("ShortTitle() = %q, want %q", p.ShortTitle(), "🤖")
	}
	if p.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0 for empty pane", p.Badge())
	}
}

func TestAgentsPaneBadge(t *testing.T) {
	p := NewAgentsPane()
	p.agents = []AgentInfo{
		{Name: "dag", Status: "working"},
		{Name: "nux", Status: "idle"},
		{Name: "quartz", Status: "stale"},
	}
	// Badge counts non-idle agents
	if got := p.Badge(); got != 2 {
		t.Errorf("Badge() = %d, want 2 (working + stale)", got)
	}
}

func TestAgentsPaneUpdateWithAgentData(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "witness", Rig: "gastown", Role: "witness", Status: "working", Age: 2 * time.Minute},
		{Name: "refinery", Rig: "gastown", Role: "refinery", Status: "working", Age: 2 * time.Minute},
		{Name: "dag", Rig: "gastown", Role: "polecat", Status: "working", Age: 5 * time.Minute, IssueID: "gt-kr3", IssueTitle: "Fix auth bug"},
		{Name: "nux", Rig: "gastown", Role: "polecat", Status: "idle", Age: 30 * time.Minute},
	}

	updated, _ := p.Update(AgentUpdateMsg{Agents: agents})
	p = updated.(*AgentsPane)

	if len(p.agents) != 4 {
		t.Fatalf("agents count = %d, want 4", len(p.agents))
	}
	if p.Badge() != 3 {
		t.Errorf("Badge() = %d, want 3 (3 non-idle)", p.Badge())
	}
}

func TestAgentsPaneView(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "witness", Rig: "gastown", Role: "witness", Status: "working", Age: 2 * time.Minute},
		{Name: "dag", Rig: "gastown", Role: "polecat", Status: "working", Age: 5 * time.Minute, IssueID: "gt-kr3", IssueTitle: "Fix auth bug"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	view := p.View()

	// Header should show running count
	if !strings.Contains(view, "AGENTS") {
		t.Error("View should contain 'AGENTS' header")
	}

	// Should contain agent names
	if !strings.Contains(view, "witness") {
		t.Error("View should contain agent name 'witness'")
	}
	if !strings.Contains(view, "dag") {
		t.Error("View should contain agent name 'dag'")
	}

	// Should contain issue info
	if !strings.Contains(view, "gt-kr3") {
		t.Error("View should contain issue ID 'gt-kr3'")
	}
	if !strings.Contains(view, "Fix auth bug") {
		t.Error("View should contain issue title")
	}

	// Should contain footer
	if !strings.Contains(view, "j/k to scroll") {
		t.Error("View should contain scroll help footer")
	}
}

func TestAgentsPaneViewEmpty(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No agents running") {
		t.Error("Empty pane should show 'No agents running'")
	}
}

func TestAgentsPaneViewError(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	p.Update(AgentUpdateMsg{Err: errTest})

	view := p.View()
	if !strings.Contains(view, "Error") {
		t.Error("Error state should show error message")
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestAgentsPaneScrolling(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 5) // Tiny viewport: header(1) + content(2) + footer(1) = very constrained

	// Create many agents to require scrolling
	var agents []AgentInfo
	for i := 0; i < 20; i++ {
		agents = append(agents, AgentInfo{
			Name:   "agent" + string(rune('a'+i)),
			Role:   "polecat",
			Status: "working",
			Age:    time.Duration(i) * time.Minute,
		})
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Cursor starts at 0
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

func TestAgentStatusFromAge(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"active", 2 * time.Minute, "working"},
		{"boundary active", 4*time.Minute + 59*time.Second, "working"},
		{"stale", 10 * time.Minute, "stale"},
		{"boundary stale", 29*time.Minute + 59*time.Second, "stale"},
		{"stuck", 30 * time.Minute, "stuck"},
		{"very stuck", 2 * time.Hour, "stuck"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgentStatusFromAge(tt.age)
			if got != tt.want {
				t.Errorf("AgentStatusFromAge(%v) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func TestDetectRole(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"witness", "witness"},
		{"refinery", "refinery"},
		{"mayor", "mayor"},
		{"dag", "polecat"},
		{"quartz", "polecat"},
		{"nux", "polecat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRole(tt.name)
			if got != tt.want {
				t.Errorf("DetectRole(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRoleIcon(t *testing.T) {
	tests := []struct {
		role string
		want string
	}{
		{"witness", "👁"},
		{"refinery", "🔧"},
		{"polecat", "🦨"},
		{"crew", "👷"},
		{"mayor", "👑"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := RoleIcon(tt.role)
			if got != tt.want {
				t.Errorf("RoleIcon(%q) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}

func TestAgentsPaneSetSize(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(120, 40)
	if p.width != 120 || p.height != 40 {
		t.Errorf("SetSize(120, 40) -> width=%d, height=%d", p.width, p.height)
	}
}

func TestAgentsPaneViewZeroSize(t *testing.T) {
	p := NewAgentsPane()
	// Don't set size — should return empty
	view := p.View()
	if view != "" {
		t.Errorf("View with zero size should be empty, got %q", view)
	}
}

func TestPadOrTruncate(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"pad short", "hi", 5, "hi   "},
		{"exact", "hello", 5, "hello"},
		{"truncate", "hello world", 5, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padOrTruncate(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("padOrTruncate(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}
