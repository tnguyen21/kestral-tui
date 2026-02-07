package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tnguyen21/kestral-tui/internal/config"
	"github.com/tnguyen21/kestral-tui/internal/data"
	"github.com/tnguyen21/kestral-tui/internal/pane"
)

func testModel() Model {
	cfg := config.Default()
	return New(cfg)
}

func sized(m Model, w, h int) Model {
	newM, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return newM.(Model)
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	m := testModel()

	if len(m.panes) != 10 {
		t.Fatalf("expected 10 panes, got %d", len(m.panes))
	}
	if m.panes[0].ID() != pane.PaneDashboard {
		t.Errorf("pane 0 should be Dashboard, got %d", m.panes[0].ID())
	}
	if m.panes[1].ID() != pane.PaneAgents {
		t.Errorf("pane 1 should be Agents, got %d", m.panes[1].ID())
	}
	if m.panes[2].ID() != pane.PaneRefinery {
		t.Errorf("pane 2 should be Refinery, got %d", m.panes[2].ID())
	}
	if m.panes[3].ID() != pane.PanePRs {
		t.Errorf("pane 3 should be PRs, got %d", m.panes[3].ID())
	}
	if m.panes[4].ID() != pane.PaneConvoys {
		t.Errorf("pane 4 should be Convoys, got %d", m.panes[4].ID())
	}
	if m.panes[5].ID() != pane.PaneResources {
		t.Errorf("pane 5 should be Resources, got %d", m.panes[5].ID())
	}
	if m.panes[6].ID() != pane.PaneHistory {
		t.Errorf("pane 6 should be History, got %d", m.panes[6].ID())
	}
	if m.panes[7].ID() != pane.PaneNewIssue {
		t.Errorf("pane 7 should be NewIssue, got %d", m.panes[7].ID())
	}
	if m.panes[8].ID() != pane.PaneMail {
		t.Errorf("pane 8 should be Mail, got %d", m.panes[8].ID())
	}
	if m.panes[9].ID() != pane.PaneWitness {
		t.Errorf("pane 9 should be Witness, got %d", m.panes[9].ID())
	}
	if m.activePane != 0 {
		t.Errorf("activePane should start at 0, got %d", m.activePane)
	}
	if m.fetcher == nil {
		t.Error("fetcher should not be nil")
	}
	if m.config == nil {
		t.Error("config should not be nil")
	}
}

func TestInit(t *testing.T) {
	m := testModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a batch command")
	}
}

// ---------------------------------------------------------------------------
// Window resize
// ---------------------------------------------------------------------------

func TestWindowSizeMsg(t *testing.T) {
	m := testModel()
	m = sized(m, 100, 40)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
	if m.layoutMode != LayoutWide {
		t.Errorf("layoutMode = %d, want LayoutWide", m.layoutMode)
	}
}

func TestLayoutModeNarrow(t *testing.T) {
	m := testModel()
	m = sized(m, 30, 20)
	if m.layoutMode != LayoutNarrow {
		t.Errorf("layoutMode = %d, want LayoutNarrow", m.layoutMode)
	}
}

func TestLayoutModeMedium(t *testing.T) {
	m := testModel()
	m = sized(m, 60, 20)
	if m.layoutMode != LayoutMedium {
		t.Errorf("layoutMode = %d, want LayoutMedium", m.layoutMode)
	}
}

// ---------------------------------------------------------------------------
// Tab switching via keys
// ---------------------------------------------------------------------------

func TestTabKey(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	if m.activePane != 0 {
		t.Fatal("should start on pane 0")
	}

	// Tab forward: 0 -> 1
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 1 {
		t.Errorf("after tab: activePane = %d, want 1", m.activePane)
	}

	// Tab forward: 1 -> 2 (Refinery)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 2 {
		t.Errorf("after 2nd tab: activePane = %d, want 2", m.activePane)
	}

	// Tab forward: 2 -> 3 (PRs)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 3 {
		t.Errorf("after 3rd tab: activePane = %d, want 3", m.activePane)
	}

	// Tab forward: 3 -> 4 (Convoys)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 4 {
		t.Errorf("after 4th tab: activePane = %d, want 4", m.activePane)
	}

	// Tab forward: 4 -> 5 (Resources)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 5 {
		t.Errorf("after 5th tab: activePane = %d, want 5", m.activePane)
	}

	// Tab forward: 5 -> 6 (History)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 6 {
		t.Errorf("after 6th tab: activePane = %d, want 6", m.activePane)
	}

	// Tab forward: 6 -> 7 (NewIssue)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 7 {
		t.Errorf("after 7th tab: activePane = %d, want 7", m.activePane)
	}

	// From pane 7 (NewIssue/input pane), tab is captured by the pane.
	// Verify wrap from a non-input pane.
	m.activePane = 6
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activePane != 7 {
		t.Errorf("tab from 6: activePane = %d, want 7", m.activePane)
	}
}

func TestShiftTabKey(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Shift+tab wraps backward: 0 -> 9 (last pane)
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newM.(Model)
	if m.activePane != 9 {
		t.Errorf("shift+tab from 0: activePane = %d, want 9", m.activePane)
	}
}

func TestNumberKeys(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Press "2" to go to agents pane
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = newM.(Model)
	if m.activePane != 1 {
		t.Errorf("after '2': activePane = %d, want 1", m.activePane)
	}

	// Press "1" to go back to dashboard
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = newM.(Model)
	if m.activePane != 0 {
		t.Errorf("after '1': activePane = %d, want 0", m.activePane)
	}

	// Press "3" to go to refinery pane
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = newM.(Model)
	if m.activePane != 2 {
		t.Errorf("after '3': activePane = %d, want 2", m.activePane)
	}

	// Press "4" to go to PRs pane
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m = newM.(Model)
	if m.activePane != 3 {
		t.Errorf("after '4': activePane = %d, want 3", m.activePane)
	}

	// Press "5" to go to convoys pane
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = newM.(Model)
	if m.activePane != 4 {
		t.Errorf("after '5': activePane = %d, want 4", m.activePane)
	}

	// Press "6" to go to resources pane
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
	m = newM.(Model)
	if m.activePane != 5 {
		t.Errorf("after '6': activePane = %d, want 5", m.activePane)
	}

	// Press "7" to go to history pane
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}})
	m = newM.(Model)
	if m.activePane != 6 {
		t.Errorf("after '7': activePane = %d, want 6", m.activePane)
	}
}

func TestQuitKey(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("quit key should return a command")
	}
}

func TestHelpToggle(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	if m.showHelp {
		t.Fatal("help should be off initially")
	}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newM.(Model)
	if !m.showHelp {
		t.Error("help should be on after pressing ?")
	}

	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newM.(Model)
	if m.showHelp {
		t.Error("help should be off after pressing ? again")
	}
}

func TestRefreshKey(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("refresh should return a command")
	}
}

// ---------------------------------------------------------------------------
// Mouse clicks on tab bar
// ---------------------------------------------------------------------------

func TestMouseTabClick(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Verify initial pane
	if m.activePane != 0 {
		t.Fatal("should start on pane 0")
	}

	// Find the x position that corresponds to pane 1 (Agents tab)
	// In wide mode: " Dashboard " | " Agents "
	// TabActiveStyle has Padding(0,1) + underline for active
	// TabInactiveStyle has Padding(0,1)
	// We need to find an x within the "Agents" tab
	// The separator " | " is between tabs
	// First tab: " Dashboard " (Dashboard is 9 chars + 2 padding = 11)
	// Separator: " | " (3 chars)
	// Second tab starts at 11 + 3 = 14
	// Use tabAtX to find the correct position
	agentIdx := m.tabAtX(14)
	if agentIdx != 1 {
		// Fall back to scanning for a valid position
		for x := 0; x < m.width; x++ {
			if m.tabAtX(x) == 1 {
				agentIdx = 1
				// Click at this position
				newM, _ := m.Update(tea.MouseMsg{
					X:      x,
					Y:      0,
					Button: tea.MouseButtonLeft,
					Action: tea.MouseActionPress,
				})
				m = newM.(Model)
				break
			}
		}
	} else {
		newM, _ := m.Update(tea.MouseMsg{
			X:      14,
			Y:      0,
			Button: tea.MouseButtonLeft,
			Action: tea.MouseActionPress,
		})
		m = newM.(Model)
	}

	if m.activePane != 1 {
		t.Errorf("after clicking agents tab: activePane = %d, want 1", m.activePane)
	}
}

func TestMouseNonTabRow(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	newM, _ := m.Update(tea.MouseMsg{
		X:      5,
		Y:      5, // Not the tab bar row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	m = newM.(Model)

	if m.activePane != 0 {
		t.Errorf("click on non-tab row should not change pane: activePane = %d, want 0", m.activePane)
	}
}

// ---------------------------------------------------------------------------
// Data message forwarding
// ---------------------------------------------------------------------------

func TestStatusUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	now := time.Now()
	msg := pane.StatusUpdateMsg{
		Status: &data.TownStatus{
			Agents: []data.AgentInfo{
				{Name: "test", Running: true},
			},
		},
		Sessions:  []data.SessionInfo{{Name: "gt-rig-test", Activity: now.Unix()}},
		FetchedAt: now,
	}

	newM, cmd := m.Update(msg)
	m = newM.(Model)

	if m.lastRefresh.IsZero() {
		t.Error("lastRefresh should be updated")
	}
	if cmd == nil {
		t.Error("should schedule next poll")
	}
}

func TestAgentUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	msg := pane.AgentUpdateMsg{
		Agents: []pane.AgentInfo{
			{Name: "obsidian", Role: "polecat", Status: "working"},
		},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next agent poll")
	}
}

func TestRefineryUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	msg := pane.RefineryUpdateMsg{
		Statuses: []data.RefineryStatus{
			{Rig: "testrig", Running: true, QueueDepth: 2},
		},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next refinery poll")
	}
}

func TestPRUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	msg := pane.PRUpdateMsg{
		PRs: []data.PRInfo{
			{Number: 42, Title: "Fix bug", Author: data.PRAuthor{Login: "alice"}},
		},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next PR poll")
	}
}

func TestHistoryUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	msg := pane.HistoryUpdateMsg{
		ClosedBeads: []data.ClosedBeadInfo{
			{ID: "kt-abc", Title: "Fix bug", Status: "closed", ClosedAt: "2026-02-07T01:00:00Z"},
		},
		Convoys: []data.AllConvoyInfo{
			{ID: "cv-1", Title: "Convoy 1", Status: "closed"},
		},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next history poll")
	}
}

func TestConvoyUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	msg := pane.ConvoyUpdateMsg{
		Convoys:  []data.ConvoyInfo{{ID: "c1", Title: "Test"}},
		Progress: map[string][2]int{"c1": {3, 5}},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next convoy poll")
	}
}

// ---------------------------------------------------------------------------
// Tick messages
// ---------------------------------------------------------------------------

func TestStatusTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.StatusTickMsg(time.Now()))
	if cmd == nil {
		t.Error("StatusTickMsg should trigger a fetch command")
	}
}

func TestAgentTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.AgentTickMsg(time.Now()))
	if cmd == nil {
		t.Error("AgentTickMsg should trigger a fetch command")
	}
}

func TestRefineryTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.RefineryTickMsg(time.Now()))
	if cmd == nil {
		t.Error("RefineryTickMsg should trigger a fetch command")
	}
}

func TestHistoryTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.HistoryTickMsg(time.Now()))
	if cmd == nil {
		t.Error("HistoryTickMsg should trigger a fetch command")
	}
}

func TestConvoyTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.ConvoyTickMsg(time.Now()))
	if cmd == nil {
		t.Error("ConvoyTickMsg should trigger a fetch command")
	}
}

func TestResourceTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.ResourceTickMsg(time.Now()))
	if cmd == nil {
		t.Error("ResourceTickMsg should trigger a fetch command")
	}
}

func TestResourceUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 120, 24)

	msg := pane.ResourceUpdateMsg{
		Sessions: []data.SessionResource{
			{Name: "gt-kestral-witness", CPUPercent: 12.5, MemRSS: 50 << 20},
		},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next resource poll")
	}
}

func TestWitnessTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.WitnessTickMsg(time.Now()))
	if cmd == nil {
		t.Error("WitnessTickMsg should trigger a fetch command")
	}
}

func TestWitnessUpdateForwarding(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	msg := pane.WitnessUpdateMsg{
		Witnesses: []pane.WitnessInfo{
			{Rig: "gastown", Status: "alive", HasSession: true},
		},
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("should schedule next witness poll")
	}
}

func TestPRTickTriggersCmd(t *testing.T) {
	m := testModel()
	_, cmd := m.Update(data.PRTickMsg(time.Now()))
	if cmd == nil {
		t.Error("PRTickMsg should trigger a fetch command")
	}
}

// ---------------------------------------------------------------------------
// View rendering
// ---------------------------------------------------------------------------

func TestViewEmptySize(t *testing.T) {
	m := testModel()
	if v := m.View(); v != "" {
		t.Errorf("View with zero size should be empty, got %q", v)
	}
}

func TestViewRendersTabBar(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	v := m.View()
	if v == "" {
		t.Fatal("View should not be empty after resize")
	}
	if !containsText(v, "Dashboard") {
		t.Error("View should contain Dashboard tab")
	}
	if !containsText(v, "Agents") {
		t.Error("View should contain Agents tab")
	}
	if !containsText(v, "Refinery") {
		t.Error("View should contain Refinery tab")
	}
	if !containsText(v, "PRs") {
		t.Error("View should contain PRs tab")
	}
	if !containsText(v, "History") {
		t.Error("View should contain History tab")
	}
}

func TestViewRendersStatusBar(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	v := m.View()
	if !containsText(v, "q=quit") {
		t.Error("View should contain status bar with q=quit")
	}
	if !containsText(v, "?=help") {
		t.Error("View should contain status bar with ?=help")
	}
}

func TestViewShowsHelpWhenToggled(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Toggle help on
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newM.(Model)

	v := m.View()
	// Help view should include keybinding descriptions
	if !containsText(v, "quit") {
		t.Error("help view should contain 'quit' keybinding")
	}
}

// ---------------------------------------------------------------------------
// Tab bar rendering per layout mode
// ---------------------------------------------------------------------------

func TestTabBarNarrowMode(t *testing.T) {
	m := testModel()
	m = sized(m, 30, 20)

	tabBar := m.renderTabBar()
	// In narrow mode, tabs should use short titles (emojis)
	if containsText(tabBar, "Dashboard") {
		t.Error("narrow mode should not show full title 'Dashboard'")
	}
}

func TestTabBarWideMode(t *testing.T) {
	m := testModel()
	m = sized(m, 100, 20)

	tabBar := m.renderTabBar()
	if !containsText(tabBar, "Dashboard") {
		t.Error("wide mode should show full title 'Dashboard'")
	}
	if !containsText(tabBar, "|") {
		t.Error("wide mode should have pipe separators")
	}
}

func TestTabBarActivePaneHighlighted(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// activePane 0 = Dashboard should be rendered with accent style
	// We can verify by checking the rendered output contains the expected tab
	tabBar := m.renderTabBar()
	if tabBar == "" {
		t.Error("tab bar should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Status bar
// ---------------------------------------------------------------------------

func TestStatusBarShowsRefreshTime(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)
	m.lastRefresh = time.Now()

	bar := m.renderStatusBar()
	if !containsText(bar, "just now") {
		t.Error("status bar should show 'just now' for recent refresh")
	}
}

func TestStatusBarShowsEllipsisBeforeFirstRefresh(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	bar := m.renderStatusBar()
	if !containsText(bar, "…") {
		t.Error("status bar should show '…' before first refresh")
	}
}

// ---------------------------------------------------------------------------
// Tab label badge rendering
// ---------------------------------------------------------------------------

func TestTabLabelWithBadge(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Send agent data with active agents to trigger a badge
	msg := pane.AgentUpdateMsg{
		Agents: []pane.AgentInfo{
			{Name: "a1", Status: "working"},
			{Name: "a2", Status: "working"},
		},
	}
	newM, _ := m.Update(msg)
	m = newM.(Model)

	label := m.tabLabel(m.panes[1]) // Agents pane
	if label != "Agents(2)" {
		t.Errorf("tab label = %q, want %q", label, "Agents(2)")
	}
}

// ---------------------------------------------------------------------------
// KeyMap help interface
// ---------------------------------------------------------------------------

func TestKeyMapShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	bindings := km.ShortHelp()
	if len(bindings) == 0 {
		t.Error("ShortHelp should return bindings")
	}
}

func TestKeyMapFullHelp(t *testing.T) {
	km := DefaultKeyMap()
	groups := km.FullHelp()
	if len(groups) == 0 {
		t.Error("FullHelp should return binding groups")
	}
}

// ---------------------------------------------------------------------------
// Key forwarding to active pane
// ---------------------------------------------------------------------------

func TestKeyForwardingToActivePane(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Switch to agents pane
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = newM.(Model)

	// Send j key (scroll down in agents pane) — should not error
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newM.(Model)

	if m.activePane != 1 {
		t.Error("should still be on agents pane after j key")
	}
}

// ---------------------------------------------------------------------------
// tabAtX
// ---------------------------------------------------------------------------

func TestTabAtXFirstTab(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// x=0 should be in the first tab
	idx := m.tabAtX(0)
	if idx != 0 {
		t.Errorf("tabAtX(0) = %d, want 0", idx)
	}
}

func TestTabAtXOutOfBounds(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	idx := m.tabAtX(200)
	if idx != -1 {
		t.Errorf("tabAtX(200) = %d, want -1", idx)
	}
}

func TestTabAtXSecondTab(t *testing.T) {
	m := testModel()
	m = sized(m, 80, 24)

	// Find the second tab's x range
	found := false
	for x := 0; x < 40; x++ {
		if m.tabAtX(x) == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("should be able to find second tab within x=0..39")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// containsText strips ANSI sequences and checks for substring presence.
func containsText(s, sub string) bool {
	stripped := stripANSI(s)
	return contains(stripped, sub)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var b []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find a letter
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++ // skip the final letter
			}
			i = j
		} else {
			b = append(b, s[i])
			i++
		}
	}
	return string(b)
}

// Verify Model satisfies tea.Model at compile time.
var _ tea.Model = Model{}

// Verify lipgloss import is used (for width measurement in tabAtX).
var _ = lipgloss.Width
