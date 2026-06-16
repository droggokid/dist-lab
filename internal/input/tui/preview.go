package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const valuesHeading = "Values"

func (m *Model) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m *Model) previewView() string {
	return m.screenView(
		m.previewHeaderText(),
		m.previewContent(),
		m.previewFooterText(),
	)
}

func (m *Model) previewContent() string {
	return fmt.Sprintf("%s%s", valuesHeader(), m.preview.View())
}

func (m *Model) previewHeaderText() string {
	return fmt.Sprintf("Preview\n%s\nPath: %s", m.fileInfoStatus(), m.selectedPath)
}

func (m *Model) previewFooterText() string {
	return helpFooter("up/down scroll", "pgup/pgdn page", "f change field", "o change file")
}

func (m *Model) resizePreview() {
	m.preview.Width = m.contentWidth()
	m.preview.Height = m.childContentHeight(
		m.previewHeaderText(),
		m.previewFooterText(),
		lipgloss.Height(valuesHeader()),
	)
}

func valuesHeader() string {
	return valuesHeading + "\n"
}

func formatValues(values []any) string {
	if len(values) == 0 {
		return "    [none]"
	}

	var b strings.Builder
	for i, value := range values {
		rendered := formatValue(value)
		lines := strings.Split(rendered, "\n")

		_, err := fmt.Fprintf(&b, "    %d. %s", i+1, lines[0])
		if err != nil {
			return ""
		}
		for _, line := range lines[1:] {
			b.WriteString("\n       ")
			b.WriteString(line)
		}

		if i < len(values)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func formatValue(value any) string {
	rendered, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}

	return string(rendered)
}
