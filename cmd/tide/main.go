package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"tide/internal/app"
)

var version = "0.1.0"

func main() {
	versionFlag := flag.Bool("version", false, "Print tide version and exit")
	vFlag := flag.Bool("v", false, "Print tide version and exit")
	flag.Parse()

	if *versionFlag || *vFlag {
		fmt.Printf("tide version %s\n", version)
		os.Exit(0)
	}

	startPath := "."
	args := flag.Args()
	if len(args) > 0 {
		startPath = args[0]
	}

	p := tea.NewProgram(
		app.InitialModel(startPath),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running tide: %v\n", err)
		os.Exit(1)
	}
}
