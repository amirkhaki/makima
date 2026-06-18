package main

import (
	"fmt"
	"os"
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
		fmt.Println("daemon: start - not implemented yet")
	case "stop":
		fmt.Println("daemon: stop - not implemented yet")
	case "restart":
		fmt.Println("daemon: restart - not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "Unknown daemon subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func statusCmd(args []string) {
	fmt.Println("status: not implemented yet")
}

func ruleCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: makima rule <list|add|remove|enable|disable>")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		fmt.Println("rule: list - not implemented yet")
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

	switch args[0] {
	case "list":
		fmt.Println("category: list - not implemented yet")
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

	switch args[0] {
	case "list":
		fmt.Println("todo: list - not implemented yet")
	case "add":
		fmt.Println("todo: add - not implemented yet")
	case "done":
		fmt.Println("todo: done - not implemented yet")
	case "remove":
		fmt.Println("todo: remove - not implemented yet")
	case "tree":
		fmt.Println("todo: tree - not implemented yet")
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
