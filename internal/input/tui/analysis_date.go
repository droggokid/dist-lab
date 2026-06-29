package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
