package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Command timeout constants matching gastown conventions.
const (
	cmdTimeout     = 15 * time.Second // timeout for most commands (bd can be slow with large datasets)
	ghCmdTimeout   = 10 * time.Second // longer timeout for GitHub API calls
	tmuxCmdTimeout = 2 * time.Second  // short timeout for tmux queries
)

// Fetcher shells out to gt/bd/gh/tmux CLIs to fetch data.
type Fetcher struct {
	TownRoot string // path to gt workspace
}

// runCmd executes a command with a timeout and returns stdout.
func runCmd(timeout time.Duration, name string, args ...string) (*bytes.Buffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s timed out after %v", name, timeout)
		}
		return nil, err
	}
	return &stdout, nil
}

// runBdCmd executes a bd command with cmdTimeout in TownRoot.
func (f *Fetcher) runBdCmd(args ...string) (*bytes.Buffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bd", args...)
	cmd.Dir = f.TownRoot
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd timed out after %v", cmdTimeout)
		}
		// If we got some output, return it anyway (bd may exit non-zero with warnings)
		if stdout.Len() > 0 {
			return &stdout, nil
		}
		return nil, err
	}
	return &stdout, nil
}

// FetchStatus runs gt status --json and parses agent info.
func (f *Fetcher) FetchStatus() (*TownStatus, error) {
	stdout, err := runCmd(cmdTimeout, "gt", "status", "--json")
	if err != nil {
		return nil, fmt.Errorf("running gt status: %w", err)
	}

	var status TownStatus
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		return nil, fmt.Errorf("parsing gt status output: %w", err)
	}
	return &status, nil
}

// FetchSessions runs tmux list-sessions and parses session info.
func (f *Fetcher) FetchSessions() ([]SessionInfo, error) {
	stdout, err := runCmd(tmuxCmdTimeout, "tmux", "list-sessions", "-F", "#{session_name}|#{window_activity}")
	if err != nil {
		return nil, fmt.Errorf("listing tmux sessions: %w", err)
	}

	var sessions []SessionInfo
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}

		s := SessionInfo{Name: parts[0]}
		var ts int64
		if _, err := fmt.Sscanf(parts[1], "%d", &ts); err == nil {
			s.Activity = ts
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// FetchAgents parses tmux gt-* sessions into agent info, enriched with
// hooked issue data from bd. This mirrors gastown's FetchWorkers pattern.
func (f *Fetcher) FetchAgents() ([]AgentDetail, error) {
	sessions, err := f.FetchSessions()
	if err != nil {
		return nil, err
	}

	// Pre-fetch assigned issues: assignee path -> (id, title)
	assigned := f.fetchAssignedIssues()

	now := time.Now()
	var agents []AgentDetail
	for _, s := range sessions {
		if !strings.HasPrefix(s.Name, "gt-") {
			continue
		}

		parts := strings.SplitN(s.Name, "-", 3)
		if len(parts) != 3 {
			continue
		}
		rig := parts[1]
		name := parts[2]

		// Determine role
		role := "polecat"
		switch name {
		case "witness":
			role = "witness"
		case "refinery":
			role = "refinery"
		case "mayor":
			role = "mayor"
		}

		var age time.Duration
		if s.Activity > 0 {
			age = now.Sub(time.Unix(s.Activity, 0))
		}

		// Status based on activity age
		status := "idle"
		if s.Activity > 0 {
			switch {
			case age < 5*time.Minute:
				status = "working"
			case age < 30*time.Minute:
				status = "stale"
			default:
				status = "stuck"
			}
		}

		ad := AgentDetail{
			Name:    name,
			Rig:     rig,
			Role:    role,
			Status:  status,
			AgeSecs: int64(age.Seconds()),
		}

		// Look up hooked issue for this agent
		assignee := fmt.Sprintf("%s/polecats/%s", rig, name)
		if issue, ok := assigned[assignee]; ok {
			ad.IssueID = issue.ID
			ad.IssueTitle = issue.Title
		}

		agents = append(agents, ad)
	}
	return agents, nil
}

// fetchAssignedIssues returns a map of assignee -> issue for all in_progress issues.
func (f *Fetcher) fetchAssignedIssues() map[string]IssueDetail {
	result := make(map[string]IssueDetail)

	stdout, err := f.runBdCmd("list", "--status=in_progress", "--json")
	if err != nil {
		return result
	}

	var issues []IssueDetail
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return result
	}

	for _, issue := range issues {
		if issue.Assignee != "" {
			result[issue.Assignee] = issue
		}
	}
	return result
}

// FetchRefineryStatus queries merge queue state from bd and tmux.
// It lists MR-type beads in various states to build the queue, history,
// and metrics for each active refinery.
func (f *Fetcher) FetchRefineryStatus() ([]RefineryStatus, error) {
	// Discover active refineries from tmux sessions
	sessions, err := f.FetchSessions()
	if err != nil {
		return nil, fmt.Errorf("listing sessions for refineries: %w", err)
	}

	rigSet := make(map[string]bool)
	rigRunning := make(map[string]bool)
	for _, s := range sessions {
		if !strings.HasPrefix(s.Name, "gt-") {
			continue
		}
		parts := strings.SplitN(s.Name, "-", 3)
		if len(parts) == 3 && parts[2] == "refinery" {
			rigSet[parts[1]] = true
			rigRunning[parts[1]] = s.Activity > 0
		}
	}

	// Also check for MR beads even if no refinery session is active
	queuedMRs := f.fetchMRsByStatus("OPEN")
	inProgressMRs := f.fetchMRsByStatus("IN_PROGRESS")
	completedMRs := f.fetchMRsByStatus("COMPLETED")

	// Track all rigs that have MRs
	for _, mr := range queuedMRs {
		rigSet[mr.rig] = true
	}
	for _, mr := range inProgressMRs {
		rigSet[mr.rig] = true
	}

	var statuses []RefineryStatus
	for rig := range rigSet {
		rs := RefineryStatus{
			Rig:     rig,
			Running: rigRunning[rig],
		}

		// Build queue from open MRs for this rig
		for _, mr := range queuedMRs {
			if mr.rig == rig {
				rs.Queue = append(rs.Queue, mr.MergeRequest)
			}
		}

		// Current = in-progress MR for this rig
		for _, mr := range inProgressMRs {
			if mr.rig == rig {
				current := mr.MergeRequest
				rs.Current = &current
				break
			}
		}

		rs.QueueDepth = len(rs.Queue)
		if rs.Current != nil {
			rs.QueueDepth++
		}

		// History from completed MRs (last 10)
		count := 0
		for _, mr := range completedMRs {
			if mr.rig == rig && count < 10 {
				rs.History = append(rs.History, mr.MergeRequest)
				count++
			}
		}

		// Calculate metrics from history
		if len(rs.History) > 0 {
			passed := 0
			for _, mr := range rs.History {
				if mr.Status == "COMPLETED" || mr.Status == "merged" {
					passed++
				}
			}
			rs.SuccessRate = float64(passed) / float64(len(rs.History)) * 100
		}

		statuses = append(statuses, rs)
	}

	return statuses, nil
}

// rigMR pairs a MergeRequest with its rig for internal routing.
type rigMR struct {
	rig string
	MergeRequest
}

// fetchMRsByStatus lists MR-type beads with a given status.
func (f *Fetcher) fetchMRsByStatus(status string) []rigMR {
	stdout, err := f.runBdCmd("list", "--type=mr", "--status="+strings.ToLower(status), "--json")
	if err != nil {
		return nil
	}

	var issues []IssueDetail
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return nil
	}

	var results []rigMR
	for _, iss := range issues {
		rig := extractRig(iss.ID)
		mr := MergeRequest{
			ID:     iss.ID,
			BeadID: iss.ID,
			Title:  iss.Title,
			Status: iss.Status,
		}
		results = append(results, rigMR{rig: rig, MergeRequest: mr})
	}
	return results
}

// extractRig extracts the rig name from a bead ID prefix.
// Bead IDs use format like "kt-xxx" where "kt" maps to a rig.
func extractRig(id string) string {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) < 2 {
		return "unknown"
	}
	return parts[0]
}

// FetchConvoys runs bd list --type=convoy --status=open --json in TownRoot.
func (f *Fetcher) FetchConvoys() ([]ConvoyInfo, error) {
	stdout, err := f.runBdCmd("list", "--type=convoy", "--status=open", "--json")
	if err != nil {
		return nil, fmt.Errorf("listing convoys: %w", err)
	}

	var convoys []ConvoyInfo
	if err := json.Unmarshal(stdout.Bytes(), &convoys); err != nil {
		return nil, fmt.Errorf("parsing convoy list: %w", err)
	}
	return convoys, nil
}

// FetchTrackedIssues runs bd dep list <convoyID> -t tracks --json, then
// bd show <ids> --json to get full issue details.
func (f *Fetcher) FetchTrackedIssues(convoyID string) ([]IssueDetail, error) {
	// Get tracked dependency IDs
	stdout, err := f.runBdCmd("dep", "list", convoyID, "-t", "tracks", "--json")
	if err != nil {
		return nil, fmt.Errorf("listing tracked deps for %s: %w", convoyID, err)
	}

	var deps []TrackedDep
	if err := json.Unmarshal(stdout.Bytes(), &deps); err != nil {
		return nil, fmt.Errorf("parsing tracked deps: %w", err)
	}

	if len(deps) == 0 {
		return nil, nil
	}

	// Collect IDs for batch fetch
	ids := make([]string, len(deps))
	for i, d := range deps {
		ids[i] = d.ID
	}

	// Batch fetch issue details: bd show <id1> <id2> ... --json
	args := append([]string{"show"}, ids...)
	args = append(args, "--json")

	stdout, err = f.runBdCmd(args...)
	if err != nil {
		return nil, fmt.Errorf("fetching issue details: %w", err)
	}

	var issues []IssueDetail
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return nil, fmt.Errorf("parsing issue details: %w", err)
	}
	return issues, nil
}
