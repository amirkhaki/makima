# Makima Design Spec

## [S1] Architecture Overview

**Three-layer architecture:**

1. **Go daemon** (`makima daemon`) — background process that:
   - Manages tracking APIs (Hyprland IPC, Chrome CDP)
   - Runs the DSL rule engine
   - Executes actions (popups, hyprctl commands, CDP tab control)
   - Exposes a Unix socket for IPC with DMS plugin and CLI

2. **DMS plugin** — QML widget for DankMaterialShell:
   - Shows current status in bar (active window, browser URL)
   - Popout for viewing rules, todos, categories, and system state
   - Full CRUD operations for rules, categories, todos
   - Connects to daemon via Unix socket

3. **CLI** (`makima rule`, `makima todo`, `makima status`) — talks to daemon over socket

**Communication:** All components talk to the daemon via a single Unix socket using JSON-lines protocol.

## [S2] Custom DSL — "MakimaScript"

### Rule File Structure

```
# categories.makima — site categorization
category games {
  match "*.game.com"
  match "*.io"
  match "*steam*"
  match "*itch.io*"
}

category social {
  match "*.twitter.com"
  match "*.reddit.com"
  match "*.instagram.com"
}

category entertainment {
  match "*.youtube.com"
  match "*.netflix.com"
  match "*anime*"
}

# rules.makima — automation rules
when entering browser.category is games {
  budget prompt "How much time do you want to spend here?" {
    options: [5m, 15m, 30m]
    default: 15m
  }
  grace 30s
  then cdp close-tab
}

when app.mpv.running for 30m then hyprctl "monitor , special-sauce, multiply, 0.6"
when workspace changes then notify "Switched to workspace {workspace.name}"
```

### Condition Primitives

- `browser.url matches <pattern>` — regex/glob match on current URL
- `browser.tab.title matches <pattern>` — tab title match
- `browser.category is <name>` — matches category from categories.makima
- `browser.time_on_site` — duration on current site
- `browser.domain` — extracted domain
- `app.<name>.running` — process is running
- `app.<name>.running for <duration>` — process running for N time
- `workspace.count > <n>` — workspace count
- `window.class matches <pattern>` — active window class
- `time between <start> and <end>` — time-of-day guard
- Logical: `and`, `or`, `not`

### Action Primitives

- `cdp close-tab` — close the current tab via Chrome CDP
- `cdp close-domain <domain>` — close all tabs matching domain
- `cdp navigate "<url>"` — redirect to URL
- `cdp new-tab "<url>"` — open in new tab
- `cdp mute-tab` — mute audio on current tab
- `hyprctl "<command>"` — execute hyprctl command
- `popup "<msg>" for <duration>` — show popup
- `notify "<msg>"` — desktop notification
- `exec "<command>"` — arbitrary command

### Time Budget System

When a rule with `budget` triggers:

1. Non-dismissable popup appears for grace period
2. Popup asks "How much time do you want to spend here?"
3. User selects duration or lets timer expire (uses default)
4. During allowed time: user can browse freely
5. Leaving and returning doesn't re-prompt (same session)
6. When time budget exhausted: configured action fires (e.g., `cdp close-tab`)

### Session Logic

- **Grace period**: time before action fires after rule triggers
- **Cooldown**: time after action before rule can re-trigger
- **Session**: if user leaves site and returns within cooldown, it's a new session
- **Budget**: explicit time user chooses to spend on site

### State Variables

- `browser.url` — current URL
- `browser.tab.title` — tab title
- `browser.category` — matched category name
- `browser.time_on_site` — duration on current site
- `browser.domain` — extracted domain
- `app.<name>.running` / `app.<name>.uptime`
- `workspace.active`, `workspace.count`
- `window.class`, `window.title`
- `{variable}` interpolation in strings

### Category System

- Categories defined in `~/.config/makima/categories.makima`
- Glob patterns on URL or domain
- First-match wins if categories overlap
- Can be updated via CLI: `makima category add games "*.game.com"`
- Rules can target categories instead of raw URLs

## [S3] Go Daemon Structure

```
makima/
├── cmd/
│   └── makima/
│       └── main.go          # Entry point, CLI parsing
├── internal/
│   ├── daemon/
│   │   ├── daemon.go        # Main daemon loop, socket server
│   │   └── socket.go        # Unix socket IPC handler
│   ├── tracker/
│   │   ├── tracker.go       # Tracker interface
│   │   ├── hyprland.go      # Hyprland IPC tracker
│   │   ├── chrome.go        # Chrome CDP tracker
│   │   └── state.go         # Aggregated state
│   ├── dsl/
│   │   ├── lexer.go         # Tokenizer
│   │   ├── parser.go        # AST builder
│   │   ├── evaluator.go     # Rule evaluation
│   │   ├── categories.go    # Category loader
│   │   └── ast.go           # AST types
│   ├── engine/
│   │   ├── engine.go        # Rule engine (eval + actions)
│   │   ├── session.go       # Grace/cooldown/session tracking
│   │   └── actions.go       # Action executors
│   └── todo/
│       ├── todo.go          # Todo CRUD
│       └── store.go         # JSON file storage
├── plugin/
│   ├── plugin.json          # DMS plugin manifest
│   ├── MakimaWidget.qml     # Bar widget
│   ├── MakimaDaemon.qml     # Daemon connection
│   └── MakimaSettings.qml   # Settings UI
├── flake.nix                # Build system
├── flake.lock
├── go.mod
├── go.sum
└── rules.makima             # Default rule file
```

### Key Interfaces

```go
type Tracker interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    State() State
    Events() <-chan Event
}

type State struct {
    Browser   BrowserState
    Hyprland  HyprlandState
    Apps      map[string]AppStatus
}

type Rule struct {
    Condition Condition
    Actions   []Action
    Grace     time.Duration
    Cooldown  time.Duration
    Budget    *BudgetConfig
}
```

## [S4] Testing Strategy

### Test Structure

```
makima/
├── internal/
│   ├── dsl/
│   │   ├── lexer_test.go
│   │   ├── parser_test.go
│   │   ├── evaluator_test.go
│   │   └── categories_test.go
│   ├── engine/
│   │   ├── engine_test.go
│   │   ├── session_test.go
│   │   └── actions_test.go
│   ├── tracker/
│   │   ├── hyprland_test.go   # Mock Hyprland IPC
│   │   ├── chrome_test.go     # Mock CDP
│   │   └── state_test.go
│   ├── daemon/
│   │   └── socket_test.go     # Mock socket IPC
│   └── todo/
│       └── todo_test.go
```

### Test Categories

1. **Unit tests** — DSL parsing, rule evaluation, state aggregation
2. **Integration tests** — Daemon ↔ tracker communication, socket IPC
3. **Mock-based tests** — Mock Hyprland IPC responses, mock CDP commands
4. **Table-driven tests** — DSL syntax variations, edge cases
5. **Concurrency tests** — Multiple rules firing simultaneously, session state races

### Key Test Scenarios

- DSL parser: valid/invalid syntax, edge cases, comments
- Rule engine: grace period timing, cooldown logic, session tracking
- Categories: pattern matching, priority, overlap handling
- CDP: tab close, navigation, state queries
- Hyprland: workspace queries, window focus events
- Socket IPC: request/response, reconnect, error handling
- Todo: CRUD operations, persistence, concurrency, hierarchical nesting

## [S5] DMS Plugin Design

### Plugin Type

`composite` (daemon + widget + settings)

### plugin.json

```json
{
  "id": "makima",
  "name": "Makima",
  "description": "Personal assistant with rule-based automation",
  "version": "1.0.0",
  "author": "makima",
  "type": "composite",
  "capabilities": ["daemon", "dankbar-widget"],
  "components": {
    "daemon": "./MakimaDaemon.qml",
    "widget": "./MakimaWidget.qml"
  },
  "settings": "./MakimaSettings.qml",
  "requires_dms": ">=0.1.0",
  "permissions": ["settings_read", "settings_write"]
}
```

### Widget (Bar Pill)

- Shows current status: browser category, active app, timer countdown
- Click opens popout

### Popout Sections

1. **Dashboard** — live status (browser, apps, active rules)
2. **Rules** — list, add, edit, delete, enable/disable
3. **Categories** — list, add, edit, delete patterns
4. **Todos** — hierarchical tree view with nesting
5. **History** — recent rule triggers and actions

### Full CLI Parity

All CLI operations available in plugin:
- Rule CRUD with condition builder and action selector
- Category CRUD with pattern editor
- Todo CRUD with drag-and-drop reordering
- Status viewing and history

### IPC Extension

```
makima rule add        → {"method": "rule.add", "params": {...}}
makima rule list       → {"method": "rule.list"}
makima category add    → {"method": "category.add", "params": {...}}
makima todo add        → {"method": "todo.add", "params": {...}}
makima status          → {"method": "status"}
```

## [S6] CLI Design

```
makima daemon              # Start daemon
makima status              # Show current status
makima rule list           # List all rules
makima rule add <file>     # Add rule from file
makima rule remove <id>    # Remove rule
makima rule enable <id>    # Enable rule
makima rule disable <id>   # Disable rule
makima category list       # List all categories
makima category add <name> # Add category
makima category remove <name>
makima todo list           # List todos (hierarchical)
makima todo add <text>     # Add top-level todo
makima todo add <text> --parent <id>  # Add nested todo
makima todo tree           # Show todo hierarchy
makima todo done <id>      # Mark complete (cascades to children)
makima todo remove <id>    # Remove todo
makima config              # Show/edit config
makima log                 # Show recent events
```

### Hierarchical Todos

Structure supports nested todos (books with chapters, projects with tasks):

```json
{
  "id": "todo-1",
  "text": "Read Dune",
  "completed": false,
  "children": [
    {
      "id": "todo-1.1",
      "text": "Chapter 1: Desert Planet",
      "completed": true,
      "children": []
    }
  ]
}
```

- Collapsible tree view in plugin
- Completion cascades: completing parent marks all children complete
- Progress: show "3/5 chapters done" for parent items

## [S7] Data Flow

### Browser Tracking

```
Chrome CDP (WebSocket) → makima daemon → State update → Rule evaluation → Action execution
                    ↓
              DMS plugin (Unix socket)
```

### Hyprland Tracking

```
Hyprland IPC (socket) → makima daemon → State update → Rule evaluation → Action execution
                    ↓
              DMS plugin (Unix socket)
```

### Rule Evaluation

```
State change event → Rule engine → Check conditions → Evaluate categories → Check grace/cooldown → Execute actions
```

### CLI/Plugin → Daemon

```
CLI/Plugin → Unix socket → Daemon handler → Execute command → Response
```

### Event Loop

```go
func (d *Daemon) Run(ctx context.Context) {
    for {
        select {
        case event := <-d.hyprland.Events():
            d.state.Update(event)
            d.engine.Evaluate(d.state)
        case event := <-d.chrome.Events():
            d.state.Update(event)
            d.engine.Evaluate(d.state)
        case req := <-d.socket.Requests():
            d.handleRequest(req)
        case <-ctx.Done():
            return
        }
    }
}
```

## [S8] Nix Build System

### flake.nix

```nix
{
  description = "makima - Personal assistant with rule-based automation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.golangci-lint
            pkgs.gopls
          ];
        };

        packages.default = pkgs.buildGoModule {
          pname = "makima";
          version = "1.0.0";
          src = ./.;
          vendorHash = null;
          doCheck = true;
          checkPhase = ''
            go test ./...
          '';
        };

        nixosModules.default = import ./module.nix;
      }
    );
}
```

### module.nix (optional)

```nix
{ config, lib, pkgs, ... }:
{
  options.services.makima = {
    enable = lib.mkEnableOption "makima daemon";
  };

  config = lib.mkIf config.services.makima.enable {
    systemd.user.services.makima = {
      description = "Makima personal assistant";
      serviceConfig = {
        ExecStart = "${pkgs.makima}/bin/makima daemon";
        Restart = "on-failure";
      };
      wantedBy = [ "default.target" ];
    };
  };
}
```
