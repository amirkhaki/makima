package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/amirkhaki/makima/internal/cli"
	"github.com/amirkhaki/makima/internal/daemon"
	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/engine"
	"github.com/amirkhaki/makima/internal/tracker"
	"github.com/amirkhaki/makima/internal/todo"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	args := os.Args[1:]
	cmd := args[0]

	switch cmd {
	case "daemon":
		daemonCmd(args[1:])
	case "status":
		statusCmd(args[1:])
	case "rule":
		ruleCmd(args[1:])
	case "category":
		categoryCmd(args[1:])
	case "todo":
		todoCmd(args[1:])
	case "config":
		configCmd(args[1:])
	case "log":
		logCmd(args[1:])
	case "version":
		fmt.Println("makima", version)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `makima - window-aware automation

Usage: makima <command> [subcommand] [flags]

Commands:
  daemon          Manage the background daemon
    start         Start the daemon
    stop          Stop the daemon
    restart       Restart the daemon

  status          Show daemon and system status

  rule            Manage rules
    list          List all rules
    add           Add a new rule
    remove        Remove a rule
    enable        Enable a rule
    disable       Disable a rule

  category        Manage categories
    list          List all categories
    add           Add a new category
    remove        Remove a category

  todo            Manage todos
    list          List all todos
    add           Add a new todo
    done          Mark a todo as complete
    remove        Remove a todo
    tree          Show todo tree

  config          Show or set configuration
  log             Show daemon logs
  version         Show version
  help            Show this help message
`)
}

func daemonCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: makima daemon <start|stop|restart>")
		os.Exit(1)
	}

	switch args[0] {
	case "start":
		startDaemon()
	case "stop":
		fmt.Println("daemon: stop - not implemented yet")
	case "restart":
		fmt.Println("daemon: restart - not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "Unknown daemon subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func startDaemon() {
	// Create state
	state := tracker.NewState()

	// Create session manager
	sessionMgr := engine.NewSessionManager()

	// Create trackers
	hyprland := tracker.NewHyprlandTracker(state)
	chrome := tracker.NewChromeTracker(state)

	// Create action executor
	actionExecutor := engine.NewActionExecutor(state, chrome)

	// Create rule engine
	ruleEngine := engine.NewEngine(state)

	// Load categories and rules from config
	configDir := getConfigDir()
	makimaFile, err := dsl.LoadConfigDir(configDir)
	if err == nil {
		for k, v := range makimaFile.Categories {
			ruleEngine.AddCategory(k, v)
		}
		for _, rule := range makimaFile.Rules {
			ruleEngine.AddRule(rule)
		}
	}

	// Create daemon
	d := daemon.NewDaemon(state, sessionMgr, actionExecutor, ruleEngine)
	d.AddTracker(hyprland)
	d.AddTracker(chrome)

	// Create socket server
	sockPath := getSocketPath()
	socketServer, err := daemon.NewSocketServer(sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create socket server: %v\n", err)
		os.Exit(1)
	}
	defer socketServer.Close()

	// Set up request handler
	socketServer.SetHandler(func(req daemon.Request) daemon.Response {
		return handleRequest(req, state, ruleEngine, sessionMgr)
	})

	// Start socket server
	go socketServer.Serve()

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Println("Makima daemon started")
	fmt.Printf("Socket: %s\n", sockPath)

	// Run daemon
	if err := d.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err)
		os.Exit(1)
	}
}

func handleRequest(req daemon.Request, state *tracker.State, ruleEngine *engine.Engine, sessionMgr *engine.SessionManager) daemon.Response {
	switch req.Method {
	case "status":
		return daemon.Response{
			ID: req.ID,
			Result: map[string]interface{}{
				"browser":   state.GetBrowser(),
				"hyprland":  state.GetHyprland(),
				"version":   version,
				"running":   true,
			},
		}
	case "rule.list":
		rules := ruleEngine.GetRules()
		return daemon.Response{
			ID:     req.ID,
			Result: rules,
		}
	case "todo.list":
		store, err := todo.NewStore(getConfigDir() + "/todos.json")
		if err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		todos := store.List()
		return daemon.Response{
			ID:     req.ID,
			Result: todos,
		}
	case "todo.add":
		var params struct {
			Text     string  `json:"text"`
			ParentID *string `json:"parent_id,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		store, err := todo.NewStore(getConfigDir() + "/todos.json")
		if err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		parentID := ""
		if params.ParentID != nil {
			parentID = *params.ParentID
		}
		id, err := store.Add(params.Text, parentID)
		if err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		return daemon.Response{
			ID:     req.ID,
			Result: id,
		}
	case "todo.done":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		store, err := todo.NewStore(getConfigDir() + "/todos.json")
		if err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		if err := store.Complete(params.ID); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		return daemon.Response{ID: req.ID, Result: "ok"}
	case "todo.remove":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		store, err := todo.NewStore(getConfigDir() + "/todos.json")
		if err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		if err := store.Remove(params.ID); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		return daemon.Response{ID: req.ID, Result: "ok"}
	default:
		return daemon.Response{
			ID:    req.ID,
			Error: fmt.Sprintf("unknown method: %s", req.Method),
		}
	}
}

func statusCmd(args []string) {
	client, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	status, err := client.RuleList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Makima %s\n", version)
	fmt.Printf("Rules: %d\n", len(status))
}

func ruleCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: makima rule <list|add|remove|enable|disable>")
		os.Exit(1)
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	switch args[0] {
	case "list":
		rules, err := client.RuleList()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list rules: %v\n", err)
			os.Exit(1)
		}
		for i, rule := range rules {
			fmt.Printf("%d. %v\n", i+1, rule)
		}
	case "add":
		fmt.Println("rule: add - not implemented yet")
	case "remove":
		fmt.Println("rule: remove - not implemented yet")
	case "enable":
		fmt.Println("rule: enable - not implemented yet")
	case "disable":
		fmt.Println("rule: disable - not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "Unknown rule subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func categoryCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: makima category <list|add|remove>")
		os.Exit(1)
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	switch args[0] {
	case "list":
		categories, err := client.CategoryList()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list categories: %v\n", err)
			os.Exit(1)
		}
		for name, patterns := range categories {
			fmt.Printf("%s: %v\n", name, patterns)
		}
	case "add":
		fmt.Println("category: add - not implemented yet")
	case "remove":
		fmt.Println("category: remove - not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "Unknown category subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func todoCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: makima todo <list|add|done|remove|tree>")
		os.Exit(1)
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	switch args[0] {
	case "list":
		todos, err := client.TodoList()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list todos: %v\n", err)
			os.Exit(1)
		}
		for _, todo := range todos {
			status := " "
			if todo.Completed {
				status = "x"
			}
			fmt.Printf("[%s] %s\n", status, todo.Text)
		}
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima todo add <text>")
			os.Exit(1)
		}
		id, err := client.TodoAdd(args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to add todo: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Todo added with ID: %s\n", id)
	case "done":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima todo done <id>")
			os.Exit(1)
		}
		err := client.TodoDone(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to mark todo done: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Todo marked done")
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima todo remove <id>")
			os.Exit(1)
		}
		err := client.TodoRemove(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove todo: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Todo removed")
	case "tree":
		store, err := todo.NewStore(getConfigDir() + "/todos.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load todos: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(store.TreeString())
	default:
		fmt.Fprintf(os.Stderr, "Unknown todo subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func configCmd(args []string) {
	fmt.Println("config: not implemented yet")
}

func logCmd(args []string) {
	fmt.Println("log: not implemented yet")
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home + "/.config/makima"
}

func getSocketPath() string {
	return "/tmp/makima.sock"
}

func newClient() (*cli.Client, error) {
	return cli.NewClient(getSocketPath())
}
