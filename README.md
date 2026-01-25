# Quarterdeck

A configurable terminal UI for viewing dashboards and executing commands, built
with [Bubbletea](https://github.com/charmbracelet/bubbletea).

```
┌─────────────────────────────────────────────────────────────────────┐
│ Quarterdeck     Hosts: 3 up    Services: 12 OK    Disk: 45%         │
├────────────────────────┬────────────────────────────────────────────┤
│                        │                                            │
│ Monitoring             │ {                                          │
│ > Icinga               │   "icinga2": {                             │
│   Failed Services      │     "status": "running",                   │
│                        │     "uptime": "3d 14h"                     │
│ Server                 │   }                                        │
│   Disk Usage           │ }                                          │
│   Processes            │                                            │
│                        │                                            │
│ Backups                │                                            │
│   Backup Status        │                                            │
│   Run Incremental      │                                            │
│                        │                                            │
├────────────────────────┴────────────────────────────────────────────┤
│ o: Open Web UI    r: Reload    enter: run    q: quit                │
└─────────────────────────────────────────────────────────────────────┘
```

## Features

- **Configurable status bar** - Run any script to provide status indicators
- **Hierarchical menu** - Define categories and actions in YAML
- **Vim-style navigation** - Use `j`/`k` or arrow keys
- **Run commands** - Execute shell commands with streaming output
- **Open URLs** - Launch browser for web UIs

## Installation

### Binary (Linux)

Download the latest binary from
[GitHub Releases](https://github.com/sourcehaven-bv/quarterdeck/releases):

```bash
# amd64
curl -Lo quarterdeck https://github.com/sourcehaven-bv/quarterdeck/releases/latest/download/quarterdeck-linux-amd64
chmod +x quarterdeck
sudo mv quarterdeck /usr/local/bin/

# arm64
curl -Lo quarterdeck https://github.com/sourcehaven-bv/quarterdeck/releases/latest/download/quarterdeck-linux-arm64
chmod +x quarterdeck
sudo mv quarterdeck /usr/local/bin/
```

### Docker

```bash
docker pull ghcr.io/sourcehaven/quarterdeck:latest

# Run with config mounted
docker run -it --rm \
  -v ~/.config/quarterdeck/config.yaml:/etc/quarterdeck/config.yaml:ro \
  ghcr.io/sourcehaven/quarterdeck:latest
```

### From source

```bash
git clone https://github.com/sourcehaven-bv/quarterdeck.git
cd quarterdeck
make build
sudo make install
```

## Configuration

Create `config.yaml` (searched in `./`, `~/.config/quarterdeck/`,
`/etc/quarterdeck/`):

```yaml
# Status bar - displays output from any script
status:
  command: "./status.sh"
  interval: 5s

# Menu structure
menu:
  - category: Monitoring
    items:
      - name: Open Icinga
        open: "https://icinga.example.com"

      - name: Failed Services
        run: "systemctl list-units --failed"

  - category: Logs
    items:
      - name: View Logs
        run: "journalctl -f -n 50"

  - category: Backups
    items:
      - name: Backup Status
        run: "pgbackrest info"

      - name: Run Incremental
        run: "pgbackrest backup --type=incr"
        mode: confirm
```

### Status Script

The status command can output any text. The output is displayed directly in the
status bar, preserving ANSI colors. Long output wraps with proper indentation
and truncates after 3 lines.

```bash
#!/bin/bash
echo "Hosts: 3 up    Services: 12 OK    Disk: 45%"
```

For colored output, use ANSI escape codes or tools like `gum`:

```bash
#!/bin/bash
gum style --foreground 42 "Hosts: 3 up" | tr '\n' ' '
gum style --foreground 214 "Disk: 72%" | tr '\n' ' '
echo
```

The script is called at the configured `interval` (default: 5s).

### Menu Items

Menu items support two modes:

#### Standard mode (press Enter to execute)

```yaml
- name: Run Backup
  run: "pgbackrest backup --type=incr"
  mode: confirm  # optional: prompt before running
```

#### Auto-execute mode (runs on selection)

```yaml
- name: Disk Usage
  run: "df -h"
  mode: auto  # executes as you navigate to it
```

Great for status views that should update as you browse the menu.

#### Periodic refresh with interval

```yaml
- name: Live Processes
  run: "ps aux | head -20"
  mode: auto
  interval: 5s  # re-run every 5 seconds while focused
```

The `interval` option re-runs the command periodically while the menu item is
focused. Useful for monitoring commands. Minimum interval is 1 second.

#### Multiple actions with keybindings

```yaml
- name: Icinga
  run: "curl -s https://icinga/api/status | jq ."
  mode: auto
  actions:
    - key: o
      name: Open Web UI
      open: "https://icinga.example.com"
    - key: r
      name: Restart Service
      run: "systemctl restart icinga2"
      mode: confirm
```

When this item is selected, pressing `o` opens the URL, `r` runs the restart
command.

#### Action types

| Key        | Description                                                      |
| ---------- | ---------------------------------------------------------------- |
| `run`      | Execute a shell command, stream output to pane                   |
| `open`     | Open a URL in the default browser                                |
| `mode`     | Execution mode: `auto` (run on select), `confirm` (prompt first) |
| `interval` | Re-run command periodically while focused (e.g. `5s`, `1m`)      |
| `actions`  | List of additional actions with keybindings                      |

## Keybindings

### Global

| Key            | Action                              |
| -------------- | ----------------------------------- |
| `Tab`          | Switch between menu and output pane |
| `q` / `Ctrl+C` | Quit                                |

### Menu pane

| Key          | Action                                |
| ------------ | ------------------------------------- |
| `j` / `↓`    | Move down                             |
| `k` / `↑`    | Move up                               |
| `g` / `Home` | Go to top                             |
| `G` / `End`  | Go to bottom                          |
| `Enter`      | Execute selected item's `run` command |
| `a-z`        | Trigger item action bound to that key |

### Output pane

| Key          | Action           |
| ------------ | ---------------- |
| `j` / `↓`    | Scroll down      |
| `k` / `↑`    | Scroll up        |
| `g` / `Home` | Scroll to top    |
| `G` / `End`  | Scroll to bottom |

## Full Example

```yaml
status:
  command: "./status.sh"
  interval: 5s

menu:
  - category: Monitoring
    items:
      - name: Icinga
        run: "curl -s https://icinga:5665/v1/status | jq ."
        mode: auto
        actions:
          - key: o
            name: Open Web UI
            open: "https://icinga.example.com"
          - key: r
            name: Reload
            run: "systemctl reload icinga2"

      - name: Failed Services
        run: "systemctl list-units --failed"
        mode: auto
        interval: 10s

  - category: Server
    items:
      - name: Disk Usage
        run: "df -h"
        mode: auto
        interval: 30s

      - name: Processes
        run: "htop"

  - category: Backups
    items:
      - name: Backup Status
        run: "pgbackrest info"
        mode: auto

      - name: Run Incremental
        run: "pgbackrest backup --type=incr"
        mode: confirm

  - category: Deploy
    items:
      - name: Deploy App
        run: "./deploy.sh"
        mode: confirm
        actions:
          - key: l
            name: View Logs
            run: "tail -f /var/log/deploy.log"
          - key: s
            name: Status
            run: "systemctl status myapp"
```

## Example Status Scripts

### Simple status

```bash
#!/bin/bash
echo "Hosts: 3 up    Services: OK    Disk: 45%"
```

### Colored status with gum

```bash
#!/bin/bash
# Using gum for easy ANSI colors (https://github.com/charmbracelet/gum)
hosts=$(gum style --foreground 42 "Hosts: 3 up")
disk_pct=$(df -h / | awk 'NR==2 {gsub("%",""); print $5}')
if [[ $disk_pct -gt 80 ]]; then
  disk=$(gum style --foreground 196 "Disk: ${disk_pct}%")
else
  disk=$(gum style --foreground 42 "Disk: ${disk_pct}%")
fi
echo "$hosts    $disk"
```

### Raw ANSI colors

```bash
#!/bin/bash
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

pct=$(df -h / | awk 'NR==2 {gsub("%",""); print $5}')
if [[ $pct -gt 95 ]]; then
  color=$RED
elif [[ $pct -gt 80 ]]; then
  color=$YELLOW
else
  color=$GREEN
fi
echo -e "Hosts: ${GREEN}3 up${NC}    Disk: ${color}${pct}%${NC}"
```

## Development

```bash
make deps    # Install dependencies
make run     # Run locally
make lint    # Lint
make test    # Test
```

## Architecture

```
internal/
├── app/        # Bubbletea application (Model-Update-View)
├── config/     # YAML configuration loading
├── status/     # Status bar with external script support
└── ui/         # UI components
    ├── menu.go    # Hierarchical menu with vim navigation
    ├── output.go  # Output pane for command results
    └── styles.go  # Lipgloss styles
```
