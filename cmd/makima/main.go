package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: makima <command>\n\nCommands:\n  daemon    Start the background daemon\n  status    Show daemon status\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daemon":
		fmt.Println("daemon: not implemented yet")
	case "status":
		fmt.Println("status: not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
