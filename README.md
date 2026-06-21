# makima

A personal assistant for NixOS/Hyprland with rule-based automation.

## Features

- **Browser tracking** — Chrome CDP integration for tracking active tabs
- **Hyprland integration** — Workspace and window tracking via IPC
- **Rule engine** — Custom DSL for defining automation rules
- **Time budgets** — Set time limits on distracting sites with popup prompts
- **Category system** — Group websites into categories (games, social, etc.)
- **Hierarchical todos** — Nested todo lists with progress tracking
- **DMS plugin** — Quickshell widget for DankMaterialShell

## Installation

```bash
# Build from source
go build -o makima ./cmd/makima

# Or install with Nix
nix build
```

## Usage

```bash
# Start the daemon
./makima daemon start

# Start with verbose logging
./makima daemon start --verbose

# Stop the daemon
./makima daemon stop

# Check status
./makima status

# View logs
./makima log
```

## Configuration

Configuration files are stored in `~/.config/makima/`:

### categories.makima

Define website categories:

```
category games: *.game.com, *.io, *steam*
category social: *.twitter.com, *.reddit.com
category entertainment: *.youtube.com, *.netflix.com
```

### rules.makima

Define automation rules:

```
# Game sites: ask for time budget
when entering browser.category is games then popup "How much time?" budget [5, 15, 30]

# Social media warning
when browser.category is social then notify "Social media check" "You've been on social media for 1 minute."
```

## CLI Commands

```bash
# Daemon management
makima daemon start [--verbose]
makima daemon stop
makima daemon restart

# Status and logs
makima status
makima log [lines]

# Rule management
makima rule list
makima rule add <name> <condition> <action>
makima rule remove <id>
makima rule enable <id>
makima rule disable <id>

# Category management
makima category list
makima category add <name> <patterns...>
makima category remove <name>

# Todo management
makima todo list
makima todo add <text>
makima todo add <text> --parent <id>
makima todo done <id>
makima todo remove <id>
makima todo tree

# Configuration
makima config
makima config path
makima config categories
makima config rules
```

## DSL Syntax

### Conditions

- `browser.category is <name>` — Match category
- `browser.url matches <pattern>` — Match URL pattern
- `browser.tab.title matches <pattern>` — Match tab title
- `app.<name>.running` — App is running
- `workspace.count > <n>` — Workspace count
- `window.class matches <pattern>` — Window class match

### Actions

- `popup "<message>" budget [5, 15, 30]` — Show popup with time options
- `hyprctl "<command>"` — Execute Hyprland command
- `notify "<title>" "<body>"` — Desktop notification
- `cdp close-tab` — Close current browser tab
- `cdp navigate "<url>"` — Navigate to URL
- `exec "<command>"` — Execute shell command

### Triggers

- `when` — Fire on every match
- `when entering` — Fire once per URL visit

## Architecture

```
cmd/makima/main.go          — CLI entry point, daemon lifecycle
internal/dsl/               — MakimaScript language parser
internal/tracker/           — System state sources (Hyprland, Chrome)
internal/engine/            — Rule evaluation and action execution
internal/daemon/            — Background daemon with socket server
internal/cli/               — IPC client for CLI commands
internal/todo/              — Hierarchical todo store
plugin/                     — Quickshell QML components
```

## Development

```bash
# Enter dev shell
nix develop

# Run tests
go test ./...

# Build
go build -o makima ./cmd/makima
```

## License

MIT
