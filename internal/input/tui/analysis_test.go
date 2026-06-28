package tui

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeValuesClassifiesScalars(t *testing.T) {
	values := []any{
		nil,
		"",
		"  ",
		"1.5",
		2,
		float64(3),
		"apple",
		"apple",
		"banana",
		true,
		false,
		true,
		map[string]any{"unsupported": true},
		[]any{"unsupported"},
	}

	stats := analyzeValues(values)

	if stats.total != len(values) {
		t.Fatalf("total = %d, want %d", stats.total, len(values))
	}
	if stats.empty != 3 {
		t.Fatalf("empty = %d, want 3", stats.empty)
	}
	if !reflect.DeepEqual(stats.numeric, []float64{1.5, 2, 3}) {
		t.Fatalf("numeric = %#v, want [1.5 2 3]", stats.numeric)
	}
	if got, want := stats.categorical, map[string]int{"apple": 2, "banana": 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("categorical = %#v, want %#v", got, want)
	}
	if got, want := stats.booleans, map[bool]int{true: 2, false: 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("booleans = %#v, want %#v", got, want)
	}
	if stats.unsupported != 2 {
		t.Fatalf("unsupported = %d, want 2", stats.unsupported)
	}
}

func TestSummarizeNumeric(t *testing.T) {
	values := []float64{1, 2, 3, 4}
	summary := summarizeNumeric(values)

	if summary.count != 4 {
		t.Fatalf("count = %d, want 4", summary.count)
	}
	if summary.min != 1 || summary.max != 4 {
		t.Fatalf("min/max = %v/%v, want 1/4", summary.min, summary.max)
	}
	if summary.mean != 2.5 {
		t.Fatalf("mean = %v, want 2.5", summary.mean)
	}
	if summary.median != 2.5 {
		t.Fatalf("median = %v, want 2.5", summary.median)
	}
	if math.Abs(summary.stddev-1.2909944487358056) > 0.0000001 {
		t.Fatalf("stddev = %v, want sample stddev", summary.stddev)
	}
	if len(summary.buckets) != 4 {
		t.Fatalf("bucket count = %d, want 4", len(summary.buckets))
	}
}

func TestHistogramHandlesSingleValue(t *testing.T) {
	buckets := histogram([]float64{7, 7, 7}, analysisBucketCount)

	want := []histogramBucket{{start: 7, end: 7, count: 3}}
	if !reflect.DeepEqual(buckets, want) {
		t.Fatalf("histogram() = %#v, want %#v", buckets, want)
	}
}

func TestTopFrequenciesSortsByCountThenValue(t *testing.T) {
	values := map[string]int{
		"b": 2,
		"a": 2,
		"c": 1,
	}

	got := topFrequencies(values, 2)
	want := []valueFrequency{
		{value: "a", count: 2},
		{value: "b", count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topFrequencies() = %#v, want %#v", got, want)
	}
}

func TestFrequencyBarUsesMinimumWidth(t *testing.T) {
	got := frequencyBar("alpha", 2, 4, 12)

	if !strings.Contains(got, "alpha") {
		t.Fatalf("frequencyBar() = %q, want label", got)
	}
	if !strings.Contains(got, "###---") {
		t.Fatalf("frequencyBar() = %q, want half-filled min-width bar", got)
	}
	if !strings.HasSuffix(got, " 2") {
		t.Fatalf("frequencyBar() = %q, want count suffix", got)
	}
}

func TestAnalysisContentRendersNumericCategoricalAndBoolean(t *testing.T) {
	content := analysisContent([]any{1, 2, "3", "apple", "apple", true, false, nil, map[string]any{}}, 90)

	for _, want := range []string{
		"Summary",
		"Numeric",
		"Distribution",
		"Categories",
		"Top Values",
		"Booleans",
		"apple",
		"true",
		"false",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("analysisContent() missing %q in:\n%s", want, content)
		}
	}
}

func TestAnalysisContentHandlesNoScalarValues(t *testing.T) {
	content := analysisContent([]any{nil, "", map[string]any{}, []any{}}, 90)

	if !strings.Contains(content, "No scalar values to analyze.") {
		t.Fatalf("analysisContent() = %q, want no scalar message", content)
	}
}
