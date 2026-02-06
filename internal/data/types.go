// Package data provides types for unmarshalling Gas Town CLI JSON output.
package data

// TownStatus represents the overall status returned by gt status --json.
type TownStatus struct {
	Agents []AgentInfo `json:"agents"`
}

// AgentInfo represents a single agent in the town status.
type AgentInfo struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	State   string `json:"state"` // running, stopped, idle
}

// ConvoyInfo represents a convoy from bd list --type=convoy --json.
type ConvoyInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// TrackedDep represents a tracked dependency from bd dep list <id> -t tracks --json.
type TrackedDep struct {
	ID string `json:"id"`
}

// IssueDetail represents issue details from bd show <ids> --json.
type IssueDetail struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Assignee  string `json:"assignee"`
	UpdatedAt string `json:"updated_at"`
}

// SessionInfo represents a parsed tmux session.
type SessionInfo struct {
	Name     string
	Activity int64 // unix timestamp
}

// AgentDetail holds enriched agent info for the TUI agents pane.
type AgentDetail struct {
	Name       string `json:"name"`
	Rig        string `json:"rig"`
	Role       string `json:"role"`    // witness, refinery, polecat
	Status     string `json:"status"`  // working, stale, stuck, idle
	AgeSecs    int64  `json:"age_sec"` // seconds since last activity
	IssueID    string `json:"issue_id"`
	IssueTitle string `json:"issue_title"`
}
