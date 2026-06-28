package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type startupChoice int

const (
	startupChoiceOpen startupChoice = iota
	startupChoiceCreate
)

var startupChoices = []struct {
	title       string
	description string
}{
	{
		title:       "Open file",
		description: "Load JSON, JSONL, YAML, CSV, or TSV data.",
	},
	{
		title:       "Create dataset",
		description: "Generate scalar, list, or matrix data.",
	},
}

func (m *Model) updateStartup(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.String() == "esc" && m.clearTransientPopup() {
		return m, nil
	}
	if m.err != nil || m.notice != "" {
		return m, nil
	}

	switch key.String() {
	case "q":
		return m, tea.Quit
	case "?", "h":
		m.openHelp()
		return m, nil
	case "up", "k":
		m.moveStartupChoice(-1)
		return m, nil
	case "down", "j", "tab":
		m.moveStartupChoice(1)
		return m, nil
	case "o":
		m.startChoice = startupChoiceOpen
		return m, m.openFilePicker(true)
	case "c":
		m.startChoice = startupChoiceCreate
		return m, m.openCreateDataset()
	case "enter":
		if m.startChoice == startupChoiceCreate {
			return m, m.openCreateDataset()
		}
		return m, m.openFilePicker(true)
	default:
		return m, nil
	}
}

func (m *Model) moveStartupChoice(delta int) {
	next := int(m.startChoice) + delta
	if next < 0 {
		next = len(startupChoices) - 1
	}
	if next >= len(startupChoices) {
		next = 0
	}

	m.startChoice = startupChoice(next)
}

func (m *Model) startupView() string {
	return m.screenView(
		m.startupHeader(),
		m.startupContent(),
		m.startupFooter(),
	)
}

func (m *Model) startupHeader() string {
	return viewHeader(
		"dist-lab",
		statusLine(statusItem{label: "Start", value: "Open a file or create generated data"}),
	)
}

func (m *Model) startupContent() string {
	lines := make([]string, 0, len(startupChoices)*3)
	for i, choice := range startupChoices {
		prefix := " "
		if startupChoice(i) == m.startChoice {
			prefix = ">"
		}

		lines = append(lines,
			prefix+" "+titleStyle.Render(choice.title),
			"  "+choice.description,
			"",
		)
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func (m *Model) startupFooter() string {
	return helpFooter(
		keyHelp{key: "up/down", label: "choose"},
		keyHelp{key: "enter", label: "select"},
		keyHelp{key: "o", label: "open"},
		keyHelp{key: "c", label: "create"},
		keyHelp{key: "?", label: "help"},
		keyHelp{key: "q", label: "quit"},
	)
}
