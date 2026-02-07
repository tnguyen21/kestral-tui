package data

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
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

// FetchRigs runs gt rig list and returns rig names.
func (f *Fetcher) FetchRigs() ([]string, error) {
	stdout, err := runCmd(cmdTimeout, "gt", "rig", "list")
	if err != nil {
		return nil, fmt.Errorf("listing rigs: %w", err)
	}

	var rigs []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			rigs = append(rigs, name)
		}
	}
	return rigs, nil
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

// FetchMail runs gt mail inbox --all --json and parses the result.
func (f *Fetcher) FetchMail() ([]MailMessage, error) {
	stdout, err := runCmd(cmdTimeout, "gt", "mail", "inbox", "--all", "--json")
	if err != nil {
		return nil, fmt.Errorf("fetching mail: %w", err)
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var messages []MailMessage
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil, fmt.Errorf("parsing mail: %w", err)
	}
	return messages, nil
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

// FetchResources collects CPU and memory usage for each tmux session by:
// 1. Listing all sessions with pane PIDs
// 2. Getting all process stats in one ps call
// 3. Walking the process tree to aggregate per-session
func (f *Fetcher) FetchResources() ([]SessionResource, error) {
	// Step 1: Get session metadata
	sessOut, err := runCmd(tmuxCmdTimeout, "tmux", "list-sessions", "-F",
		"#{session_name}|#{session_created}|#{session_activity}")
	if err != nil {
		return nil, fmt.Errorf("listing tmux sessions: %w", err)
	}

	type sessionMeta struct {
		name     string
		created  int64
		activity int64
	}
	sessions := make(map[string]*sessionMeta)
	for _, line := range strings.Split(strings.TrimSpace(sessOut.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		sm := &sessionMeta{name: parts[0]}
		fmt.Sscanf(parts[1], "%d", &sm.created)
		fmt.Sscanf(parts[2], "%d", &sm.activity)
		sessions[sm.name] = sm
	}

	if len(sessions) == 0 {
		return nil, nil
	}

	// Step 2: Get pane PIDs per session
	paneOut, err := runCmd(tmuxCmdTimeout, "tmux", "list-panes", "-a", "-F",
		"#{session_name}|#{pane_pid}")
	if err != nil {
		return nil, fmt.Errorf("listing tmux panes: %w", err)
	}

	// sessionName -> list of pane PIDs
	sessionPanes := make(map[string][]int)
	for _, line := range strings.Split(strings.TrimSpace(paneOut.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		sessionPanes[parts[0]] = append(sessionPanes[parts[0]], pid)
	}

	// Step 3: Get all process info in one call
	psOut, err := runCmd(tmuxCmdTimeout, "ps", "ax", "-o", "pid=,ppid=,pcpu=,rss=", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("listing processes: %w", err)
	}

	type procInfo struct {
		pid  int
		ppid int
		cpu  float64
		rss  int64 // KB
	}

	// Parse all processes
	var procs []procInfo
	children := make(map[int][]int) // ppid -> child pids
	pidIdx := make(map[int]int)     // pid -> index in procs

	scanner := bufio.NewScanner(psOut)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		rss, _ := strconv.ParseInt(fields[3], 10, 64)

		idx := len(procs)
		procs = append(procs, procInfo{pid: pid, ppid: ppid, cpu: cpu, rss: rss})
		children[ppid] = append(children[ppid], pid)
		pidIdx[pid] = idx
	}

	// Step 4: For each session, walk the tree from pane PIDs and aggregate
	now := time.Now()
	var results []SessionResource

	for sessName, sm := range sessions {
		panePIDs := sessionPanes[sessName]
		if len(panePIDs) == 0 {
			results = append(results, SessionResource{
				Name:       sessName,
				UptimeSecs: now.Unix() - sm.created,
				ActivityTS: sm.activity,
			})
			continue
		}

		// BFS to collect all descendant PIDs
		visited := make(map[int]bool)
		queue := make([]int, 0, len(panePIDs))
		for _, pid := range panePIDs {
			queue = append(queue, pid)
			visited[pid] = true
		}

		var totalCPU float64
		var totalRSS int64
		var procCount int

		for len(queue) > 0 {
			pid := queue[0]
			queue = queue[1:]

			if idx, ok := pidIdx[pid]; ok {
				p := procs[idx]
				totalCPU += p.cpu
				totalRSS += p.rss
				procCount++
			}

			for _, child := range children[pid] {
				if !visited[child] {
					visited[child] = true
					queue = append(queue, child)
				}
			}
		}

		results = append(results, SessionResource{
			Name:         sessName,
			CPUPercent:   totalCPU,
			MemRSS:       totalRSS * 1024, // convert KB to bytes
			ProcessCount: procCount,
			UptimeSecs:   now.Unix() - sm.created,
			ActivityTS:   sm.activity,
		})
	}

	return results, nil
}

// FetchPullRequests runs gh pr list --json for each rig discovered via git remotes
// and returns aggregated PR info.
func (f *Fetcher) FetchPullRequests() ([]PRInfo, error) {
	stdout, err := runCmd(ghCmdTimeout, "gh", "pr", "list",
		"--json", "number,title,author,headRefName,createdAt,isDraft,reviewDecision,mergeable,additions,deletions,changedFiles,url,statusCheckRollup",
		"--limit", "50",
	)
	if err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}

	var prs []PRInfo
	if err := json.Unmarshal(stdout.Bytes(), &prs); err != nil {
		return nil, fmt.Errorf("parsing PR list: %w", err)
	}
	return prs, nil
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

// FetchAgentBranch returns the current git branch for a polecat's worktree.
func (f *Fetcher) FetchAgentBranch(rig, name string) string {
	worktree := filepath.Join(f.TownRoot, rig, "polecats", name)
	stdout, err := runCmd(cmdTimeout, "git", "-C", worktree, "branch", "--show-current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

// FetchAgentCommits returns the last N commits from a polecat's worktree.
func (f *Fetcher) FetchAgentCommits(rig, name string, count int) []CommitInfo {
	worktree := filepath.Join(f.TownRoot, rig, "polecats", name)
	stdout, err := runCmd(cmdTimeout, "git", "-C", worktree, "log",
		"--oneline", fmt.Sprintf("-n%d", count))
	if err != nil {
		return nil
	}

	var commits []CommitInfo
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		c := CommitInfo{Hash: parts[0]}
		if len(parts) > 1 {
			c.Message = parts[1]
		}
		commits = append(commits, c)
	}
	return commits
}

// FetchAgentOutput captures recent tmux pane output for an agent session.
func (f *Fetcher) FetchAgentOutput(rig, name string, lines int) string {
	sessionName := fmt.Sprintf("gt-%s-%s", rig, name)
	stdout, err := runCmd(tmuxCmdTimeout, "tmux", "capture-pane",
		"-t", sessionName, "-p", fmt.Sprintf("-S-%d", lines))
	if err != nil {
		return ""
	}
	return stdout.String()
}

// FetchWitnesses detects witness sessions from tmux, computes heartbeat
// status, and counts managed polecats per rig.
func (f *Fetcher) FetchWitnesses() ([]WitnessDetail, error) {
	stdout, err := runCmd(tmuxCmdTimeout, "tmux", "list-sessions", "-F",
		"#{session_name}|#{window_activity}|#{session_created}")
	if err != nil {
		return nil, fmt.Errorf("listing tmux sessions: %w", err)
	}

	type sessionExt struct {
		Name     string
		Activity int64
		Created  int64
	}

	var sessions []sessionExt
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		s := sessionExt{Name: parts[0]}
		fmt.Sscanf(parts[1], "%d", &s.Activity)
		fmt.Sscanf(parts[2], "%d", &s.Created)
		sessions = append(sessions, s)
	}

	now := time.Now()

	// Track all rigs and their polecats
	allRigs := make(map[string]bool)
	polecatCounts := make(map[string]int)
	witnessSessions := make(map[string]sessionExt)

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
		allRigs[rig] = true

		if name == "witness" {
			witnessSessions[rig] = s
		} else if name != "refinery" && name != "mayor" {
			polecatCounts[rig]++
		}
	}

	var witnesses []WitnessDetail
	for rig := range allRigs {
		wd := WitnessDetail{
			Rig:          rig,
			PolecatCount: polecatCounts[rig],
		}

		if ws, ok := witnessSessions[rig]; ok {
			wd.HasSession = true
			wd.LastHeartbeat = ws.Activity
			wd.SessionCreated = ws.Created

			if ws.Activity > 0 {
				age := now.Sub(time.Unix(ws.Activity, 0))
				switch {
				case age < 5*time.Minute:
					wd.Status = "alive"
				case age < 15*time.Minute:
					wd.Status = "stale"
				default:
					wd.Status = "dead"
				}
			} else {
				wd.Status = "dead"
			}
		} else {
			wd.Status = "dead"
		}

		witnesses = append(witnesses, wd)
	}

	sort.Slice(witnesses, func(i, j int) bool {
		return witnesses[i].Rig < witnesses[j].Rig
	})

	return witnesses, nil
}
