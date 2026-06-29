package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateAnalysis(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.analysisFilterActive && m.updateAnalysisFilter(msg) {
			return m, nil
		}

		switch msg.String() {
		case "p":
			m.changeState(viewPreview)
			return m, nil
		case "esc":
			if m.clearAnalysisContext() {
				return m, nil
			}
			m.changeState(viewPreview)
			return m, nil
		case "1":
			m.setAnalysisMode(analysisModeOverview)
			return m, nil
		case "2":
			m.setAnalysisMode(analysisModeMissing)
			return m, nil
		case "3":
			m.setAnalysisMode(analysisModeFields)
			return m, nil
		case "/":
			m.analysisMode = analysisModeFields
			m.analysisFilterActive = true
			m.refreshAnalysisAtSelectedField()
			return m, nil
		case "n":
			m.cycleAnalysisField(1)
			return m, nil
		case "N":
			m.cycleAnalysisField(-1)
			return m, nil
		case "enter":
			m.focusAnalysisField()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.analysis, cmd = m.analysis.Update(msg)
	return m, cmd
}

func (m *Model) analysisView() string {
	return m.screenView(
		m.analysisHeaderText(),
		m.analysis.View(),
		m.analysisFooterText(),
	)
}

func (m *Model) analysisHeaderText() string {
	lines := []string{
		m.fileInfoStatus(),
		statusLine(statusItem{label: "Path", value: m.selectedPath}),
		statusLine(
			statusItem{label: "Mode", value: m.analysisMode.label()},
			statusItem{label: "Values", value: m.valuesStatus()},
		),
	}

	if m.analysisFilterActive || m.analysisFilter != "" {
		lines = append(lines, statusLine(
			statusItem{label: "Filter", value: m.analysisFilterStatus()},
			statusItem{label: "Match", value: m.analysisMatchStatus()},
		))
	}

	if m.analysisMode == analysisModeFocus && m.analysisFocusedField != "" {
		lines = append(lines, statusLine(statusItem{label: "Focus", value: m.analysisFocusedField}))
	}

	return viewHeaderTitle(
		titleStyle.Render("Analysis")+" "+badge("CURRENT VALUES"),
		lines...,
	)
}

func (m *Model) analysisFooterText() string {
	if m.analysisFilterActive {
		return helpFooter(
			keyHelp{key: "type", label: "filter"},
			keyHelp{key: "enter", label: "focus"},
			keyHelp{key: "esc", label: "done"},
			keyHelp{key: "?", label: "help"},
			keyHelp{key: "q", label: "quit"},
		)
	}

	if m.analysisMode == analysisModeFocus {
		return helpFooter(
			keyHelp{key: "n/N", label: "next field"},
			keyHelp{key: "esc", label: "fields"},
			keyHelp{key: "p", label: "preview"},
			keyHelp{key: "?", label: "help"},
			keyHelp{key: "q", label: "quit"},
		)
	}

	if m.analysisMode == analysisModeFields {
		return helpFooter(
			keyHelp{key: "1/2/3", label: "tabs"},
			keyHelp{key: "/", label: "filter"},
			keyHelp{key: "n/N", label: "jump"},
			keyHelp{key: "enter", label: "focus"},
			keyHelp{key: "?", label: "help"},
			keyHelp{key: "q", label: "quit"},
		)
	}

	return helpFooter(
		keyHelp{key: "1/2/3", label: "tabs"},
		keyHelp{key: "up/down", label: "scroll"},
		keyHelp{key: "p/esc", label: "preview"},
		keyHelp{key: "?", label: "help"},
		keyHelp{key: "q", label: "quit"},
	)
}

func (m *Model) rebuildAnalysis() {
	m.clampAnalysisFieldIndex()
	content := analysisContentForState(m.values, m.analysisContentWidth(), m.currentAnalysisViewState())
	if m.analysis.Width == 0 {
		m.analysis = viewport.New(m.contentWidth(), defaultContentHeight)
	}

	m.analysis.SetContent(content)
	m.analysis.GotoTop()
}

func (m *Model) resizeAnalysis() {
	m.analysis.Width = m.contentWidth()
	m.analysis.Height = m.analysisContentHeight()
	if m.analysis.Height < minContentHeight {
		m.analysis.Height = minContentHeight
	}

	if m.state == viewAnalysis {
		m.analysis.SetContent(analysisContentForState(m.values, m.analysisContentWidth(), m.currentAnalysisViewState()))
	}
}

func (m *Model) analysisContentHeight() int {
	return m.childContentHeight(
		m.analysisHeaderText(),
		m.analysisFooterText(),
		0,
	)
}

func (m *Model) analysisContentWidth() int {
	width := m.contentWidth() - 4
	if width < 24 {
		return 24
	}
	return width
}

func (m *Model) currentAnalysisViewState() analysisViewState {
	return analysisViewState{
		mode:         m.analysisMode,
		filter:       m.analysisFilter,
		fieldIndex:   m.analysisFieldIndex,
		focusedField: m.analysisFocusedField,
	}
}

func (m *Model) resetAnalysisState() {
	m.analysisMode = analysisModeOverview
	m.analysisFilterActive = false
	m.analysisFilter = ""
	m.analysisFieldIndex = 0
	m.analysisFocusedField = ""
}

func (m *Model) refreshAnalysis() {
	m.resizeViews()
	m.rebuildAnalysis()
}

func (m *Model) refreshAnalysisAtSelectedField() {
	m.refreshAnalysis()
	m.scrollAnalysisToSelectedField()
}

func (m *Model) setAnalysisMode(mode analysisMode) {
	m.analysisMode = mode
	m.analysisFilterActive = false
	m.refreshAnalysis()
}

func (m *Model) updateAnalysisFilter(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "esc":
		if m.analysisFilter != "" {
			m.analysisFilter = ""
			m.analysisFieldIndex = 0
		} else {
			m.analysisFilterActive = false
		}
		m.refreshAnalysisAtSelectedField()
		return true
	case "enter":
		m.focusAnalysisField()
		return true
	case "backspace", "ctrl+h":
		m.analysisFilter = trimLastRune(m.analysisFilter)
		m.analysisFieldIndex = 0
		m.refreshAnalysisAtSelectedField()
		return true
	}

	if msg.Type == tea.KeyRunes && !msg.Alt {
		m.analysisFilter += string(msg.Runes)
		m.analysisFieldIndex = 0
		m.refreshAnalysisAtSelectedField()
		return true
	}

	return false
}

func (m *Model) clearAnalysisContext() bool {
	switch {
	case m.analysisMode == analysisModeFocus:
		m.analysisMode = analysisModeFields
	case m.analysisFilter != "":
		m.analysisFilter = ""
		m.analysisFieldIndex = 0
	default:
		return false
	}

	m.analysisFilterActive = false
	m.refreshAnalysisAtSelectedField()
	return true
}

func (m *Model) cycleAnalysisField(delta int) {
	matches := m.analysisFieldMatches()
	if len(matches) == 0 {
		m.analysisFieldIndex = 0
		m.analysisMode = analysisModeFields
		m.refreshAnalysisAtSelectedField()
		return
	}

	m.analysisFieldIndex = (m.analysisFieldIndex + delta) % len(matches)
	if m.analysisFieldIndex < 0 {
		m.analysisFieldIndex += len(matches)
	}

	if m.analysisMode == analysisModeFocus {
		m.analysisFocusedField = matches[m.analysisFieldIndex]
	} else {
		m.analysisMode = analysisModeFields
	}
	m.analysisFilterActive = false
	m.refreshAnalysisAtSelectedField()
}

func (m *Model) focusAnalysisField() {
	matches := m.analysisFieldMatches()
	if len(matches) == 0 {
		m.analysisMode = analysisModeFields
		m.analysisFilterActive = false
		m.refreshAnalysisAtSelectedField()
		return
	}

	m.clampAnalysisFieldIndex()
	matches = m.analysisFieldMatches()
	m.analysisFocusedField = matches[m.analysisFieldIndex]
	m.analysisMode = analysisModeFocus
	m.analysisFilterActive = false
	m.refreshAnalysis()
}

func (m *Model) clampAnalysisFieldIndex() {
	matches := m.analysisFieldMatches()
	if len(matches) == 0 {
		m.analysisFieldIndex = 0
		return
	}

	if m.analysisFieldIndex < 0 {
		m.analysisFieldIndex = 0
	}
	if m.analysisFieldIndex >= len(matches) {
		m.analysisFieldIndex = len(matches) - 1
	}
}

func (m *Model) analysisFieldMatches() []string {
	stats := analyzeValues(m.values)
	return analysisFieldMatches(stats.fields, m.analysisFilter)
}

func (m *Model) scrollAnalysisToSelectedField() {
	if m.analysisMode != analysisModeFields {
		return
	}

	stats := analyzeValues(m.values)
	offset, ok := analysisSelectedFieldLineOffset(stats, m.analysisContentWidth(), m.currentAnalysisViewState())
	if !ok {
		return
	}

	m.analysis.SetYOffset(offset)
}

func (m *Model) analysisFilterStatus() string {
	if m.analysisFilter == "" {
		if m.analysisFilterActive {
			return "typing"
		}
		return ""
	}

	return m.analysisFilter
}

func (m *Model) analysisMatchStatus() string {
	matches := m.analysisFieldMatches()
	if len(matches) == 0 {
		return "0"
	}

	index := m.analysisFieldIndex
	if index < 0 {
		index = 0
	}
	if index >= len(matches) {
		index = len(matches) - 1
	}

	return fmt.Sprintf("%d/%d", index+1, len(matches))
}

func trimLastRune(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	return string(runes[:len(runes)-1])
}

func analysisContent(values []any, width int) string {
	return analysisContentForState(values, width, analysisViewState{mode: analysisModeOverview})
}

func analysisContentForState(values []any, width int, state analysisViewState) string {
	stats := analyzeValues(values)
	switch state.mode {
	case analysisModeMissing:
		return strings.Join(missingAnalysisContent(stats, width), "\n")
	case analysisModeFields:
		return strings.Join(fieldsAnalysisContent(stats, width, state), "\n")
	case analysisModeFocus:
		return strings.Join(focusedFieldAnalysisContent(stats, width, state), "\n")
	default:
		return strings.Join(overviewAnalysisContent(values, stats, width), "\n")
	}
}

func overviewAnalysisContent(values []any, stats analysisStats, width int) []string {
	dateStats := analyzeDateValues(values)
	lines := analysisSummaryView(stats)

	if len(stats.numeric) > 0 {
		lines = append(lines, "")
		lines = append(lines, numericAnalysisView(stats.numeric, width)...)
	}

	if len(stats.categorical) > 0 {
		lines = append(lines, "")
		lines = append(lines, categoricalAnalysisView(stats.categorical, width)...)
	}

	if len(stats.booleans) > 0 {
		lines = append(lines, "")
		lines = append(lines, booleanAnalysisView(stats.booleans, width)...)
	}

	if dateStats.validCount() > 0 {
		lines = append(lines, "")
		lines = append(lines, dateAnalysisView(dateStats, width)...)
	}

	if len(stats.numeric) == 0 && len(stats.categorical) == 0 && len(stats.booleans) == 0 && dateStats.validCount() == 0 && len(stats.fields) > 0 {
		lines = append(lines, "", "No top-level scalar values. Use Fields for recursive object values.")
	}

	if len(stats.numeric) == 0 && len(stats.categorical) == 0 && len(stats.booleans) == 0 && len(stats.fields) == 0 {
		lines = append(lines, "", "No scalar values to analyze.")
	}

	return lines
}

func analysisSummaryView(stats analysisStats) []string {
	return []string{
		titleStyle.Render("Summary"),
		statusLine(statusItem{label: "Total", value: fmt.Sprint(stats.total)}),
		statusLine(
			statusItem{label: "Numeric", value: fmt.Sprint(len(stats.numeric))},
			statusItem{label: "Categories", value: fmt.Sprint(categoricalCount(stats.categorical))},
			statusItem{label: "Booleans", value: fmt.Sprint(booleanCount(stats.booleans))},
			statusItem{label: "Empty", value: fmt.Sprint(stats.empty)},
			statusItem{label: "Unsupported", value: fmt.Sprint(stats.unsupported)},
			statusItem{label: "Fields", value: fmt.Sprint(len(stats.fields))},
		),
	}
}

func missingAnalysisContent(stats analysisStats, width int) []string {
	missing := missingFields(stats.fields)
	if len(missing) == 0 {
		return []string{
			titleStyle.Render("Missing Data"),
			"No missing fields found.",
		}
	}

	return missingAnalysisView(missing, width)
}

func fieldsAnalysisContent(stats analysisStats, width int, state analysisViewState) []string {
	paths := analysisFieldMatches(stats.fields, state.filter)
	lines := []string{
		titleStyle.Render("Fields"),
	}

	if state.filter != "" {
		lines = append(lines, statusLine(
			statusItem{label: "Filter", value: state.filter},
			statusItem{label: "Matches", value: fmt.Sprint(len(paths))},
		))
	}

	if len(stats.fields) == 0 {
		return append(lines, "No recursive fields found.")
	}
	if len(paths) == 0 {
		return append(lines, "No matching fields.")
	}

	return append(lines, fieldsAnalysisViewForPaths(stats.fields, paths, width, state.fieldIndex)...)
}

func focusedFieldAnalysisContent(stats analysisStats, width int, state analysisViewState) []string {
	if state.focusedField == "" {
		return []string{
			titleStyle.Render("Focused Field"),
			"No focused field. Press / to filter fields, then enter to focus a match.",
		}
	}

	field := stats.fields[state.focusedField]
	if field == nil {
		return []string{
			titleStyle.Render("Focused Field"),
			fmt.Sprintf("Field %q is no longer available.", state.focusedField),
		}
	}

	lines := []string{
		titleStyle.Render("Focused Field"),
	}
	return append(lines, analysisFieldView(state.focusedField, field, width, false)...)
}

func analysisFieldMatches(fields map[string]*analysisStats, filter string) []string {
	paths := sortedAnalysisFieldPaths(fields)
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return paths
	}

	matches := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.Contains(strings.ToLower(path), filter) {
			matches = append(matches, path)
		}
	}

	return matches
}

func analysisSelectedFieldLineOffset(stats analysisStats, width int, state analysisViewState) (int, bool) {
	paths := analysisFieldMatches(stats.fields, state.filter)
	if len(paths) == 0 {
		return 0, false
	}

	index := state.fieldIndex
	if index < 0 {
		index = 0
	}
	if index >= len(paths) {
		index = len(paths) - 1
	}

	lines := []string{titleStyle.Render("Fields")}
	if state.filter != "" {
		lines = append(lines, statusLine(
			statusItem{label: "Filter", value: state.filter},
			statusItem{label: "Matches", value: fmt.Sprint(len(paths))},
		))
	}

	for i, path := range paths {
		lines = append(lines, "")
		if i == index {
			return len(lines), true
		}
		lines = append(lines, analysisFieldView(path, stats.fields[path], width, false)...)
	}

	return 0, false
}
