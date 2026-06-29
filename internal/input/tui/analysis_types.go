package tui

import "time"

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
