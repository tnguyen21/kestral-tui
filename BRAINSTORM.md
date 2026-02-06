# Kestral TUI — Feature Brainstorm

## Command & Control (ship from phone)

- **Mail Composer** — inline compose mail to Mayor with priority picker, subject, body; vim-style text input or simple line editor
- **Quick Commands** — palette (`:` or `/`) for common actions: boot rig, stop rig, spawn polecat, kill polecat, check convoy, etc.
- **Issue Creator** — create beads/specs directly from TUI with title, description, tags, priority; attach to existing convoy or create new one
- **Convoy Builder** — select issues from a list, group into convoy, set dependencies, ship to Mayor in one shot
- **PR Review Pane** — view diffs, approve/request changes, merge PRs from phone; inline comment support
- **Rig Controls** — boot/stop/restart rigs, start/stop witness/refinery per rig; confirm dialog for destructive ops
- **Priority Override** — bump issue priority, reorder convoy work, reassign polecats to different issues
- **Mayor Directives** — send freeform instructions to Mayor ("pause kestral_tui rig", "focus on kt-xyz next", "skip tests on this one")
- **Dependency Editor** — add/remove `blocked-by` relationships between issues inline
- **Template Library** — saved spec templates for common issue types (bug fix, new pane, refactor, test coverage)

## Real-Time Monitoring

- **Live Log Tail** — stream polecat tmux output in real time; select agent, see scrolling log; regex filter on log lines
- **Agent Detail View** — expand an agent card to see: current issue, branch, recent commits, time on task, last N lines of output
- **Diff Preview** — see uncommitted changes a polecat has made so far (live `git diff` from worktree)
- **Commit Feed** — real-time stream of commits across all polecats; tap to see diff
- **Error Alerts** — visual banner/bell when a polecat hits an error, test failure, or goes idle unexpectedly; red badge on Agents tab
- **Health Sparklines** — tiny inline charts showing agent activity over last N minutes (commits, lines changed, test runs)
- **Session Timeline** — horizontal timeline view showing when each agent started, what it worked on, when it finished; gantt-style
- **Resource Monitor** — CPU/memory per tmux session; warn if a polecat is hung or spinning
- **Refinery Status** — merge queue depth, current PR being tested, pass/fail history
- **Witness Heartbeat** — show last heartbeat time per rig, highlight stale/dead witnesses

## Panes to Add

- **Convoys Pane** — dedicated view: active convoys with progress bars, issue breakdown, completion %, ETA based on velocity
- **Mail Pane** — inbox/outbox view; read mail, reply, compose new; mark read/unread; filter by sender/priority
- **Logs Pane** — multiplexed log viewer; pick which agent(s) to follow; color-coded by source
- **PRs Pane** — open PRs across all rigs; status checks, review state, mergability; one-tap merge
- **Issues Pane** — full bead browser with filters (status, priority, assignee, rig); search; bulk actions
- **Config Pane** — view/edit kestral.yaml live; adjust poll intervals, theme, keybindings without restarting
- **History Pane** — completed work log; which polecats did what, when; convoy completion history

## Navigation & UX

- **Command Palette** — fuzzy search across all actions, issues, agents, convoys; triggered by `/` or `:`
- **Breadcrumbs** — show navigation path (Agents > quartz > kt-7dv) so you know where you are on small screens
- **Deep Linking** — jump directly to an issue/agent/convoy from any pane; cross-reference tap targets
- **Split View** — on wide terminals (iPad landscape), show two panes side by side
- **Pinned Items** — pin specific agents, issues, or convoys to top of their respective lists
- **Search** — global `/` search across all panes; filter as you type
- **Notifications Bar** — scrolling ticker at top or bottom showing recent events (PR merged, convoy completed, agent error)
- **Focus Mode** — hide everything except one agent's live log; full screen, minimal chrome
- **Swipe Gestures** — left/right swipe to switch panes (if terminal supports mouse drag events)
- **Bookmark/Favorites** — star issues or agents for quick access across sessions

## Data & Intelligence

- **Velocity Metrics** — issues closed per hour, avg time per issue, throughput per polecat; trend arrows
- **Convoy ETA** — estimated completion based on historical velocity and remaining issues
- **Cost Tracker** — estimated API token spend per polecat session (if trackable via agent output)
- **Anomaly Detection** — flag agents that are taking unusually long, stuck in loops, or producing unusually large diffs
- **Daily Digest** — on-demand summary: "today: 8 issues closed, 3 convoys completed, 2 PRs merged, 1 error"
- **Comparison View** — side-by-side polecat performance; who's fast, who's stuck
- **Dependency Graph** — visual DAG of issue dependencies; highlight critical path
- **Burndown Chart** — ASCII art burndown for active convoy; track progress against time

## Polish & QOL

- **Persistent Sessions** — reconnect to same TUI state across SSH disconnects; server-side session storage
- **Theme Picker** — switch between Ayu, Dracula, Solarized, etc. from within TUI
- **Configurable Keybindings** — remap keys from config file; important for different phone keyboards
- **Clipboard Integration** — yank issue IDs, branch names, commit hashes via OSC52 escape sequences (works in Termius/Blink)
- **Touch-Friendly Sizing** — larger tap targets option for phone use; configurable padding
- **Status Bar Customization** — choose what shows in bottom bar (clock, agent count, convoy progress, etc.)
- **Auto-Refresh Indicator** — pulsing dot or spinner showing data is being polled; last-updated timestamp per section
- **Offline Resilience** — if gt/bd CLIs hang or fail, show last-known-good data with staleness warning instead of blank screen
- **Multi-Town Support** — connect to multiple Gas Town instances from one Kestral; switch between towns
- **SSH Tunneling Helper** — built-in instructions/config for connecting through jump hosts or tailscale
- **Audit Log** — record all commands sent through Kestral (mails, rig boots, PR approvals) for accountability
- **Sound/Bell Alerts** — terminal bell on configurable events (convoy complete, agent error); phone SSH apps support this
- **Export** — dump current view as markdown to clipboard or file; share status updates easily
- **Onboarding Tour** — first-run walkthrough highlighting key features and keybindings
