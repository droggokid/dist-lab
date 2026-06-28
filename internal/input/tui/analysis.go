package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	analysisTopValueLimit = 8
	analysisBarMaxWidth   = 32
	analysisBucketCount   = 10
)

type analysisStats struct {
	total       int
	empty       int
	numeric     []float64
	categorical map[string]int
	booleans    map[bool]int
	unsupported int
}

type numericSummary struct {
	count   int
	min     float64
	max     float64
	mean    float64
	median  float64
	stddev  float64
	buckets []histogramBucket
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

func (m *Model) updateAnalysis(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "p":
			m.changeState(viewPreview)
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
	return viewHeaderTitle(
		titleStyle.Render("Analysis")+" "+badge("CURRENT VALUES"),
		m.fileInfoStatus(),
		statusLine(statusItem{label: "Path", value: m.selectedPath}),
		statusLine(statusItem{label: "Values", value: m.valuesStatus()}),
	)
}

func (m *Model) analysisFooterText() string {
	return helpFooter(
		keyHelp{key: "up/down", label: "scroll"},
		keyHelp{key: "pgup/pgdn", label: "page"},
		keyHelp{key: "p/esc", label: "preview"},
		keyHelp{key: "f", label: "change field"},
		keyHelp{key: "a", label: "add file"},
		keyHelp{key: "o", label: "new file"},
		keyHelp{key: "q", label: "quit"},
	)
}

func (m *Model) rebuildAnalysis() {
	content := analysisContent(m.values, m.analysisContentWidth())
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
		m.analysis.SetContent(analysisContent(m.values, m.analysisContentWidth()))
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

func analysisContent(values []any, width int) string {
	stats := analyzeValues(values)
	lines := []string{
		titleStyle.Render("Summary"),
		statusLine(statusItem{label: "Total", value: fmt.Sprint(stats.total)}),
		statusLine(
			statusItem{label: "Numeric", value: fmt.Sprint(len(stats.numeric))},
			statusItem{label: "Categories", value: fmt.Sprint(categoricalCount(stats.categorical))},
			statusItem{label: "Booleans", value: fmt.Sprint(booleanCount(stats.booleans))},
			statusItem{label: "Empty", value: fmt.Sprint(stats.empty)},
			statusItem{label: "Unsupported", value: fmt.Sprint(stats.unsupported)},
		),
	}

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

	if len(stats.numeric) == 0 && len(stats.categorical) == 0 && len(stats.booleans) == 0 {
		lines = append(lines, "", "No scalar values to analyze.")
	}

	return strings.Join(lines, "\n")
}

func analyzeValues(values []any) analysisStats {
	stats := analysisStats{
		total:       len(values),
		categorical: make(map[string]int),
		booleans:    make(map[bool]int),
	}

	for _, value := range values {
		classifyAnalysisValue(value, &stats)
	}

	sort.Float64s(stats.numeric)
	return stats
}

func classifyAnalysisValue(value any, stats *analysisStats) {
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
		),
		"",
		labelStyle.Render("Distribution"),
	}

	maxCount := maxBucketCount(summary.buckets)
	for _, bucket := range summary.buckets {
		label := fmt.Sprintf("%s..%s", formatAnalysisNumber(bucket.start), formatAnalysisNumber(bucket.end))
		lines = append(lines, frequencyBar(label, bucket.count, maxCount, width))
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
	} else {
		summary.median = values[middle]
	}

	if len(values) > 1 {
		var squared float64
		for _, value := range values {
			delta := value - summary.mean
			squared += delta * delta
		}
		summary.stddev = math.Sqrt(squared / float64(len(values)-1))
	}

	summary.buckets = histogram(values, analysisBucketCount)
	return summary
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
	frequencies := topFrequencies(values, analysisTopValueLimit)
	lines := []string{
		titleStyle.Render("Categories"),
		statusLine(
			statusItem{label: "Unique", value: fmt.Sprint(len(values))},
			statusItem{label: "Shown", value: fmt.Sprint(len(frequencies))},
		),
		"",
		labelStyle.Render("Top Values"),
	}

	maxCount := maxFrequencyCount(frequencies)
	for _, frequency := range frequencies {
		lines = append(lines, frequencyBar(frequency.value, frequency.count, maxCount, width))
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
		frequencyBar("true", values[true], maxCount, width),
		frequencyBar("false", values[false], maxCount, width),
	)

	return lines
}

func topFrequencies(values map[string]int, limit int) []valueFrequency {
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

	if len(frequencies) > limit {
		frequencies = frequencies[:limit]
	}

	return frequencies
}

func frequencyBar(label string, count int, maxCount int, width int) string {
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
	return fmt.Sprintf("%-24s %s %d", truncateAnalysisLabel(label, 24), bar, count)
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
