package main

import (
	"flag"
	"fmt"
	"log"

	"dist-lab/internal/input/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	showExports := flag.Bool("exports", false, "print the exports directory and exit")
	flag.Parse()

	if *showExports {
		dir, err := tui.EnsureExportDir()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(dir)
		return
	}

	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
