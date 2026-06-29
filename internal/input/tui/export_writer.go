package tui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type exportWriteCloser interface {
	io.Writer
	Close() error
}

func writeValues(path string, format exportFormat, values []any) (string, error) {
	path, err := normalizeExportPath(path, format)
	if err != nil {
		return "", err
	}

	return writeValuesToPath(path, format, values)
}

func writeValuesToPath(path string, format exportFormat, values []any) (string, error) {
	switch format {
	case exportFormatJSON:
		return writeValuesJSON(path, values)
	case exportFormatJSONL:
		return writeValuesJSONL(path, values)
	case exportFormatYAML:
		return writeValuesYAML(path, values)
	case exportFormatCSV:
		return writeValuesDelimited(path, ',', "csv", values)
	case exportFormatTSV:
		return writeValuesDelimited(path, '\t', "tsv", values)
	default:
		return "", fmt.Errorf("unsupported export format %q", format)
	}
}

func normalizeExportPath(path string, format exportFormat) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("export path is required")
	}

	path, err := expandHomePath(path)
	if err != nil {
		return "", err
	}

	if filepath.Ext(path) == "" {
		path += "." + exportFormatExtension(format)
	}

	path, err = filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve export path: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create export directory %q: %w", dir, err)
	}

	return path, nil
}

func expandHomePath(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}

func writeValuesJSON(path string, values []any) (string, error) {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode json export: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write json export %q: %w", path, err)
	}

	return path, nil
}

func writeValuesJSONL(path string, values []any) (string, error) {
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create jsonl export %q: %w", path, err)
	}

	if err := writeValuesJSONLFile(path, file, values); err != nil {
		return "", err
	}

	return path, nil
}

func writeValuesJSONLFile(path string, file exportWriteCloser, values []any) error {
	encoder := json.NewEncoder(file)
	var writeErr error
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			writeErr = fmt.Errorf("encode jsonl export: %w", err)
			break
		}
	}

	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return fmt.Errorf("close jsonl export %q: %w", path, closeErr)
	}

	return nil
}

func writeValuesYAML(path string, values []any) (string, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode yaml export: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write yaml export %q: %w", path, err)
	}

	return path, nil
}

func writeValuesDelimited(path string, comma rune, formatName string, values []any) (string, error) {
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create %s export %q: %w", formatName, path, err)
	}

	if err := writeValuesDelimitedFile(path, file, comma, formatName, values); err != nil {
		return "", err
	}

	return path, nil
}

func writeValuesDelimitedFile(path string, file exportWriteCloser, comma rune, formatName string, values []any) error {
	writer := csv.NewWriter(file)
	writer.Comma = comma

	records := flattenRecords(values)
	headers := collectHeaders(records)
	writeErr := writeRecords(writer, headers, records, formatName)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return fmt.Errorf("close %s export %q: %w", formatName, path, closeErr)
	}

	return nil
}

func writeRecords(writer *csv.Writer, headers []string, records []csvRecord, formatName string) error {
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write %s header: %w", formatName, err)
	}

	for _, record := range records {
		row, err := rowFromRecord(record, headers)
		if err != nil {
			return err
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write %s row: %w", formatName, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush %s export: %w", formatName, err)
	}

	return nil
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
