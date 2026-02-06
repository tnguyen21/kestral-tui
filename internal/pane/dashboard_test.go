package pane

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tnguyen21/kestral-tui/internal/data"
)

func TestDashboardInterface(t *testing.T) {
	d := NewDashboard()

	// Verify Pane interface compliance
	var _ Pane = d

	if d.ID() != PaneDashboard {
		t.Errorf("ID() = %d, want PaneDashboard (%d)", d.ID(), PaneDashboard)
	}
	if d.Title() != "Dashboard" {
		t.Errorf("Title() = %q, want %q", d.Title(), "Dashboard")
	}
	if d.ShortTitle() != "\U0001F3E0" {
		t.Errorf("ShortTitle() = %q, want 🏠", d.ShortTitle())
	}
	if d.Badge() != 0 {
		t.Errorf("Badge() = %d, want 0", d.Badge())
	}
}

func TestDashboardInit(t *testing.T) {
	d := NewDashboard()
	cmd := d.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestDashboardSetSize(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	if d.width != 40 {
		t.Errorf("width = %d, want 40", d.width)
	}
	if d.height != 20 {
		t.Errorf("height = %d, want 20", d.height)
	}
}

func TestDashboardStatusUpdate(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	status := &data.TownStatus{
		Agents: []data.AgentInfo{
			{Name: "witness", Running: true, State: "active"},
			{Name: "refinery", Running: true, State: "active"},
			{Name: "dag", Running: true, State: "working"},
			{Name: "nux", Running: false, State: "idle"},
		},
	}
	sessions := []data.SessionInfo{
		{Name: "main", Activity: time.Now().Unix()},
		{Name: "work", Activity: time.Now().Unix()},
	}

	msg := StatusUpdateMsg{
		Status:    status,
		Sessions:  sessions,
		FetchedAt: time.Now(),
	}

	model, cmd := d.Update(msg)
	if cmd != nil {
		t.Error("StatusUpdateMsg should not return a cmd")
	}

	updated := model.(*Dashboard)
	if updated.status != status {
		t.Error("status not updated")
	}
	if len(updated.sessions) != 2 {
		t.Errorf("sessions count = %d, want 2", len(updated.sessions))
	}
}

func TestDashboardConvoyUpdate(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	msg := ConvoyUpdateMsg{
		Convoys: []data.ConvoyInfo{
			{ID: "c1", Title: "Deploy v2", Status: "open"},
			{ID: "c2", Title: "Bug sweep", Status: "open"},
		},
		Progress: map[string][2]int{
			"c1": {3, 5},
			"c2": {1, 3},
		},
	}

	model, cmd := d.Update(msg)
	if cmd != nil {
		t.Error("ConvoyUpdateMsg should not return a cmd")
	}

	updated := model.(*Dashboard)
	if len(updated.convoys) != 2 {
		t.Errorf("convoys count = %d, want 2", len(updated.convoys))
	}
	if updated.progress["c1"] != [2]int{3, 5} {
		t.Errorf("progress c1 = %v, want {3, 5}", updated.progress["c1"])
	}
}

func TestDashboardRenderContent(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	// Render with no data — should show loading state
	content := d.renderContent()
	if !strings.Contains(content, "KESTRAL") {
		t.Error("content missing KESTRAL header")
	}

	// Update with status data
	d.status = &data.TownStatus{
		Agents: []data.AgentInfo{
			{Name: "witness", Running: true, State: "active"},
			{Name: "refinery", Running: true, State: "active"},
		},
	}
	d.sessions = []data.SessionInfo{{Name: "s1"}}
	d.lastUpdate = time.Now()

	content = d.renderContent()
	if !strings.Contains(content, "AGENTS") {
		t.Error("content missing AGENTS section")
	}
	if !strings.Contains(content, "2 running") {
		t.Error("content missing agent count")
	}
	if !strings.Contains(content, "witness") {
		t.Error("content missing agent name")
	}
	if !strings.Contains(content, "SESSIONS") {
		t.Error("content missing SESSIONS section")
	}
}

func TestDashboardRenderConvoyProgress(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	d.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Deploy v2", Status: "open"},
	}
	d.progress = map[string][2]int{
		"c1": {3, 5},
	}

	content := d.renderContent()
	if !strings.Contains(content, "CONVOYS") {
		t.Error("content missing CONVOYS section")
	}
	if !strings.Contains(content, "Deploy v2") {
		t.Error("content missing convoy title")
	}
	if !strings.Contains(content, "3/5") {
		t.Error("content missing progress fraction")
	}
}

func TestDashboardNarrowWidth(t *testing.T) {
	d := NewDashboard()
	d.SetSize(30, 15)

	d.status = &data.TownStatus{
		Agents: []data.AgentInfo{
			{Name: "witness", Running: true, State: "active"},
		},
	}
	d.lastUpdate = time.Now()

	content := d.renderContent()
	// Should still render without panicking at narrow width
	if content == "" {
		t.Error("content should not be empty at narrow width")
	}
}

func TestDashboardDegradedHealth(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	d.status = &data.TownStatus{
		Agents: []data.AgentInfo{
			{Name: "witness", Running: true, State: "active"},
			{Name: "refinery", Running: false, State: "stopped"},
		},
	}
	d.lastUpdate = time.Now()

	content := d.renderContent()
	if !strings.Contains(content, "DEGRADED") {
		t.Error("content should show DEGRADED when agents are stopped")
	}
}

func TestDashboardZeroWidth(t *testing.T) {
	d := NewDashboard()
	d.SetSize(0, 0)
	content := d.renderContent()
	if content != "" {
		t.Error("content should be empty at zero width")
	}
}

func TestDashboardViewportScrolling(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 5) // Very small height to force scrolling

	d.status = &data.TownStatus{
		Agents: []data.AgentInfo{
			{Name: "a1", Running: true, State: "active"},
			{Name: "a2", Running: true, State: "active"},
			{Name: "a3", Running: true, State: "active"},
			{Name: "a4", Running: true, State: "active"},
			{Name: "a5", Running: true, State: "active"},
		},
	}
	d.convoys = []data.ConvoyInfo{
		{ID: "c1", Title: "Convoy 1", Status: "open"},
		{ID: "c2", Title: "Convoy 2", Status: "open"},
	}
	d.progress = map[string][2]int{
		"c1": {1, 3},
		"c2": {2, 4},
	}
	d.sessions = []data.SessionInfo{{Name: "s1"}, {Name: "s2"}}
	d.lastUpdate = time.Now()
	d.viewport.SetContent(d.renderContent())

	// Viewport should render without panic
	view := d.View()
	if view == "" {
		t.Error("View() should not be empty")
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		done, total, width int
		wantFilled         int
	}{
		{0, 5, 6, 0},
		{3, 5, 6, 3},
		{5, 5, 6, 6},
		{0, 0, 6, 0}, // zero total => all empty
	}

	for _, tt := range tests {
		bar := progressBar(tt.done, tt.total, tt.width)
		if bar == "" {
			t.Errorf("progressBar(%d, %d, %d) should not be empty", tt.done, tt.total, tt.width)
		}
	}
}

func TestCenterPad(t *testing.T) {
	result := centerPad("TEST", 20)
	if !strings.Contains(result, "TEST") {
		t.Error("centerPad should contain the text")
	}
	// Should have padding characters
	if len(result) < 20 {
		t.Error("centerPad should pad to width")
	}
}

func TestAgentIcon(t *testing.T) {
	running := data.AgentInfo{Name: "a", Running: true, State: "active"}
	idle := data.AgentInfo{Name: "b", Running: true, State: "idle"}
	stopped := data.AgentInfo{Name: "c", Running: false, State: "stopped"}

	if agentIcon(running) == "" {
		t.Error("running agent icon should not be empty")
	}
	if agentIcon(idle) == "" {
		t.Error("idle agent icon should not be empty")
	}
	if agentIcon(stopped) == "" {
		t.Error("stopped agent icon should not be empty")
	}
}

func TestDashboardConvoyUpdateNilProgress(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)

	msg := ConvoyUpdateMsg{
		Convoys:  []data.ConvoyInfo{{ID: "c1", Title: "Test"}},
		Progress: nil,
	}

	model, _ := d.Update(msg)
	updated := model.(*Dashboard)
	if updated.progress == nil {
		t.Error("progress map should be initialized even when nil is passed")
	}
}

func TestDashboardTeaModel(t *testing.T) {
	d := NewDashboard()
	// Verify it satisfies tea.Model
	var _ tea.Model = d
}

func TestDashboardEmptyConvoys(t *testing.T) {
	d := NewDashboard()
	d.SetSize(40, 20)
	d.convoys = []data.ConvoyInfo{}
	d.lastUpdate = time.Now()

	content := d.renderContent()
	if !strings.Contains(content, "0 open") {
		t.Error("should show 0 open convoys")
	}
	if !strings.Contains(content, "(none)") {
		t.Error("should show (none) for empty convoy list")
	}
}
