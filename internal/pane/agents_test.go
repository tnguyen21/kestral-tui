package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/data"
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

// ---------------------------------------------------------------------------
// Detail view tests
// ---------------------------------------------------------------------------

func TestAgentsPaneEnterDetailMode(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working", Age: 2 * time.Minute, IssueID: "kt-jzx0", IssueTitle: "Implement Agent Detail View"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Press enter to select
	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	if !p.detailMode {
		t.Error("should be in detail mode after pressing enter")
	}
	if p.selectedAgent.Name != "obsidian" {
		t.Errorf("selectedAgent.Name = %q, want %q", p.selectedAgent.Name, "obsidian")
	}
	if cmd == nil {
		t.Error("entering detail mode should return a command")
	}

	// Execute the command and check it produces AgentSelectedMsg
	msg := cmd()
	if _, ok := msg.(AgentSelectedMsg); !ok {
		t.Errorf("command should produce AgentSelectedMsg, got %T", msg)
	}
}

func TestAgentsPaneExitDetailMode(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	if !p.detailMode {
		t.Fatal("should be in detail mode")
	}

	// Press Esc to exit
	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	p = updated.(*AgentsPane)

	if p.detailMode {
		t.Error("should not be in detail mode after esc")
	}
	if cmd == nil {
		t.Error("exiting detail mode should return a command")
	}

	// Execute the command and check it produces AgentDeselectedMsg
	msg := cmd()
	if _, ok := msg.(AgentDeselectedMsg); !ok {
		t.Errorf("command should produce AgentDeselectedMsg, got %T", msg)
	}
}

func TestAgentsPaneDetailViewRender(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working", Age: 2 * time.Minute, IssueID: "kt-jzx0", IssueTitle: "Implement Agent Detail View"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	view := p.View()

	// Should show agent name in header
	if !strings.Contains(view, "obsidian") {
		t.Error("detail view should contain agent name")
	}

	// Should show loading state before data arrives
	if !strings.Contains(view, "Loading") {
		t.Error("detail view should show Loading before data arrives")
	}

	// Should show footer
	if !strings.Contains(view, "esc to go back") {
		t.Error("detail view should contain esc footer")
	}
}

func TestAgentsPaneDetailDataUpdate(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working", Age: 2 * time.Minute},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	// Send detail data
	detailMsg := AgentDetailDataMsg{
		Name:   "obsidian",
		Branch: "polecat/obsidian/kt-jzx0",
		Commits: []data.CommitInfo{
			{Hash: "d194ca7", Message: "docs: add feature brainstorm"},
			{Hash: "8bf4de1", Message: "docs: add README"},
		},
		Output: "Running tests...\nAll tests passed.",
	}
	updated, _ = p.Update(detailMsg)
	p = updated.(*AgentsPane)

	if p.detailData == nil {
		t.Fatal("detailData should be set after receiving AgentDetailDataMsg")
	}
	if p.detailData.Branch != "polecat/obsidian/kt-jzx0" {
		t.Errorf("branch = %q, want %q", p.detailData.Branch, "polecat/obsidian/kt-jzx0")
	}
	if len(p.detailData.Commits) != 2 {
		t.Errorf("commits count = %d, want 2", len(p.detailData.Commits))
	}

	view := p.View()

	// Should show branch info
	if !strings.Contains(view, "polecat/obsidian/kt-jzx0") {
		t.Error("detail view should show branch name")
	}

	// Should show commits
	if !strings.Contains(view, "d194ca7") {
		t.Error("detail view should show commit hash")
	}
	if !strings.Contains(view, "docs: add feature brainstorm") {
		t.Error("detail view should show commit message")
	}

	// Should show session output
	if !strings.Contains(view, "Running tests") {
		t.Error("detail view should show session output")
	}

	// Should no longer show Loading
	if strings.Contains(view, "Loading") {
		t.Error("detail view should not show Loading after data arrives")
	}
}

func TestAgentsPaneDetailDataIgnoredForWrongAgent(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	// Send detail data for a DIFFERENT agent
	detailMsg := AgentDetailDataMsg{
		Name:   "quartz",
		Branch: "some-branch",
	}
	updated, _ = p.Update(detailMsg)
	p = updated.(*AgentsPane)

	if p.detailData != nil {
		t.Error("detailData should remain nil when data is for a different agent")
	}
}

func TestAgentsPaneDetailViewWithEmptyData(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "witness", Rig: "gastown", Role: "witness", Status: "working"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	// Send empty detail data (no worktree, no tmux output)
	detailMsg := AgentDetailDataMsg{
		Name:    "witness",
		Branch:  "",
		Commits: nil,
		Output:  "",
	}
	updated, _ = p.Update(detailMsg)
	p = updated.(*AgentsPane)

	view := p.View()

	// Should show graceful empty states
	if !strings.Contains(view, "(unavailable)") {
		t.Error("detail view should show (unavailable) for empty branch")
	}
	if !strings.Contains(view, "(no commits)") {
		t.Error("detail view should show (no commits) for empty commits")
	}
	if !strings.Contains(view, "(no output)") {
		t.Error("detail view should show (no output) for empty output")
	}
}

func TestAgentsPaneEnterOnEmptyList(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	// No agents loaded - press enter
	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	if p.detailMode {
		t.Error("should not enter detail mode with no agents")
	}
	if cmd != nil {
		t.Error("should not return a command with no agents")
	}
}

func TestAgentsPaneDetailModeSelectedAgentUpdate(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working", Age: 2 * time.Minute, IssueID: "kt-a", IssueTitle: "Old task"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	// Set some detail data so we can verify the refresh
	p.detailData = &detailViewData{Branch: "test"}

	// Send updated agent data
	updatedAgents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "stale", Age: 10 * time.Minute, IssueID: "kt-b", IssueTitle: "New task"},
	}
	updated, _ = p.Update(AgentUpdateMsg{Agents: updatedAgents})
	p = updated.(*AgentsPane)

	// Selected agent should be updated
	if p.selectedAgent.Status != "stale" {
		t.Errorf("selectedAgent.Status = %q, want %q", p.selectedAgent.Status, "stale")
	}
	if p.selectedAgent.IssueID != "kt-b" {
		t.Errorf("selectedAgent.IssueID = %q, want %q", p.selectedAgent.IssueID, "kt-b")
	}
}

func TestAgentsPaneDetailViewNoIssue(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "witness", Rig: "gastown", Role: "witness", Status: "working"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	view := p.View()
	if !strings.Contains(view, "(none)") {
		t.Error("detail view should show (none) for agent with no hooked issue")
	}
}

func TestAgentsPaneDetailViewSizeUpdate(t *testing.T) {
	p := NewAgentsPane()
	p.SetSize(80, 24)

	agents := []AgentInfo{
		{Name: "obsidian", Rig: "kestral_tui", Role: "polecat", Status: "working"},
	}
	p.Update(AgentUpdateMsg{Agents: agents})

	// Enter detail mode
	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = updated.(*AgentsPane)

	// Resize
	p.SetSize(120, 40)
	if p.detailVP.Width != 120 {
		t.Errorf("detailVP.Width = %d, want 120", p.detailVP.Width)
	}
	if p.detailVP.Height != 39 { // 40 - 1 for footer
		t.Errorf("detailVP.Height = %d, want 39", p.detailVP.Height)
	}
}
