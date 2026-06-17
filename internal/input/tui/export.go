package tui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type exportFormat string

const (
	exportFormatJSON exportFormat = "json"
	exportFormatCSV  exportFormat = "csv"
	exportDirName                 = "exports"
)

func EnsureExportDir() (string, error) {
	dir, err := filepath.Abs(exportDirName)
	if err != nil {
		return "", fmt.Errorf("resolve exports directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create exports directory: %w", err)
	}

	return dir, nil
}

func (m *Model) exportValues(format exportFormat) (string, error) {
	switch format {
	case exportFormatJSON:
		return m.exportValuesJSON()
	case exportFormatCSV:
		return m.exportValuesCSV()
	default:
		return "", fmt.Errorf("unsupported export format %q", format)
	}
}

func (m *Model) exportValuesJSON() (string, error) {
	path, err := m.exportPath("json")
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(m.values, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode json export: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write json export %q: %w", path, err)
	}

	return path, nil
}

func (m *Model) exportValuesCSV() (string, error) {
	path, err := m.exportPath("csv")
	if err != nil {
		return "", err
	}

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create csv export %q: %w", path, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	headers := csvHeaders(m.values)
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("write csv header: %w", err)
	}

	for _, value := range m.values {
		row, err := csvRow(value, headers)
		if err != nil {
			return "", err
		}

		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush csv export: %w", err)
	}

	return path, nil
}

func (m *Model) exportPath(ext string) (string, error) {
	dir, err := EnsureExportDir()
	if err != nil {
		return "", err
	}

	name := sanitizeExportName(m.selectedPath)
	if name == "" {
		name = "values"
	}

	state := "raw"
	if m.valuesFiltered {
		state = "filtered"
	}

	return filepath.Join(dir, fmt.Sprintf("%s-%s.%s", name, state, ext)), nil
}

func sanitizeExportName(value string) string {
	value = strings.ToLower(value)

	var b strings.Builder
	lastWasSeparator := false
	for _, r := range value {
		isLetter := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit {
			b.WriteRune(r)
			lastWasSeparator = false
		} else if b.Len() > 0 && !lastWasSeparator {
			b.WriteByte('_')
			lastWasSeparator = true
		}

		if b.Len() >= 80 {
			break
		}
	}

	return strings.Trim(b.String(), "_")
}

func csvHeaders(values []any) []string {
	keys := make(map[string]struct{})
	hasObject := false
	hasValueColumn := len(values) == 0

	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			hasValueColumn = true
			continue
		}

		hasObject = true
		for key := range object {
			keys[key] = struct{}{}
		}
	}

	objectHeaders := make([]string, 0, len(keys))
	for key := range keys {
		objectHeaders = append(objectHeaders, key)
	}
	sort.Strings(objectHeaders)

	if !hasObject {
		return []string{"value"}
	}

	if hasValueColumn {
		valueHeader := uniqueValueHeader(keys)
		return append([]string{valueHeader}, objectHeaders...)
	}

	return objectHeaders
}

func uniqueValueHeader(keys map[string]struct{}) string {
	header := "value"
	for {
		if _, exists := keys[header]; !exists {
			return header
		}

		header = "_" + header
	}
}

func csvRow(value any, headers []string) ([]string, error) {
	row := make([]string, len(headers))
	object, isObject := value.(map[string]any)

	if !isObject {
		cell, err := csvCell(value)
		if err != nil {
			return nil, err
		}

		if len(row) > 0 {
			row[0] = cell
		}

		return row, nil
	}

	for i, header := range headers {
		cell, err := csvCell(object[header])
		if err != nil {
			return nil, err
		}

		row[i] = cell
	}

	return row, nil
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
