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
