package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
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

// FetchResources collects per-tmux-session CPU and memory usage
// by querying tmux for session panes and ps for process metrics.
func (f *Fetcher) FetchResources() ([]ResourceInfo, error) {
	// Step 1: Get session -> pane PIDs mapping from tmux
	stdout, err := runCmd(tmuxCmdTimeout, "tmux", "list-panes", "-a",
		"-F", "#{session_name}|#{pane_pid}|#{session_created}|#{session_activity}")
	if err != nil {
		return nil, fmt.Errorf("listing tmux panes: %w", err)
	}

	type sessionMeta struct {
		panePIDs []int
		created  int64
		activity int64
	}
	sessions := make(map[string]*sessionMeta)

	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		pid, _ := strconv.Atoi(parts[1])
		created, _ := strconv.ParseInt(parts[2], 10, 64)
		activity, _ := strconv.ParseInt(parts[3], 10, 64)

		if _, ok := sessions[name]; !ok {
			sessions[name] = &sessionMeta{created: created, activity: activity}
		}
		sessions[name].panePIDs = append(sessions[name].panePIDs, pid)
		if activity > sessions[name].activity {
			sessions[name].activity = activity
		}
	}

	// Step 2: Get all processes
	stdout, err = runCmd(tmuxCmdTimeout, "ps", "-e", "-o", "pid,ppid,pcpu,rss", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("listing processes: %w", err)
	}

	type proc struct {
		pid  int
		ppid int
		cpu  float64
		rss  int64 // KB
	}
	children := make(map[int][]int)
	procs := make(map[int]*proc)

	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		p := &proc{}
		p.pid, _ = strconv.Atoi(fields[0])
		p.ppid, _ = strconv.Atoi(fields[1])
		p.cpu, _ = strconv.ParseFloat(fields[2], 64)
		p.rss, _ = strconv.ParseInt(fields[3], 10, 64)

		procs[p.pid] = p
		children[p.ppid] = append(children[p.ppid], p.pid)
	}

	// Step 3: Walk process tree from each session's pane PIDs
	var collectDescendants func(pid int, seen map[int]bool)
	collectDescendants = func(pid int, seen map[int]bool) {
		if seen[pid] {
			return
		}
		seen[pid] = true
		for _, child := range children[pid] {
			collectDescendants(child, seen)
		}
	}

	now := time.Now()
	var resources []ResourceInfo

	for name, meta := range sessions {
		seen := make(map[int]bool)
		for _, pid := range meta.panePIDs {
			collectDescendants(pid, seen)
		}

		var totalCPU float64
		var totalRSS int64
		for pid := range seen {
			if p, ok := procs[pid]; ok {
				totalCPU += p.cpu
				totalRSS += p.rss
			}
		}

		var uptime time.Duration
		if meta.created > 0 {
			uptime = now.Sub(time.Unix(meta.created, 0))
		}

		resources = append(resources, ResourceInfo{
			SessionName:  name,
			CPU:          totalCPU,
			MemoryMB:     float64(totalRSS) / 1024.0,
			UptimeSecs:   int64(uptime.Seconds()),
			ProcessCount: len(seen),
			LastActivity: meta.activity,
		})
	}

	return resources, nil
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
