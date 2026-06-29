package input

import "fmt"

func normalizeDataValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(v))
		for key, item := range v {
			normalized[key] = normalizeDataValue(item)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(v))
		for key, item := range v {
			normalized[fmt.Sprint(key)] = normalizeDataValue(item)
		}
		return normalized
	case []any:
		normalized := make([]any, len(v))
		for i, item := range v {
			normalized[i] = normalizeDataValue(item)
		}
		return normalized
	case []map[string]any:
		normalized := make([]any, len(v))
		for i, item := range v {
			normalized[i] = normalizeDataValue(item)
		}
		return normalized
	case nil, string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return v
	default:
		return fmt.Sprint(v)
	}
}
