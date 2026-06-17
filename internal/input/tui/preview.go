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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			m.toggleEmptyValueFilter()
			return m, nil
		case "s":
			m.exportPreviewValues(exportFormatJSON)
			return m, nil
		case "c":
			m.exportPreviewValues(exportFormatCSV)
			return m, nil
		}
	}

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
	return fmt.Sprintf("Preview\n%s\nPath: %s\nValues: %s", m.fileInfoStatus(), m.selectedPath, m.valuesStatus())
}

func (m *Model) previewFooterText() string {
	filterAction := "e filter nil/empty"
	if m.valuesFiltered {
		filterAction = "e show raw"
	}

	return helpFooter("up/down scroll", "pgup/pgdn page", filterAction, "s save json", "c save csv", "f change field", "a add file", "o new file")
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

func (m *Model) setValues(values []any) {
	m.rawValues = cloneValues(values)
	m.values = cloneValues(values)
	m.valuesFiltered = false
}

func (m *Model) clearValues() {
	m.rawValues = nil
	m.values = nil
	m.valuesFiltered = false
	m.preview.SetContent("")
	m.preview.GotoTop()
}

func (m *Model) toggleEmptyValueFilter() {
	if len(m.rawValues) == 0 {
		return
	}

	m.valuesFiltered = !m.valuesFiltered
	m.notice = ""
	if m.valuesFiltered {
		m.values = filterEmptyValues(m.rawValues)
	} else {
		m.values = cloneValues(m.rawValues)
	}

	m.resizeViews()
	m.renderValues()
}

func (m *Model) renderValues() {
	m.preview.SetContent(formatValues(m.values))
	m.preview.GotoTop()
}

func (m *Model) valuesStatus() string {
	if m.valuesFiltered {
		return fmt.Sprintf("%d shown / %d raw (nil/empty filtered)", len(m.values), len(m.rawValues))
	}

	return fmt.Sprintf("%d raw", len(m.rawValues))
}

func cloneValues(values []any) []any {
	cloned := make([]any, len(values))
	copy(cloned, values)
	return cloned
}

func filterEmptyValues(values []any) []any {
	filtered := make([]any, 0, len(values))
	for _, value := range values {
		if isEmptyValue(value) {
			continue
		}

		filtered = append(filtered, value)
	}

	return filtered
}

func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

func (m *Model) exportPreviewValues(format exportFormat) {
	path, err := m.exportValues(format)
	if err != nil {
		m.setError(err)
		return
	}

	m.setNotice(path)
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
