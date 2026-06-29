package tui

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

func analyzeValues(values []any) analysisStats {
	stats := analysisStats{
		total:       len(values),
		categorical: make(map[string]int),
		booleans:    make(map[bool]int),
		fields:      make(map[string]*analysisStats),
	}

	fieldPaths := discoverAnalysisFieldPaths(values)
	for _, value := range values {
		rowFields := make(map[string][]any)
		collectAnalysisFieldValues(value, "", rowFields)

		if len(rowFields) == 0 && shouldClassifyTopLevelValue(value, len(fieldPaths) > 0) {
			classifyAnalysisScalar(value, &stats)
		}

		for _, path := range fieldPaths {
			field := analysisField(&stats, path)
			field.total++

			values, ok := rowFields[path]
			if !ok || len(values) == 0 {
				field.empty++
				continue
			}

			for _, value := range values {
				classifyAnalysisScalar(value, field)
			}
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

func discoverAnalysisFieldPaths(values []any) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		collectAnalysisFieldPaths(value, "", seen)
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	return paths
}

func collectAnalysisFieldPaths(value any, path string, paths map[string]struct{}) {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			collectAnalysisFieldPaths(v[key], joinAnalysisPath(path, key), paths)
		}
	case []any:
		childPath := path + "[]"
		if path == "" {
			childPath = "[]"
		}

		for _, item := range v {
			collectAnalysisFieldPaths(item, childPath, paths)
		}
	default:
		if path == "" {
			return
		}

		paths[path] = struct{}{}
	}
}

func collectAnalysisFieldValues(value any, path string, fields map[string][]any) {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			collectAnalysisFieldValues(v[key], joinAnalysisPath(path, key), fields)
		}
	case []any:
		childPath := path + "[]"
		if path == "" {
			childPath = "[]"
		}

		for _, item := range v {
			collectAnalysisFieldValues(item, childPath, fields)
		}
	default:
		if path == "" {
			return
		}

		fields[path] = append(fields[path], value)
	}
}

func shouldClassifyTopLevelValue(value any, hasRecursiveFields bool) bool {
	if !hasRecursiveFields {
		return true
	}

	switch value.(type) {
	case map[string]any, []any:
		return false
	default:
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
