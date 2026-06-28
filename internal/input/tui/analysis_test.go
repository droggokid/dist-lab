package tui

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
		map[string]any{},
		[]any{},
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

func TestAnalyzeValuesCollectsRecursiveObjectFields(t *testing.T) {
	values := []any{
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 4, "month": 9, "year": 2001},
		map[string]any{"day": 4, "month": 9, "year": 1996},
		map[string]any{"day": 4, "month": 9, "year": nil},
	}

	stats := analyzeValues(values)

	if stats.unsupported != 0 {
		t.Fatalf("unsupported = %d, want 0", stats.unsupported)
	}
	if len(stats.fields) != 3 {
		t.Fatalf("len(fields) = %d, want 3", len(stats.fields))
	}

	day := stats.fields["day"]
	if day == nil {
		t.Fatal("day field was not analyzed")
	}
	if !reflect.DeepEqual(day.numeric, []float64{3, 3, 4, 4, 4}) {
		t.Fatalf("day numeric = %#v, want [3 3 4 4 4]", day.numeric)
	}

	year := stats.fields["year"]
	if year == nil {
		t.Fatal("year field was not analyzed")
	}
	if !reflect.DeepEqual(year.numeric, []float64{1996, 2001, 2001, 2001}) {
		t.Fatalf("year numeric = %#v, want [1996 2001 2001 2001]", year.numeric)
	}
	if year.empty != 1 {
		t.Fatalf("year empty = %d, want 1", year.empty)
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
	if summary.q1 != 1.5 || summary.q3 != 3.5 || summary.iqr != 2 {
		t.Fatalf("quartiles = q1:%v q3:%v iqr:%v, want 1.5/3.5/2", summary.q1, summary.q3, summary.iqr)
	}
	if math.Abs(summary.stddev-1.2909944487358056) > 0.0000001 {
		t.Fatalf("stddev = %v, want sample stddev", summary.stddev)
	}
	if len(summary.outliers) != 0 {
		t.Fatalf("outliers = %#v, want none", summary.outliers)
	}
	if len(summary.buckets) != 4 {
		t.Fatalf("bucket count = %d, want 4", len(summary.buckets))
	}
}

func TestSummarizeNumericCountsOutliers(t *testing.T) {
	values := []float64{1, 2, 2, 3, 3, 4, 4, 100}
	summary := summarizeNumeric(values)

	if !reflect.DeepEqual(summary.outliers, []float64{100}) {
		t.Fatalf("outliers = %#v, want [100]", summary.outliers)
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

	frequencies, other := topFrequencies(values, 2)
	want := []valueFrequency{
		{value: "a", count: 2},
		{value: "b", count: 2},
	}
	if !reflect.DeepEqual(frequencies, want) {
		t.Fatalf("topFrequencies() = %#v, want %#v", frequencies, want)
	}
	if other.count != 1 {
		t.Fatalf("other count = %d, want 1", other.count)
	}
}

func TestFrequencyBarUsesMinimumWidth(t *testing.T) {
	got := frequencyBar("alpha", 2, 4, 8, 12)

	if !strings.Contains(got, "alpha") {
		t.Fatalf("frequencyBar() = %q, want label", got)
	}
	if !strings.Contains(got, "###---") {
		t.Fatalf("frequencyBar() = %q, want half-filled min-width bar", got)
	}
	if !strings.Contains(got, " 2 ") {
		t.Fatalf("frequencyBar() = %q, want count", got)
	}
	if !strings.Contains(got, "(25.0%)") {
		t.Fatalf("frequencyBar() = %q, want percentage", got)
	}
}

func TestAnalysisContentRendersNumericCategoricalAndBoolean(t *testing.T) {
	content := analysisContent([]any{1, 2, "3", "apple", "apple", true, false, nil, map[string]any{}}, 90)

	for _, want := range []string{
		"Summary",
		"Numeric",
		"Distribution",
		"Outliers",
		"Categories",
		"Cardinality",
		"Top Values",
		"Booleans",
		"apple",
		"true",
		"false",
		"33.3%",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("analysisContent() missing %q in:\n%s", want, content)
		}
	}
}

func TestAnalysisContentRendersOutlierValuesWithoutQuartileJargon(t *testing.T) {
	content := analysisContent([]any{1, 2, 2, 3, 3, 4, 4, 100}, 90)

	for _, want := range []string{
		"Outliers",
		"Outlier Values",
		"100",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("analysisContent() missing %q in:\n%s", want, content)
		}
	}

	for _, unwanted := range []string{"Q1", "Q3", "IQR"} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("analysisContent() should not show %q in:\n%s", unwanted, content)
		}
	}
}

func TestNumericAnalysisRendersSmallIntegerDomainsAsDiscreteValues(t *testing.T) {
	values := []any{
		map[string]any{"month": 1},
		map[string]any{"month": 1},
		map[string]any{"month": 2},
		map[string]any{"month": 12},
		map[string]any{"month": 12},
		map[string]any{"month": 12},
	}

	content := analysisContentForState(values, 90, analysisViewState{mode: analysisModeFields})

	for _, unwanted := range []string{"1..", "2.1", "10.9"} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("integer distribution should not use decimal histogram bucket %q in:\n%s", unwanted, content)
		}
	}

	for _, want := range []string{
		fmt.Sprintf("%-24s", "1"),
		fmt.Sprintf("%-24s", "2"),
		fmt.Sprintf("%-24s", "12"),
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("integer distribution missing label %q in:\n%s", want, content)
		}
	}
}

func TestAnalysisContentModesSplitObjectAnalysis(t *testing.T) {
	values := []any{
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 4, "month": 9, "year": nil},
	}

	overview := analysisContentForState(values, 90, analysisViewState{mode: analysisModeOverview})
	for _, want := range []string{
		"Summary",
		"Date",
	} {
		if !strings.Contains(overview, want) {
			t.Fatalf("overview analysis missing %q in:\n%s", want, overview)
		}
	}
	if strings.Contains(overview, "Missing Data") {
		t.Fatalf("overview analysis should not render missing page:\n%s", overview)
	}

	fields := analysisContentForState(values, 90, analysisViewState{mode: analysisModeFields})

	for _, want := range []string{
		"Fields",
		"Field",
		"day",
		"month",
		"year",
		"Empty",
		"Numeric",
	} {
		if !strings.Contains(fields, want) {
			t.Fatalf("fields analysis missing %q in:\n%s", want, fields)
		}
	}
	if strings.Contains(fields, "No scalar values to analyze.") {
		t.Fatalf("fields analysis should not show no scalar message:\n%s", fields)
	}

	missing := analysisContentForState(values, 90, analysisViewState{mode: analysisModeMissing})
	for _, want := range []string{
		"Missing Data",
		"year",
		"50.0%",
	} {
		if !strings.Contains(missing, want) {
			t.Fatalf("missing analysis missing %q in:\n%s", want, missing)
		}
	}

}

func TestMissingFieldsSortByRate(t *testing.T) {
	fields := map[string]*analysisStats{
		"year":  {total: 5, empty: 1},
		"month": {total: 5, empty: 3},
		"day":   {total: 5},
	}

	got := missingFields(fields)
	if len(got) != 2 {
		t.Fatalf("len(missingFields) = %d, want 2", len(got))
	}
	if got[0].path != "month" || got[1].path != "year" {
		t.Fatalf("missingFields order = %#v, want month then year", got)
	}
}

func TestSortedAnalysisFieldPathsPreferUsefulFields(t *testing.T) {
	fields := map[string]*analysisStats{
		"empty": {
			total: 5,
			empty: 5,
		},
		"category": {
			total:       5,
			categorical: map[string]int{"a": 3, "b": 2},
		},
		"number": {
			total:   5,
			numeric: []float64{1, 2, 3},
		},
		"flag": {
			total:    5,
			booleans: map[bool]int{true: 4, false: 1},
		},
	}

	got := sortedAnalysisFieldPaths(fields)
	want := []string{"number", "flag", "category", "empty"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedAnalysisFieldPaths() = %#v, want %#v", got, want)
	}
}

func TestDateAnalysisDetectsDayMonthYearObjects(t *testing.T) {
	values := []any{
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 4, "month": 9, "year": 1996},
		map[string]any{"day": 4, "month": 9, "year": nil},
		map[string]any{"day": 31, "month": 2, "year": 2001},
	}

	analysis := analyzeDateValues(values)
	if analysis.validCount() != 2 {
		t.Fatalf("valid dates = %d, want 2", analysis.validCount())
	}
	if analysis.missing != 1 {
		t.Fatalf("missing dates = %d, want 1", analysis.missing)
	}
	if analysis.invalid != 1 {
		t.Fatalf("invalid dates = %d, want 1", analysis.invalid)
	}
	if analysis.years[2001] != 1 || analysis.years[1996] != 1 {
		t.Fatalf("years = %#v, want 1996 and 2001 counts", analysis.years)
	}
}

func TestAnalysisContentRendersOtherBucket(t *testing.T) {
	content := analysisContent([]any{"a", "b", "c", "d", "e", "f", "g", "h", "i"}, 90)

	if !strings.Contains(content, "Other") {
		t.Fatalf("analysisContent() = %q, want Other bucket", content)
	}
	if !strings.Contains(content, "high") {
		t.Fatalf("analysisContent() = %q, want high cardinality label", content)
	}
}

func TestAnalysisContentHandlesNoScalarValues(t *testing.T) {
	content := analysisContent([]any{nil, "", map[string]any{}, []any{}}, 90)

	if !strings.Contains(content, "No scalar values to analyze.") {
		t.Fatalf("analysisContent() = %q, want no scalar message", content)
	}
}

func TestAnalysisFieldMatchesFilter(t *testing.T) {
	fields := map[string]*analysisStats{
		"user.name":   {total: 2, categorical: map[string]int{"ada": 1, "alan": 1}},
		"user.age":    {total: 2, numeric: []float64{36, 41}},
		"viewer.name": {total: 2, categorical: map[string]int{"ada": 2}},
	}

	got := analysisFieldMatches(fields, "NAME")
	want := []string{"user.name", "viewer.name"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("analysisFieldMatches() = %#v, want %#v", got, want)
	}
}

func TestUpdateAnalysisFiltersCyclesAndFocusesFields(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.state = viewAnalysis
	m.setValues([]any{
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 4, "month": 9, "year": nil},
	})
	m.rebuildAnalysis()

	mustUpdateAnalysis(t, m, keyRunes("/"))
	if !m.analysisFilterActive {
		t.Fatal("analysis filter should be active after /")
	}
	if m.analysisMode != analysisModeFields {
		t.Fatalf("analysis mode = %v, want fields", m.analysisMode)
	}

	for _, r := range "year" {
		mustUpdateAnalysis(t, m, keyRunes(string(r)))
	}
	if m.analysisFilter != "year" {
		t.Fatalf("analysis filter = %q, want year", m.analysisFilter)
	}
	if got := m.analysisMatchStatus(); got != "1/1" {
		t.Fatalf("analysis match status = %q, want 1/1", got)
	}

	mustUpdateAnalysis(t, m, keyType(tea.KeyEnter))
	if m.analysisMode != analysisModeFocus {
		t.Fatalf("analysis mode = %v, want focus", m.analysisMode)
	}
	if m.analysisFocusedField != "year" {
		t.Fatalf("focused field = %q, want year", m.analysisFocusedField)
	}

	mustUpdateAnalysis(t, m, keyType(tea.KeyEsc))
	if m.analysisMode != analysisModeFields {
		t.Fatalf("analysis mode after focus escape = %v, want fields", m.analysisMode)
	}
	mustUpdateAnalysis(t, m, keyType(tea.KeyEsc))
	if m.analysisFilter != "" {
		t.Fatalf("analysis filter after clear escape = %q, want empty", m.analysisFilter)
	}
	mustUpdateAnalysis(t, m, keyType(tea.KeyEsc))
	if m.state != viewPreview {
		t.Fatalf("state after final escape = %v, want preview", m.state)
	}
}

func TestCycleAnalysisFieldScrollsSelectedFieldIntoView(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 18
	m.state = viewAnalysis
	m.setValues([]any{
		map[string]any{
			"a": 1,
			"b": 2,
			"c": 3,
			"d": 4,
			"e": 5,
			"f": 6,
			"g": 7,
			"h": 8,
		},
	})
	m.analysisMode = analysisModeFields
	m.rebuildAnalysis()

	mustUpdateAnalysis(t, m, keyRunes("n"))
	if m.analysis.YOffset == 0 {
		t.Fatalf("analysis YOffset = 0, want selected field to be scrolled into view")
	}
	if !strings.Contains(m.analysis.View(), "> Field") {
		t.Fatalf("visible analysis should include selected field marker:\n%s", m.analysis.View())
	}
}

func TestModelUpdateDoesNotStealAnalysisFilterText(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.state = viewAnalysis
	m.setValues([]any{map[string]any{"name": "Ada"}})
	m.analysisMode = analysisModeFields
	m.analysisFilterActive = true

	model, _ := m.Update(keyRunes("a"))
	updated := model.(*Model)
	if updated.state != viewAnalysis {
		t.Fatalf("state = %v, want analysis", updated.state)
	}
	if updated.analysisFilter != "a" {
		t.Fatalf("analysis filter = %q, want a", updated.analysisFilter)
	}
}

func mustUpdateAnalysis(t *testing.T, m *Model, msg tea.KeyMsg) {
	t.Helper()
	model, _ := m.updateAnalysis(msg)
	if model != m {
		t.Fatalf("updateAnalysis returned %T, want same model pointer", model)
	}
}

func keyRunes(value string) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(value)})
}

func keyType(key tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: key})
}
