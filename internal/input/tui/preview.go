package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const valuesHeading = "Values"

type previewMode int

const (
	previewModeText previewMode = iota
	previewModeValues
)

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
		case "g":
			if m.previewMode == previewModeText {
				m.preview.GotoTop()
				return m, nil
			}
		case "G":
			if m.previewMode == previewModeText {
				m.preview.GotoBottom()
				return m, nil
			}
		case "v":
			m.togglePreviewMode()
			return m, nil
		case "d":
			if m.previewMode == previewModeValues {
				m.deleteSelectedValue()
				return m, nil
			}
		case "r":
			if m.previewMode == previewModeValues {
				m.restoreValues()
				return m, nil
			}
		case "x":
			return m, m.openExportPrompt()
		case "i":
			m.rebuildAnalysis()
			m.changeState(viewAnalysis)
			return m, nil
		}
	}

	if m.previewMode == previewModeValues {
		return m.updateValueList(msg)
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
	if m.previewMode == previewModeValues {
		return m.valueListContent()
	}

	return fmt.Sprintf("%s%s", valuesHeader(), m.preview.View())
}

func (m *Model) previewHeaderText() string {
	return viewHeaderTitle(
		titleStyle.Render("Preview")+" "+m.previewModeBadge(),
		m.fileInfoStatus(),
		statusLine(statusItem{label: "Path", value: m.selectedPath}),
		statusLine(statusItem{label: "Values", value: m.valuesStatus()}),
	)
}

func (m *Model) previewFooterText() string {
	filterAction := keyHelp{key: "e", label: "filter nil/empty"}
	if m.valuesFiltered {
		filterAction = keyHelp{key: "e", label: "show raw"}
	}

	if m.previewMode == previewModeValues {
		return helpFooter(
			keyHelp{key: "up/down", label: "move"},
			keyHelp{key: "d", label: "delete"},
			keyHelp{key: "r", label: "restore"},
			keyHelp{key: "v", label: "text"},
			filterAction,
			keyHelp{key: "i", label: "analysis"},
			keyHelp{key: "x", label: "export"},
			keyHelp{key: "f", label: "change field"},
			keyHelp{key: "a", label: "add file"},
			keyHelp{key: "o", label: "new file"},
			keyHelp{key: "q", label: "quit"},
		)
	}

	return helpFooter(
		keyHelp{key: "up/down", label: "scroll"},
		keyHelp{key: "pgup/pgdn", label: "page"},
		keyHelp{key: "g/G", label: "top/bottom"},
		keyHelp{key: "v", label: "edit values"},
		filterAction,
		keyHelp{key: "i", label: "analysis"},
		keyHelp{key: "x", label: "export"},
		keyHelp{key: "f", label: "change field"},
		keyHelp{key: "a", label: "add file"},
		keyHelp{key: "o", label: "new file"},
		keyHelp{key: "q", label: "quit"},
	)
}

func (m *Model) previewModeBadge() string {
	if m.previewMode == previewModeValues {
		return badge("VALUES")
	}

	return badge("TEXT")
}

func (m *Model) resizePreview() {
	m.preview.Width = m.contentWidth()
	m.preview.Height = m.previewContentHeight() - valuesHeaderHeight()
	if m.preview.Height < minContentHeight {
		m.preview.Height = minContentHeight
	}
	m.resizeValueList()
}

func (m *Model) previewContentHeight() int {
	return m.childContentHeight(
		m.previewHeaderText(),
		m.previewFooterText(),
		0,
	)
}

func valuesHeader() string {
	return valuesHeading + "\n"
}

func valuesHeaderHeight() int {
	return strings.Count(valuesHeader(), "\n")
}

func (m *Model) setValues(values []any) {
	m.rawValues = cloneValues(values)
	m.values = cloneValues(values)
	m.valuesFiltered = false
	m.previewMode = previewModeText
	m.rebuildValueList(0)
}

func (m *Model) clearValues() {
	m.rawValues = nil
	m.values = nil
	m.valuesFiltered = false
	m.previewMode = previewModeText
	m.valueList = valueListModel{}
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

	m.rebuildValueList(0)
	m.rebuildAnalysis()
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

	if len(m.values) != len(m.rawValues) {
		return fmt.Sprintf("%d current / %d raw (edited)", len(m.values), len(m.rawValues))
	}

	return fmt.Sprintf("%d raw", len(m.rawValues))
}

func (m *Model) togglePreviewMode() {
	if m.previewMode == previewModeText {
		m.previewMode = previewModeValues
		m.rebuildValueList(m.selectedValueIndex())
	} else {
		m.previewMode = previewModeText
		m.renderValues()
	}

	m.notice = ""
	m.resizeViews()
}

func (m *Model) updateValueList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.valueList, cmd = m.valueList.Update(msg)
	return m, cmd
}

func (m *Model) selectedValueIndex() int {
	index, ok := m.valueList.SelectedIndex()
	if !ok {
		return 0
	}

	return index
}

func (m *Model) deleteSelectedValue() {
	index, ok := m.valueList.SelectedIndex()
	if !ok || index < 0 || index >= len(m.values) {
		return
	}

	m.values = append(m.values[:index], m.values[index+1:]...)
	m.notice = ""
	m.rebuildValueList(index)
	m.renderValues()
	m.rebuildAnalysis()
	m.resizeViews()
}

func (m *Model) restoreValues() {
	if m.valuesFiltered {
		m.values = filterEmptyValues(m.rawValues)
	} else {
		m.values = cloneValues(m.rawValues)
	}

	m.notice = ""
	m.rebuildValueList(0)
	m.renderValues()
	m.rebuildAnalysis()
	m.resizeViews()
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
		if containsEmptyValue(value) {
			continue
		}

		filtered = append(filtered, cloneValue(value))
	}

	return filtered
}

func containsEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		if len(v) == 0 {
			return true
		}
		for _, item := range v {
			if containsEmptyValue(item) {
				return true
			}
		}

		return false
	case map[string]any:
		if len(v) == 0 {
			return true
		}
		for _, item := range v {
			if containsEmptyValue(item) {
				return true
			}
		}

		return false
	default:
		return false
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
