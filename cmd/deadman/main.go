package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if os.Getenv("ENV") == "" {
		godotenv.Load()
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "serve":
		serveCmd(args)
	case "add":
		addCmd(args)
	case "remove", "rm":
		removeCmd(args)
	case "set":
		setCmd(args)
	case "list", "ls":
		listCmd(args)
	case "url":
		urlCmd(args)
	case "-V", "-version", "--version", "version":
		versionCmd()
	case "help", "-h", "-help", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("deadman - heartbeat monitoring server")
	fmt.Println()
	fmt.Println("Usage: deadman <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  serve             Start the HTTP server and check loop")
	fmt.Println("  add <name>        Add a monitored service")
	fmt.Println("  set <name>        Update a monitored service")
	fmt.Println("  remove <name>     Remove a monitored service")
	fmt.Println("  list              List services and their status")
	fmt.Println("  url <name>        Print a service's check-in URL")
	fmt.Println("  version           Print version")
	fmt.Println("  help              Show this help")
	fmt.Println()
	fmt.Println("Use 'deadman <command> -help' for command-specific help.")
}

func versionCmd() {
	fmt.Printf("deadman %s\n", Version)
	fmt.Printf("Built: %s\n", BuildTime)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// defaultConfigPath is ~/.config/deadman/services.csv, falling back to the
// current directory if the home directory can't be determined.
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./services.csv"
	}
	return filepath.Join(home, ".config", "deadman", "services.csv")
}

// defaultStatePath is ~/.local/deadman/state.json, falling back to the
// current directory if the home directory can't be determined.
func defaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./state.json"
	}
	return filepath.Join(home, ".local", "deadman", "state.json")
}
