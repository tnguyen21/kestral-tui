# Interacting with Gas Town

Quick reference for requesting features, fixing bugs, and directing agents through Gas Town's workflow.

## The Basics

Gas Town has two CLIs:
- **`gt`** — orchestration (agents, rigs, mail, convoys, work dispatch)
- **`bd`** — issue tracking (create/list/close beads, dependencies, formulas)

The primary workflow: **write specs as beads → group into convoys → mail the Mayor → polecats do the work → refinery merges it**.

## Requesting Features

### 1. Create an issue (bead)

```bash
# Simple
bd create --title "Add Convoys pane to Kestral" --type task

# With description
bd create --title "Add Convoys pane" --type task \
  --description "Show active convoys with progress bars, issue breakdown, and completion percentages"

# With priority label
bd create --title "Add Convoys pane" --type task \
  --labels "priority:1"
```

### 2. Group into a convoy

```bash
# Create convoy tracking multiple issues
gt convoy create "Kestral Phase 2: New Panes" kt-abc kt-def kt-ghi

# Add more issues to an existing convoy later
gt convoy add hq-cv-xxxxx kt-jkl kt-mno
```

### 3. Mail the Mayor

```bash
# Send a task with context
gt mail send mayor/ -s "Phase 2: New Panes" \
  -m "Convoy hq-cv-xxxxx has 3 issues for new Kestral panes. Execute in dependency order." \
  --type task --priority 1
```

The Mayor reads the mail, reviews the convoy, and spawns polecats to execute.

### 4. Monitor progress

```bash
# Overall status
gt status

# Convoy progress
gt convoy status hq-cv-xxxxx
gt convoy list

# Watch in real time
gt status --watch
```

## Filing Bug Reports

```bash
# Create bug issue
bd create --title "Dashboard pane crashes on empty rig list" --type bug \
  --description "When no rigs are registered, the dashboard panics with nil pointer"

# Send directly to Mayor with high priority
gt mail send mayor/ -s "Bug: dashboard crash on empty rigs" \
  -m "Issue kt-xyz. Dashboard pane crashes when no rigs exist. Needs nil check." \
  --type task --priority 0
```

## Quick One-Off Requests

If you don't want to create issues manually, just mail the Mayor with instructions. The Mayor can create issues and convoys itself:

```bash
gt mail send mayor/ -s "Add a Convoys pane to Kestral TUI" \
  -m "Create a new pane that shows convoy status with progress bars. Reference the existing Dashboard pane for patterns. Create issues and a convoy, then execute." \
  --type task --priority 1
```

## Directing Agents

### Sling work to specific agents

```bash
# Auto-spawn a polecat in a rig
gt sling kt-abc kestral_tui

# To a specific polecat
gt sling kt-abc kestral_tui/quartz

# To crew member
gt sling kt-abc kestral_tui/crew/yourname
```

### Nudge a running agent

```bash
# Send synchronous message to a worker
gt nudge kestral_tui/quartz "What's your current status?"
gt nudge mayor/ "Prioritize kt-abc next"
```

### Check what agents are doing

```bash
# List all polecats
gt polecat list kestral_tui

# Peek at a polecat's recent output
gt peek kestral_tui/quartz

# Attach to watch live
tmux attach -t gt-kestral_tui-quartz
```

## Managing Rigs

```bash
# Boot a rig (starts witness + refinery)
gt rig boot kestral_tui

# Shutdown
gt rig shutdown kestral_tui

# Reboot
gt rig reboot kestral_tui

# List all rigs
gt rig list
```

## Managing the Mayor & Deacon

```bash
# Start/stop
gt mayor start
gt mayor stop
gt deacon start
gt deacon stop

# Attach to watch
gt mayor attach
gt deacon attach

# Check status
gt mayor status
gt deacon status
```

## Mail System

```bash
# Check your inbox
gt mail inbox

# Read a message
gt mail read <msg-id>

# Send to specific targets
gt mail send mayor/                    # Mayor
gt mail send kestral_tui/witness       # Rig's witness
gt mail send kestral_tui/quartz        # Specific polecat
gt mail send kestral_tui/              # Broadcast to whole rig

# Priority levels: 0 (urgent) → 4 (backlog)
gt mail send mayor/ -s "Subject" -m "Body" --priority 0
```

## Issue Management with `bd`

```bash
# List all issues
bd list

# Filter
bd list --status open
bd ready                               # Open with no blockers

# Show details
bd show kt-abc

# Add dependencies
bd dep add kt-def --blocked-by kt-abc  # kt-def waits for kt-abc

# Close
bd close kt-abc

# Search
bd search "dashboard"
```

## Convoy Lifecycle

```bash
# Create
gt convoy create "Name" kt-a kt-b kt-c

# Check progress
gt convoy status hq-cv-xxxxx

# List all
gt convoy list

# Auto-close completed convoys
gt convoy check

# Find stalled convoys
gt convoy stranded
```

## Common Patterns

### "I want a feature built"
```bash
gt mail send mayor/ -s "Build feature X" \
  -m "Description of what I want. Reference files: internal/pane/dashboard.go" \
  --type task --priority 1
```

### "Something is broken, fix it"
```bash
gt mail send mayor/ -s "Bug: description" \
  -m "Steps to reproduce. Expected vs actual behavior." \
  --type task --priority 0
```

### "I want to review what was built"
```bash
gt convoy status hq-cv-xxxxx           # See what's done
gh pr list                             # Check PRs
gt peek kestral_tui/quartz             # See agent output
```

### "Stop everything"
```bash
gt rig shutdown kestral_tui            # Stop rig agents
gt mayor stop                          # Stop Mayor
gt deacon stop                         # Stop Deacon
```

### "Start everything"
```bash
gt deacon start
gt mayor start
gt rig boot kestral_tui
```
