package main

import (
	"dist-lab/internal/version"
	"fmt"
	"log"
	"os"

	"dist-lab/internal/input/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("dist-lab %s (%s, %s)\n", version.Version, version.Commit, version.Date)
			return
		case "--help", "-h":
			fmt.Println("dist-lab - terminal UI for exploring structured data and shaping distributions into datasets")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  dist-lab")
			fmt.Println("  dist-lab --version")
			return
		}
	}

	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
