package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/poiley/beady/internal/app"
	"github.com/poiley/beady/internal/bd"
	"github.com/poiley/beady/internal/selfupdate"
)

// Set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	// Handle --version / --help / update / check
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("bdy %s (commit: %s, built: %s)\n", Version, Commit, Date)
			os.Exit(0)
		case "--help", "-h", "help":
			fmt.Println("bdy - a k9s-style TUI for beads issue tracking")
			fmt.Printf("Version: %s\n\n", Version)
			fmt.Println("Usage: bdy [directory]")
			fmt.Println()
			fmt.Println("Run bdy in a directory with beads initialized (bd init).")
			fmt.Println("If no directory is given, uses the current working directory.")
			fmt.Println()
			fmt.Println("Commands:")
			fmt.Println("  update             Check for and install the latest version")
			fmt.Println("  version            Show version info")
			fmt.Println("  check              Verify bd CLI is available and beads is initialized")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --version, -v      Show version")
			fmt.Println("  --help, -h         Show this help")
			fmt.Println("  --check            Same as 'check' command")
			os.Exit(0)
		case "update":
			if err := selfupdate.Update(Version); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "--check", "check":
			workDir, _ := os.Getwd()
			client := bd.NewClient(workDir)
			if err := client.CheckInit(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("OK: bd CLI found and beads is initialized.")
			os.Exit(0)
		}
	}

	// Determine working directory
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		workDir = os.Args[1]
	}

	// Check bd is available
	client := bd.NewClient(workDir)
	if err := client.CheckInit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Start TUI
	model := app.New(workDir)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
