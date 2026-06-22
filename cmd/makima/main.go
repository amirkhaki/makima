package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/amirkhaki/makima/internal/cli"
	"github.com/amirkhaki/makima/internal/daemon"
	"github.com/amirkhaki/makima/internal/dsl"
	"github.com/amirkhaki/makima/internal/engine"
	"github.com/amirkhaki/makima/internal/log"
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
		fmt.Fprintln(os.Stderr, "Usage: makima daemon <start|stop|restart> [--verbose]")
		os.Exit(1)
	}

	// Check for --verbose flag
	for _, arg := range args {
		if arg == "--verbose" || arg == "-v" {
			log.SetVerbose(true)
		}
	}

	switch args[0] {
	case "start":
		startDaemon()
	case "stop":
		stopDaemon()
	case "restart":
		restartDaemon()
	default:
		fmt.Fprintf(os.Stderr, "Unknown daemon subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func stopDaemon() {
	sockPath := getSocketPath()
	lockPath := sockPath + ".lock"

	// Read PID from lock file
	data, err := os.ReadFile(lockPath)
	if err != nil {
		fmt.Println("No daemon running (no lock file)")
		return
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		fmt.Println("Invalid lock file")
		return
	}

	// Send SIGTERM to daemon
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Daemon process not found")
		return
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("Failed to stop daemon: %v\n", err)
		return
	}

	fmt.Println("Daemon stopped")
}

func restartDaemon() {
	stopDaemon()
	time.Sleep(1 * time.Second)
	startDaemon()
}

func startDaemon() {
	// Create state
	state := tracker.NewState()

	// Create session manager
	sessionMgr := engine.NewSessionManager()

	// Create trackers
	hyprland := tracker.NewHyprlandTracker(state)
	
	// Chrome tracker with configurable port file
	portFile := getBrowserPortFile()
	chrome := tracker.NewChromeTrackerWithPortFile(state, portFile)

	// Create action executor
	actionExecutor := engine.NewActionExecutor(state, chrome)

	// Create rule engine
	ruleEngine := engine.NewEngine(state, sessionMgr)

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
	d := daemon.NewDaemon(state, sessionMgr, actionExecutor, ruleEngine, chrome)
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

	// Connect daemon to socket server for broadcasting
	socketServer.SetDaemon(d)

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
	case "budget.select":
		var params struct {
			Minutes int `json:"minutes"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		log.Event("daemon", "budget selected: %d minutes", params.Minutes)
		// Store the budget for the current session
		sessionMgr.SetBudget(params.Minutes)
		return daemon.Response{ID: req.ID, Result: "ok"}
	case "rule.list":
		rules := ruleEngine.GetRules()
		// Convert to RuleInfo format for client
		var ruleInfos []map[string]interface{}
		for _, rule := range rules {
			// Convert condition to human-readable string
			conditionStr := conditionToString(rule.Condition)
			ruleInfos = append(ruleInfos, map[string]interface{}{
				"id":        rule.ID,
				"condition": conditionStr,
				"enabled":   rule.Enabled,
			})
		}
		return daemon.Response{
			ID:     req.ID,
			Result: ruleInfos,
		}
	case "rule.add":
		var params struct {
			Name      string `json:"name"`
			Condition string `json:"condition"`
			Action    string `json:"action"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		// Parse the rule from condition and action
		ruleStr := "when " + params.Condition + " then " + params.Action
		file, err := dsl.ParseMakimaFile(ruleStr)
		if err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		if len(file.Rules) > 0 {
			rule := file.Rules[0]
			ruleEngine.AddRule(rule)
			return daemon.Response{ID: req.ID, Result: rule.ID}
		}
		return daemon.Response{ID: req.ID, Error: "failed to parse rule"}
	case "rule.remove":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		// Find and remove rule by ID
		rules := ruleEngine.GetRules()
		for i, rule := range rules {
			if rule.ID == params.ID {
				ruleEngine.RemoveRule(i)
				return daemon.Response{ID: req.ID, Result: "ok"}
			}
		}
		return daemon.Response{ID: req.ID, Error: "rule not found"}
	case "rule.enable":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		// Find and enable rule by ID
		rules := ruleEngine.GetRules()
		for i, rule := range rules {
			if rule.ID == params.ID {
				ruleEngine.SetRuleEnabled(i, true)
				return daemon.Response{ID: req.ID, Result: "ok"}
			}
		}
		return daemon.Response{ID: req.ID, Error: "rule not found"}
	case "rule.disable":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		// Find and disable rule by ID
		rules := ruleEngine.GetRules()
		for i, rule := range rules {
			if rule.ID == params.ID {
				ruleEngine.SetRuleEnabled(i, false)
				return daemon.Response{ID: req.ID, Result: "ok"}
			}
		}
		return daemon.Response{ID: req.ID, Error: "rule not found"}
	case "category.list":
		categories := ruleEngine.GetCategories()
		result := make(map[string][]string)
		for name, cat := range categories {
			result[name] = cat.Patterns
		}
		return daemon.Response{
			ID:     req.ID,
			Result: result,
		}
	case "category.add":
		var params struct {
			Name     string   `json:"name"`
			Patterns []string `json:"patterns"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		ruleEngine.AddCategory(params.Name, &dsl.Category{
			Name:     params.Name,
			Patterns: params.Patterns,
		})
		return daemon.Response{ID: req.ID, Result: "ok"}
	case "category.remove":
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return daemon.Response{ID: req.ID, Error: err.Error()}
		}
		ruleEngine.RemoveCategory(params.Name)
		return daemon.Response{ID: req.ID, Result: "ok"}
	case "todo.list":
		store := getTodoStore()
		if store == nil {
			return daemon.Response{ID: req.ID, Error: "failed to load todo store"}
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
		store := getTodoStore()
		if store == nil {
			return daemon.Response{ID: req.ID, Error: "failed to load todo store"}
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
		store := getTodoStore()
		if store == nil {
			return daemon.Response{ID: req.ID, Error: "failed to load todo store"}
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
		store := getTodoStore()
		if store == nil {
			return daemon.Response{ID: req.ID, Error: "failed to load todo store"}
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

	status, err := client.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Makima %s\n", version)
	fmt.Printf("Running: %v\n", status["running"])
	if browser, ok := status["browser"].(map[string]interface{}); ok {
		fmt.Printf("Browser URL: %v\n", browser["url"])
		fmt.Printf("Browser Category: %v\n", browser["category"])
	}
	if hyprland, ok := status["hyprland"].(map[string]interface{}); ok {
		fmt.Printf("Window: %v\n", hyprland["windowClass"])
	}
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
			fmt.Printf("%d. %s (enabled: %v)\n", i+1, rule.Condition, rule.Enabled)
		}
	case "add":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: makima rule add <name> <condition> <action>")
			os.Exit(1)
		}
		id, err := client.RuleAdd(args[1], args[2], args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to add rule: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Rule added with ID: %s\n", id)
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima rule remove <id>")
			os.Exit(1)
		}
		if err := client.RuleRemove(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove rule: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rule removed")
	case "enable":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima rule enable <id>")
			os.Exit(1)
		}
		if err := client.RuleEnable(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to enable rule: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rule enabled")
	case "disable":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima rule disable <id>")
			os.Exit(1)
		}
		if err := client.RuleDisable(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to disable rule: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rule disabled")
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
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: makima category add <name> <patterns...>")
			os.Exit(1)
		}
		name := args[1]
		patterns := args[2:]
		if err := client.CategoryAdd(name, patterns); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to add category: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Category '%s' added\n", name)
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: makima category remove <name>")
			os.Exit(1)
		}
		if err := client.CategoryRemove(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove category: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Category '%s' removed\n", args[1])
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
		store := getTodoStore()
		if store == nil {
			fmt.Fprintf(os.Stderr, "Failed to load todos\n")
			os.Exit(1)
		}
		fmt.Println(store.TreeString())
	default:
		fmt.Fprintf(os.Stderr, "Unknown todo subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func configCmd(args []string) {
	configDir := getConfigDir()
	
	if len(args) == 0 {
		// Show config directory contents
		fmt.Printf("Config directory: %s\n", configDir)
		entries, err := os.ReadDir(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
			return
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				fmt.Printf("  %s\n", entry.Name())
			} else {
				fmt.Printf("  %s (%d bytes)\n", entry.Name(), info.Size())
			}
		}
		return
	}

	switch args[0] {
	case "path":
		fmt.Println(configDir)
	case "categories":
		catPath := filepath.Join(configDir, "categories.makima")
		data, err := os.ReadFile(catPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read categories: %v\n", err)
			return
		}
		fmt.Print(string(data))
	case "rules":
		rulesPath := filepath.Join(configDir, "rules.makima")
		data, err := os.ReadFile(rulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read rules: %v\n", err)
			return
		}
		fmt.Print(string(data))
	default:
		fmt.Fprintf(os.Stderr, "Unknown config subcommand: %s\n", args[0])
		fmt.Println("Usage: makima config [path|categories|rules]")
	}
}

func logCmd(args []string) {
	logPath := "/tmp/makima.log"
	
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Println("No log file found. Start daemon with: makima daemon start --verbose")
		return
	}

	// Show last N lines (default 50)
	n := 50
	if len(args) > 0 {
		if val, err := strconv.Atoi(args[0]); err == nil {
			n = val
		}
	}

	// Use tail command for efficiency
	cmd := exec.Command("tail", "-n", strconv.Itoa(n), logPath)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read log: %v\n", err)
		return
	}

	fmt.Print(string(output))
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home + "/.config/makima"
}

func getBrowserPortFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.config/BraveSoftware/Brave-Browser/DevToolsActivePort"
}

func getSocketPath() string {
	return "/tmp/makima.sock"
}

func newClient() (*cli.Client, error) {
	return cli.NewClient(getSocketPath())
}

var todoStore *todo.Store
var todoStoreMu sync.Mutex
var todoStoreErr error

func getTodoStore() *todo.Store {
	todoStoreMu.Lock()
	defer todoStoreMu.Unlock()
	if todoStore == nil {
		todoStore, todoStoreErr = todo.NewStore(getConfigDir() + "/todos.json")
		if todoStoreErr != nil {
			log.Error("failed to load todo store: %v", todoStoreErr)
		}
	}
	return todoStore
}

func conditionToString(cond dsl.Condition) string {
	switch c := cond.(type) {
	case *dsl.CategoryCondition:
		return "browser.category is " + c.Category
	case *dsl.URLCondition:
		return "browser.url matches " + c.Pattern
	case *dsl.TabTitleCondition:
		return "browser.tab.title matches " + c.Pattern
	case *dsl.DomainCondition:
		return "browser.domain matches " + c.Pattern
	case *dsl.AppCondition:
		return "app." + c.Name + " running"
	case *dsl.WindowClassCondition:
		return "window.class matches " + c.Pattern
	case *dsl.WorkspaceCountCondition:
		return "workspace.count " + c.Operator + " " + fmt.Sprintf("%d", c.Count)
	case *dsl.TimeOnSiteCondition:
		return "browser.time_on_site " + c.Operator + " " + c.Duration.String()
	default:
		return fmt.Sprintf("%T", cond)
	}
}
