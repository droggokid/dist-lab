package tui

import (
	"encoding/json"
	"fmt"
	"sort"
)

type csvRecord map[string]any

func flattenRecords(values []any) []csvRecord {
	records := make([]csvRecord, len(values))
	for i, value := range values {
		records[i] = flattenRecord(value)
	}

	return records
}

func flattenRecord(value any) csvRecord {
	object, ok := value.(map[string]any)
	if !ok {
		return csvRecord{"value": value}
	}

	record := make(csvRecord)
	flattenObjectFields(record, "", object)
	if len(record) == 0 {
		record["value"] = value
	}

	return record
}

func collectHeaders(records []csvRecord) []string {
	keys := make(map[string]struct{})
	for _, record := range records {
		for key := range record {
			keys[key] = struct{}{}
		}
	}

	if len(keys) == 0 {
		return []string{"value"}
	}

	headers := make([]string, 0, len(keys))
	for key := range keys {
		headers = append(headers, key)
	}
	sort.Strings(headers)

	if _, hasValue := keys["value"]; hasValue && len(headers) > 1 {
		return append([]string{"value"}, withoutHeader(headers, "value")...)
	}

	return headers
}

func withoutHeader(headers []string, remove string) []string {
	filtered := make([]string, 0, len(headers)-1)
	for _, header := range headers {
		if header != remove {
			filtered = append(filtered, header)
		}
	}

	return filtered
}

func rowFromRecord(record csvRecord, headers []string) ([]string, error) {
	row := make([]string, len(headers))
	for i, header := range headers {
		value, exists := record[header]
		if !exists {
			continue
		}

		cell, err := csvCell(value)
		if err != nil {
			return nil, err
		}

		row[i] = cell
	}

	return row, nil
}

func flattenObjectFields(record csvRecord, prefix string, object map[string]any) bool {
	var added bool

	for key, value := range object {
		header := key
		if prefix != "" {
			header = prefix + "." + key
		}

		nested, ok := value.(map[string]any)
		if !ok {
			record[header] = value
			added = true
			continue
		}

		if !flattenObjectFields(record, header, nested) {
			record[header] = nested
		}
		added = true
	}

	return added
}

func csvCell(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case bool:
		return fmt.Sprint(v), nil
	case float64:
		return fmt.Sprint(v), nil
	case int:
		return fmt.Sprint(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("encode csv cell: %w", err)
		}

		return string(data), nil
	}
}
