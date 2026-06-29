package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

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

func formatAnalysisNumber(value float64) string {
	if math.Abs(value) >= 1000 || math.Abs(value) < 0.01 && value != 0 {
		return strconv.FormatFloat(value, 'g', 4, 64)
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}
