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
	if m.export.active {
		return m.updateExportPrompt(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			m.toggleEmptyValueFilter()
			return m, nil
		case "x":
			return m, m.openExportPrompt()
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

	return helpFooter("up/down scroll", "pgup/pgdn page", filterAction, "x export", "f change field", "a add file", "o new file")
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
	for i, value := range values {
		cloned[i] = cloneValue(value)
	}
	return cloned
}

func cloneValue(value any) any {
	switch v := value.(type) {
	case []any:
		cloned := make([]any, len(v))
		for i, item := range v {
			cloned[i] = cloneValue(item)
		}

		return cloned
	case map[string]any:
		cloned := make(map[string]any, len(v))
		for key, item := range v {
			cloned[key] = cloneValue(item)
		}

		return cloned
	default:
		return value
	}
}

func filterEmptyValues(values []any) []any {
	filtered := make([]any, 0, len(values))
	for _, value := range values {
		cleaned, empty := cleanEmptyValue(value)
		if empty {
			continue
		}

		filtered = append(filtered, cleaned)
	}

	return filtered
}

func cleanEmptyValue(value any) (any, bool) {
	switch v := value.(type) {
	case nil:
		return nil, true
	case string:
		return v, strings.TrimSpace(v) == ""
	case []any:
		cleaned := make([]any, 0, len(v))
		for _, item := range v {
			cleanedItem, empty := cleanEmptyValue(item)
			if empty {
				continue
			}

			cleaned = append(cleaned, cleanedItem)
		}

		return cleaned, len(cleaned) == 0
	case map[string]any:
		cleaned := make(map[string]any, len(v))
		for key, item := range v {
			cleanedItem, empty := cleanEmptyValue(item)
			if empty {
				continue
			}

			cleaned[key] = cleanedItem
		}

		return cleaned, len(cleaned) == 0
	default:
		return value, false
	}
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
