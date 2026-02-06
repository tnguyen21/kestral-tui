# Kestral TUI

Mobile-first SSH dashboard for [Gas Town](https://github.com/tnguyen21/gastown). Monitor rigs, agents, and convoys from your phone using any SSH client.

Built with [charmbracelet/wish](https://github.com/charmbracelet/wish) + [bubbletea](https://github.com/charmbracelet/bubbletea).

## Quick Start

### 1. Build

```bash
# Clone
git clone https://github.com/tnguyen21/kestral-tui.git
cd kestral-tui

# Build (requires Go 1.22+)
go build -o kestral ./cmd/kestral

# Or install to ~/.local/bin
# (requires make — otherwise just cp the binary)
cp kestral ~/.local/bin/
```

### 2. Configure (optional)

Kestral works with zero config if your Gas Town workspace is at `~/gt`. To customize, create a config file:

```bash
mkdir -p ~/.config/kestral
cp configs/kestral.yaml.example ~/.config/kestral/kestral.yaml
```

Edit `~/.config/kestral/kestral.yaml`:

```yaml
# SSH listen port
port: 2222

# Path to Gas Town workspace
town_root: ~/gt

# Directory containing SSH host keys (auto-generated on first run)
host_key_dir: ~/.ssh

# Polling intervals in seconds
poll_interval:
  status: 10
  agents: 5
  convoys: 15
```

### 3. Run

```bash
# Start the SSH server (default port 2222)
./kestral

# Or override the port
./kestral -port 3333

# Or specify a config file
./kestral -config /path/to/kestral.yaml
```

You should see:

```
Kestral listening on :2222
```

### 4. Connect

From the same machine:

```bash
ssh localhost -p 2222
```

## Running on a VPS

To run Kestral as a persistent service on a VPS so you can connect from anywhere:

### Start as a background process

```bash
# Option A: tmux (recommended — easy to reattach)
tmux new-session -d -s kestral './kestral'

# Option B: nohup
nohup ./kestral > kestral.log 2>&1 &
```

### Firewall

Make sure port 2222 (or your chosen port) is open:

```bash
# If you have ufw
sudo ufw allow 2222/tcp

# Or via cloud provider security group / firewall rules
```

### Keep it running

If you want Kestral to survive reboots and you have systemd access:

```bash
# ~/.config/systemd/user/kestral.service
[Unit]
Description=Kestral TUI SSH Dashboard

[Service]
ExecStart=%h/.local/bin/kestral
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable --now kestral
```

If you don't have systemd access, the tmux approach works fine as a long-lived process.

## Connecting from Mobile

Kestral is designed for phone SSH clients. Any client that supports SSH will work.

### Termius (iOS / Android)

1. Install [Termius](https://termius.com/) from App Store / Play Store
2. Add a new host:
   - **Hostname**: your VPS IP or domain
   - **Port**: `2222`
   - **Username**: any (Kestral accepts all public keys)
3. Set up your SSH key:
   - In Termius, go to **Keychain** > **Keys**
   - Generate a new key or import your existing private key
   - Assign the key to your host
4. Connect — you'll land directly in the TUI

### Blink Shell (iOS)

1. Install [Blink Shell](https://blink.sh/) from App Store
2. In Blink's terminal, run:
   ```
   ssh your-vps-ip -p 2222
   ```
3. To save as a host, use Blink's config (`config` command):
   - Add host with port 2222 and your key

### Any SSH Client

Kestral is a standard SSH server. Connect with:

```bash
ssh <user>@<host> -p 2222 -i ~/.ssh/your_key
```

The username doesn't matter — Kestral accepts any public key authentication. What matters is that your client presents *a* key (password auth is not enabled).

### Generating and Transferring Keys

If you need to set up a key pair for your phone:

```bash
# On your VPS or local machine
ssh-keygen -t ed25519 -f ~/.ssh/kestral_mobile -C "phone"

# Transfer the PRIVATE key to your phone:
# Option A: QR code (for small keys like ed25519)
cat ~/.ssh/kestral_mobile | qrencode -t UTF8

# Option B: Airdrop / cloud transfer / paste into Termius key import
```

## Keybindings

| Key | Action |
|-----|--------|
| `1`-`7` | Jump to pane by number |
| `tab` | Next pane |
| `shift+tab` | Previous pane |
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `enter` | Select / expand |
| `esc` | Back |
| `r` | Force refresh all data |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

Mouse and touch input are supported — tap the tab bar to switch panes, scroll to navigate.

## Architecture

```
Phone (Termius/Blink)
  │
  └── SSH ──► Kestral (wish server, port 2222)
                │
                ├── bubbletea TUI per session
                │     ├── Dashboard pane  (rig health, sessions, convoys)
                │     └── Agents pane     (polecats, roles, current work)
                │
                └── data fetcher (shells out to gt/bd/tmux CLIs)
                      └── polls ~/gt workspace on configurable intervals
```

Each SSH connection gets its own independent TUI session with its own polling loop. Multiple users can connect simultaneously.

## Development

```bash
# Run tests
go test ./...

# Build
go build -o kestral ./cmd/kestral

# Run locally and connect
./kestral &
ssh localhost -p 2222
```
