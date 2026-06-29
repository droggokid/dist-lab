package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type exportPromptModel struct {
	active bool
	format exportFormat
	input  textinput.Model
	err    string
}

func (m *Model) openExportPrompt() tea.Cmd {
	input := textinput.New()
	input.Placeholder = "path/to/export"
	input.SetValue(m.defaultExportPath(exportFormatJSON))
	input.Width = m.exportPromptInputWidth()

	cmd := input.Focus()

	m.export = exportPromptModel{
		active: true,
		format: exportFormatJSON,
		input:  input,
	}
	m.err = nil
	m.notice = ""
	m.resizeViews()

	return cmd
}

func (m *Model) updateExportPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.closeExportPrompt()
			m.resizeViews()
			return m, nil
		case "tab":
			m.toggleExportFormat()
			m.resizeViews()
			return m, nil
		case "enter":
			m.saveExportPrompt()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.export.input, cmd = m.export.input.Update(msg)
	return m, cmd
}

func (m *Model) closeExportPrompt() {
	m.export = exportPromptModel{}
}

func (m *Model) resizeExportPrompt() {
	if !m.export.active {
		return
	}

	m.export.input.Width = m.exportPromptInputWidth()
}

func (m *Model) exportPromptInputWidth() int {
	width := m.borderedBlockWidth() - 4
	if width < 20 {
		return 20
	}

	return width
}

func (m *Model) toggleExportFormat() {
	m.export.err = ""
	oldFormat := m.export.format
	m.export.format = nextExportFormat(m.export.format)

	m.export.input.SetValue(swapExportExtension(m.export.input.Value(), oldFormat, m.export.format))
}

func (m *Model) saveExportPrompt() {
	path, err := m.exportValues(m.export.input.Value(), m.export.format)
	if err != nil {
		m.export.err = err.Error()
		m.resizeViews()
		return
	}

	m.closeExportPrompt()
	m.setNotice(path)
}

func (m *Model) exportPopup() string {
	lines := []string{
		titleStyle.Render("Export values"),
		statusLine(
			statusItem{label: "Format", value: m.exportFormatLabel()},
			statusItem{label: "Values", value: fmt.Sprint(len(m.values))},
		),
		"",
		labelStyle.Render("Path"),
		m.export.input.View(),
		"",
		helpFooter(
			keyHelp{key: "enter", label: "save"},
			keyHelp{key: "tab", label: "format"},
			keyHelp{key: "esc", label: "cancel"},
		),
	}

	if m.export.err != "" {
		lines = append(lines, "", errorTitleStyle.Render("Error"), m.export.err)
	}

	return m.popupView(strings.Join(lines, "\n"))
}

func (m *Model) exportFormatLabel() string {
	return strings.ToUpper(string(m.export.format))
}

func (m *Model) defaultExportPath(format exportFormat) string {
	name := sanitizeExportName(m.selectedPath)
	if name == "" {
		name = "values"
	}

	state := "raw"
	if m.valuesFiltered {
		state = "filtered"
	}

	return fmt.Sprintf("%s-%s.%s", name, state, exportFormatExtension(format))
}

func (m *Model) exportValues(path string, format exportFormat) (string, error) {
	path, err := normalizeExportPath(path, format)
	if err != nil {
		return "", err
	}

	return writeValuesToPath(path, format, m.values)
}
