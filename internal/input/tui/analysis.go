package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	analysisTopValueLimit = 8
	analysisOutlierLimit  = 8
	analysisBarMaxWidth   = 32
	analysisBucketCount   = 10
	analysisDiscreteLimit = 32
	highCardinalityRate   = 0.8
)

type analysisMode int

const (
	analysisModeOverview analysisMode = iota
	analysisModeMissing
	analysisModeFields
	analysisModeFocus
)

type analysisViewState struct {
	mode         analysisMode
	filter       string
	fieldIndex   int
	focusedField string
}

type analysisStats struct {
	total       int
	empty       int
	numeric     []float64
	categorical map[string]int
	booleans    map[bool]int
	unsupported int
	fields      map[string]*analysisStats
}

type numericSummary struct {
	count    int
	min      float64
	q1       float64
	max      float64
	mean     float64
	median   float64
	q3       float64
	iqr      float64
	stddev   float64
	outliers []float64
	buckets  []histogramBucket
}

type histogramBucket struct {
	start float64
	end   float64
	count int
}

type valueFrequency struct {
	value string
	count int
}

type missingField struct {
	path  string
	total int
	empty int
	rate  float64
}

type dateAnalysis struct {
	total   int
	valid   []time.Time
	missing int
	invalid int
	years   map[int]int
	months  map[time.Month]int
}

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

func (mode analysisMode) label() string {
	switch mode {
	case analysisModeOverview:
		return "Overview"
	case analysisModeMissing:
		return "Missing"
	case analysisModeFields:
		return "Fields"
	case analysisModeFocus:
		return "Focus"
	default:
		return "Overview"
	}
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

func analyzeDateValues(values []any) dateAnalysis {
	analysis := dateAnalysis{
		total:  len(values),
		years:  make(map[int]int),
		months: make(map[time.Month]int),
	}

	for _, value := range values {
		date, ok, missing := dateFromValue(value)
		if missing {
			analysis.missing++
			continue
		}
		if !ok {
			analysis.invalid++
			continue
		}

		analysis.valid = append(analysis.valid, date)
		analysis.years[date.Year()]++
		analysis.months[date.Month()]++
	}

	sort.Slice(analysis.valid, func(i, j int) bool {
		return analysis.valid[i].Before(analysis.valid[j])
	})

	return analysis
}

func (d dateAnalysis) validCount() int {
	return len(d.valid)
}

func dateFromValue(value any) (time.Time, bool, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return time.Time{}, false, false
	}

	day, dayOK, dayMissing := intField(object, "day")
	month, monthOK, monthMissing := intField(object, "month")
	year, yearOK, yearMissing := intField(object, "year")
	if dayMissing || monthMissing || yearMissing {
		return time.Time{}, false, true
	}
	if !dayOK || !monthOK || !yearOK {
		return time.Time{}, false, false
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return time.Time{}, false, false
	}

	return date, true, false
}

func intField(object map[string]any, key string) (int, bool, bool) {
	value, ok := object[key]
	if !ok || value == nil {
		return 0, false, true
	}

	switch v := value.(type) {
	case int:
		return v, true, false
	case int8:
		return int(v), true, false
	case int16:
		return int(v), true, false
	case int32:
		return int(v), true, false
	case int64:
		return int(v), true, false
	case uint:
		return int(v), true, false
	case uint8:
		return int(v), true, false
	case uint16:
		return int(v), true, false
	case uint32:
		return int(v), true, false
	case uint64:
		return int(v), true, false
	case float64:
		if math.Trunc(v) != v {
			return 0, false, false
		}
		return int(v), true, false
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false, true
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false, false
		}
		return parsed, true, false
	default:
		return 0, false, false
	}
}

func dateAnalysisView(analysis dateAnalysis, width int) []string {
	first := analysis.valid[0]
	last := analysis.valid[len(analysis.valid)-1]

	lines := []string{
		titleStyle.Render("Date"),
		statusLine(
			statusItem{label: "Valid", value: fmt.Sprint(analysis.validCount())},
			statusItem{label: "Missing", value: fmt.Sprint(analysis.missing)},
			statusItem{label: "Invalid", value: fmt.Sprint(analysis.invalid)},
		),
		statusLine(
			statusItem{label: "First", value: first.Format("2006-01-02")},
			statusItem{label: "Last", value: last.Format("2006-01-02")},
		),
		"",
		labelStyle.Render("Years"),
	}

	yearFrequencies := yearFrequencies(analysis.years)
	maxYearCount := maxFrequencyCount(yearFrequencies)
	for _, frequency := range yearFrequencies {
		lines = append(lines, frequencyBar(frequency.value, frequency.count, maxYearCount, analysis.validCount(), width))
	}

	lines = append(lines, "", labelStyle.Render("Months"))
	monthFrequencies := monthFrequencies(analysis.months)
	maxMonthCount := maxFrequencyCount(monthFrequencies)
	for _, frequency := range monthFrequencies {
		lines = append(lines, frequencyBar(frequency.value, frequency.count, maxMonthCount, analysis.validCount(), width))
	}

	return lines
}

func yearFrequencies(values map[int]int) []valueFrequency {
	years := make([]int, 0, len(values))
	for year := range values {
		years = append(years, year)
	}
	sort.Ints(years)

	frequencies := make([]valueFrequency, 0, len(years))
	for _, year := range years {
		frequencies = append(frequencies, valueFrequency{
			value: fmt.Sprint(year),
			count: values[year],
		})
	}
	return frequencies
}

func monthFrequencies(values map[time.Month]int) []valueFrequency {
	months := make([]int, 0, len(values))
	for month := range values {
		months = append(months, int(month))
	}
	sort.Ints(months)

	frequencies := make([]valueFrequency, 0, len(months))
	for _, month := range months {
		dateMonth := time.Month(month)
		frequencies = append(frequencies, valueFrequency{
			value: dateMonth.String(),
			count: values[dateMonth],
		})
	}
	return frequencies
}

func analyzeValues(values []any) analysisStats {
	stats := analysisStats{
		total:       len(values),
		categorical: make(map[string]int),
		booleans:    make(map[bool]int),
		fields:      make(map[string]*analysisStats),
	}

	for _, value := range values {
		if !collectAnalysisFields(value, "", &stats) {
			classifyAnalysisScalar(value, &stats)
		}
	}

	sortAnalysisStats(&stats)
	return stats
}

func newAnalysisStats() *analysisStats {
	return &analysisStats{
		categorical: make(map[string]int),
		booleans:    make(map[bool]int),
		fields:      make(map[string]*analysisStats),
	}
}

func sortAnalysisStats(stats *analysisStats) {
	sort.Float64s(stats.numeric)
	for _, field := range stats.fields {
		sortAnalysisStats(field)
	}
}

func collectAnalysisFields(value any, path string, stats *analysisStats) bool {
	switch v := value.(type) {
	case map[string]any:
		var found bool
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			if collectAnalysisFields(v[key], joinAnalysisPath(path, key), stats) {
				found = true
			}
		}
		return found
	case []any:
		var found bool
		childPath := path + "[]"
		if path == "" {
			childPath = "[]"
		}

		for _, item := range v {
			if collectAnalysisFields(item, childPath, stats) {
				found = true
			}
		}
		return found
	default:
		if path == "" {
			return false
		}

		field := analysisField(stats, path)
		field.total++
		classifyAnalysisScalar(value, field)
		return true
	}
}

func joinAnalysisPath(parent string, child string) string {
	if parent == "" {
		return child
	}

	return parent + "." + child
}

func analysisField(stats *analysisStats, path string) *analysisStats {
	field, ok := stats.fields[path]
	if ok {
		return field
	}

	field = newAnalysisStats()
	stats.fields[path] = field
	return field
}

func classifyAnalysisScalar(value any, stats *analysisStats) {
	switch v := value.(type) {
	case nil:
		stats.empty++
	case string:
		value := strings.TrimSpace(v)
		if value == "" {
			stats.empty++
			return
		}
		if number, ok := parseAnalysisNumber(value); ok {
			stats.numeric = append(stats.numeric, number)
			return
		}
		stats.categorical[value]++
	case bool:
		stats.booleans[v]++
	case int:
		stats.numeric = append(stats.numeric, float64(v))
	case int8:
		stats.numeric = append(stats.numeric, float64(v))
	case int16:
		stats.numeric = append(stats.numeric, float64(v))
	case int32:
		stats.numeric = append(stats.numeric, float64(v))
	case int64:
		stats.numeric = append(stats.numeric, float64(v))
	case uint:
		stats.numeric = append(stats.numeric, float64(v))
	case uint8:
		stats.numeric = append(stats.numeric, float64(v))
	case uint16:
		stats.numeric = append(stats.numeric, float64(v))
	case uint32:
		stats.numeric = append(stats.numeric, float64(v))
	case uint64:
		stats.numeric = append(stats.numeric, float64(v))
	case float32:
		stats.numeric = append(stats.numeric, float64(v))
	case float64:
		stats.numeric = append(stats.numeric, v)
	default:
		stats.unsupported++
	}
}

func parseAnalysisNumber(value string) (float64, bool) {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}

	return number, true
}

func numericAnalysisView(values []float64, width int) []string {
	summary := summarizeNumeric(values)
	lines := []string{
		titleStyle.Render("Numeric"),
		statusLine(
			statusItem{label: "Count", value: fmt.Sprint(summary.count)},
			statusItem{label: "Min", value: formatAnalysisNumber(summary.min)},
			statusItem{label: "Max", value: formatAnalysisNumber(summary.max)},
		),
		statusLine(
			statusItem{label: "Mean", value: formatAnalysisNumber(summary.mean)},
			statusItem{label: "Median", value: formatAnalysisNumber(summary.median)},
			statusItem{label: "Stddev", value: formatAnalysisNumber(summary.stddev)},
			statusItem{label: "Outliers", value: fmt.Sprint(len(summary.outliers))},
		),
		"",
	}

	if len(summary.outliers) > 0 {
		lines = append(lines, outlierAnalysisView(summary.outliers, width)...)
		lines = append(lines, "")
	}

	lines = append(lines, labelStyle.Render("Distribution"))
	if discreteNumericDistribution(values) {
		frequencies := numericValueFrequencies(values)
		maxCount := maxFrequencyCount(frequencies)
		for _, frequency := range frequencies {
			lines = append(lines, frequencyBar(frequency.value, frequency.count, maxCount, len(values), width))
		}
		return lines
	}

	maxCount := maxBucketCount(summary.buckets)
	for _, bucket := range summary.buckets {
		label := fmt.Sprintf("%s..%s", formatAnalysisNumber(bucket.start), formatAnalysisNumber(bucket.end))
		lines = append(lines, frequencyBar(label, bucket.count, maxCount, len(values), width))
	}

	return lines
}

func summarizeNumeric(values []float64) numericSummary {
	summary := numericSummary{
		count: len(values),
		min:   values[0],
		max:   values[len(values)-1],
	}

	var sum float64
	for _, value := range values {
		sum += value
	}
	summary.mean = sum / float64(len(values))

	middle := len(values) / 2
	if len(values)%2 == 0 {
		summary.median = (values[middle-1] + values[middle]) / 2
		summary.q1 = medianSorted(values[:middle])
		summary.q3 = medianSorted(values[middle:])
	} else {
		summary.median = values[middle]
		summary.q1 = medianSorted(values[:middle])
		summary.q3 = medianSorted(values[middle+1:])
	}
	if len(values) == 1 {
		summary.q1 = values[0]
		summary.q3 = values[0]
	}
	summary.iqr = summary.q3 - summary.q1

	if len(values) > 1 {
		var squared float64
		for _, value := range values {
			delta := value - summary.mean
			squared += delta * delta
		}
		summary.stddev = math.Sqrt(squared / float64(len(values)-1))
	}

	summary.outliers = outlierValues(values, summary.q1, summary.q3, summary.iqr)
	summary.buckets = histogram(values, analysisBucketCount)
	return summary
}

func medianSorted(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2
	}

	return values[middle]
}

func outlierCount(values []float64, q1 float64, q3 float64, iqr float64) int {
	return len(outlierValues(values, q1, q3, iqr))
}

func outlierValues(values []float64, q1 float64, q3 float64, iqr float64) []float64 {
	if iqr == 0 {
		return nil
	}

	lower := q1 - 1.5*iqr
	upper := q3 + 1.5*iqr
	outliers := []float64{}
	for _, value := range values {
		if value < lower || value > upper {
			outliers = append(outliers, value)
		}
	}

	return outliers
}

func outlierAnalysisView(values []float64, width int) []string {
	frequencies, other := numericFrequencies(values, analysisOutlierLimit)
	lines := []string{
		labelStyle.Render("Outlier Values"),
	}

	maxCount := maxFrequencyCount(frequencies)
	if other.count > maxCount {
		maxCount = other.count
	}
	for _, frequency := range frequencies {
		lines = append(lines, frequencyBar(frequency.value, frequency.count, maxCount, len(values), width))
	}
	if other.count > 0 {
		lines = append(lines, frequencyBar("Other", other.count, maxCount, len(values), width))
	}

	return lines
}

func numericFrequencies(values []float64, limit int) ([]valueFrequency, valueFrequency) {
	frequencies := make(map[string]int)
	for _, value := range values {
		frequencies[formatAnalysisNumber(value)]++
	}

	return topFrequencies(frequencies, limit)
}

func discreteNumericDistribution(values []float64) bool {
	unique := make(map[float64]struct{})
	for _, value := range values {
		if math.Trunc(value) != value {
			return false
		}
		unique[value] = struct{}{}
		if len(unique) > analysisDiscreteLimit {
			return false
		}
	}

	return len(unique) > 0
}

func numericValueFrequencies(values []float64) []valueFrequency {
	counts := make(map[float64]int)
	for _, value := range values {
		counts[value]++
	}

	numbers := make([]float64, 0, len(counts))
	for value := range counts {
		numbers = append(numbers, value)
	}
	sort.Float64s(numbers)

	frequencies := make([]valueFrequency, 0, len(numbers))
	for _, value := range numbers {
		frequencies = append(frequencies, valueFrequency{
			value: formatAnalysisNumber(value),
			count: counts[value],
		})
	}

	return frequencies
}

func histogram(values []float64, bucketCount int) []histogramBucket {
	if len(values) == 0 {
		return nil
	}

	min := values[0]
	max := values[len(values)-1]
	if min == max {
		return []histogramBucket{{start: min, end: max, count: len(values)}}
	}

	if bucketCount > len(values) {
		bucketCount = len(values)
	}
	if bucketCount < 1 {
		bucketCount = 1
	}

	buckets := make([]histogramBucket, bucketCount)
	width := (max - min) / float64(bucketCount)
	for i := range buckets {
		buckets[i].start = min + float64(i)*width
		buckets[i].end = buckets[i].start + width
	}
	buckets[len(buckets)-1].end = max

	for _, value := range values {
		index := int((value - min) / width)
		if index >= len(buckets) {
			index = len(buckets) - 1
		}
		buckets[index].count++
	}

	return buckets
}

func categoricalAnalysisView(values map[string]int, width int) []string {
	total := categoricalCount(values)
	frequencies, other := topFrequencies(values, analysisTopValueLimit)
	lines := []string{
		titleStyle.Render("Categories"),
		statusLine(
			statusItem{label: "Unique", value: fmt.Sprint(len(values))},
			statusItem{label: "Unique Rate", value: formatRatio(len(values), total)},
			statusItem{label: "Shown", value: fmt.Sprint(len(frequencies))},
			statusItem{label: "Cardinality", value: cardinalityLabel(values, total)},
		),
		"",
		labelStyle.Render("Top Values"),
	}

	maxCount := maxFrequencyCount(frequencies)
	for _, frequency := range frequencies {
		lines = append(lines, frequencyBar(frequency.value, frequency.count, maxCount, total, width))
	}
	if other.count > 0 {
		lines = append(lines, frequencyBar("Other", other.count, maxCount, total, width))
	}

	return lines
}

func booleanAnalysisView(values map[bool]int, width int) []string {
	lines := []string{
		titleStyle.Render("Booleans"),
	}

	maxCount := 0
	for _, count := range values {
		if count > maxCount {
			maxCount = count
		}
	}

	lines = append(lines,
		frequencyBar("true", values[true], maxCount, booleanCount(values), width),
		frequencyBar("false", values[false], maxCount, booleanCount(values), width),
	)

	return lines
}

func fieldsAnalysisView(fields map[string]*analysisStats, width int) []string {
	paths := sortedAnalysisFieldPaths(fields)
	lines := []string{
		titleStyle.Render("Fields"),
	}
	return append(lines, fieldsAnalysisViewForPaths(fields, paths, width, -1)...)
}

func fieldsAnalysisViewForPaths(fields map[string]*analysisStats, paths []string, width int, selectedIndex int) []string {
	lines := []string{}
	for i, path := range paths {
		field := fields[path]
		lines = append(lines, "")
		lines = append(lines, analysisFieldView(path, field, width, selectedIndex == i)...)
	}

	return lines
}

func analysisFieldView(path string, field *analysisStats, width int, selected bool) []string {
	title := "Field"
	if selected {
		title = "> Field"
	}

	lines := []string{
		titleStyle.Render(title) + " " + valueStyle.Render(path),
		statusLine(
			statusItem{label: "Total", value: fmt.Sprint(field.total)},
			statusItem{label: "Numeric", value: fmt.Sprint(len(field.numeric))},
			statusItem{label: "Categories", value: fmt.Sprint(categoricalCount(field.categorical))},
			statusItem{label: "Booleans", value: fmt.Sprint(booleanCount(field.booleans))},
			statusItem{label: "Empty", value: fmt.Sprint(field.empty)},
			statusItem{label: "Unsupported", value: fmt.Sprint(field.unsupported)},
		),
	}

	if len(field.numeric) > 0 {
		lines = append(lines, numericAnalysisView(field.numeric, width)...)
	}
	if len(field.categorical) > 0 {
		lines = append(lines, categoricalAnalysisView(field.categorical, width)...)
	}
	if len(field.booleans) > 0 {
		lines = append(lines, booleanAnalysisView(field.booleans, width)...)
	}

	return lines
}

func sortedAnalysisFieldPaths(fields map[string]*analysisStats) []string {
	paths := make([]string, 0, len(fields))
	for path := range fields {
		paths = append(paths, path)
	}

	sort.Slice(paths, func(i, j int) bool {
		left := fields[paths[i]]
		right := fields[paths[j]]

		leftScore := analysisFieldScore(left)
		rightScore := analysisFieldScore(right)
		if leftScore != rightScore {
			return leftScore > rightScore
		}

		leftMissing := missingRate(left)
		rightMissing := missingRate(right)
		if leftMissing != rightMissing {
			return leftMissing > rightMissing
		}

		return paths[i] < paths[j]
	})

	return paths
}

func analysisFieldScore(stats *analysisStats) int {
	switch {
	case len(stats.numeric) > 0:
		return 4
	case booleanCount(stats.booleans) > 0:
		return 3
	case categoricalCount(stats.categorical) > 0:
		return 2
	case stats.empty > 0:
		return 1
	default:
		return 0
	}
}

func missingRate(stats *analysisStats) float64 {
	if stats.total == 0 {
		return 0
	}

	return float64(stats.empty) / float64(stats.total)
}

func missingFields(fields map[string]*analysisStats) []missingField {
	missing := make([]missingField, 0, len(fields))
	for path, field := range fields {
		if field.empty == 0 {
			continue
		}

		rate := 0.0
		if field.total > 0 {
			rate = float64(field.empty) / float64(field.total)
		}
		missing = append(missing, missingField{
			path:  path,
			total: field.total,
			empty: field.empty,
			rate:  rate,
		})
	}

	sort.Slice(missing, func(i, j int) bool {
		if missing[i].rate == missing[j].rate {
			return missing[i].path < missing[j].path
		}
		return missing[i].rate > missing[j].rate
	})

	return missing
}

func missingAnalysisView(missing []missingField, width int) []string {
	lines := []string{
		titleStyle.Render("Missing Data"),
	}

	maxCount := 0
	for _, field := range missing {
		if field.empty > maxCount {
			maxCount = field.empty
		}
	}

	for _, field := range missing {
		label := fmt.Sprintf("%s (%s)", field.path, formatPercent(field.empty, field.total))
		lines = append(lines, frequencyBar(label, field.empty, maxCount, field.total, width))
	}

	return lines
}

func topFrequencies(values map[string]int, limit int) ([]valueFrequency, valueFrequency) {
	frequencies := make([]valueFrequency, 0, len(values))
	for value, count := range values {
		frequencies = append(frequencies, valueFrequency{value: value, count: count})
	}

	sort.Slice(frequencies, func(i, j int) bool {
		if frequencies[i].count == frequencies[j].count {
			return frequencies[i].value < frequencies[j].value
		}
		return frequencies[i].count > frequencies[j].count
	})

	if len(frequencies) <= limit {
		return frequencies, valueFrequency{}
	}

	var other valueFrequency
	other.value = "Other"
	for _, frequency := range frequencies[limit:] {
		other.count += frequency.count
	}

	return frequencies[:limit], other
}

func frequencyBar(label string, count int, maxCount int, total int, width int) string {
	barWidth := width / 3
	if barWidth > analysisBarMaxWidth {
		barWidth = analysisBarMaxWidth
	}
	if barWidth < 6 {
		barWidth = 6
	}

	filled := 0
	if maxCount > 0 && count > 0 {
		filled = int(math.Round(float64(count) / float64(maxCount) * float64(barWidth)))
		if filled < 1 {
			filled = 1
		}
	}

	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	return fmt.Sprintf("%-24s %s %d %s", truncateAnalysisLabel(label, 24), bar, count, formatPercent(count, total))
}

func truncateAnalysisLabel(label string, maxLength int) string {
	if len(label) <= maxLength {
		return label
	}
	if maxLength <= 3 {
		return label[:maxLength]
	}

	return label[:maxLength-3] + "..."
}

func maxBucketCount(buckets []histogramBucket) int {
	var maxCount int
	for _, bucket := range buckets {
		if bucket.count > maxCount {
			maxCount = bucket.count
		}
	}
	return maxCount
}

func maxFrequencyCount(frequencies []valueFrequency) int {
	var maxCount int
	for _, frequency := range frequencies {
		if frequency.count > maxCount {
			maxCount = frequency.count
		}
	}
	return maxCount
}

func categoricalCount(values map[string]int) int {
	var count int
	for _, value := range values {
		count += value
	}
	return count
}

func booleanCount(values map[bool]int) int {
	var count int
	for _, value := range values {
		count += value
	}
	return count
}

func formatAnalysisNumber(value float64) string {
	if math.Abs(value) >= 1000 || math.Abs(value) < 0.01 && value != 0 {
		return strconv.FormatFloat(value, 'g', 4, 64)
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatPercent(count int, total int) string {
	if total <= 0 {
		return "(0%)"
	}

	percent := float64(count) / float64(total) * 100
	return fmt.Sprintf("(%s%%)", strconv.FormatFloat(percent, 'f', 1, 64))
}

func formatRatio(count int, total int) string {
	if total <= 0 {
		return "0"
	}

	return strconv.FormatFloat(float64(count)/float64(total), 'f', 2, 64)
}

func cardinalityLabel(values map[string]int, total int) string {
	if total == 0 {
		return "none"
	}

	rate := float64(len(values)) / float64(total)
	if rate >= highCardinalityRate && len(values) > 1 {
		return "high"
	}
	if len(values) <= 1 {
		return "constant"
	}

	return "normal"
}
