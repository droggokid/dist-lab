package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

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
