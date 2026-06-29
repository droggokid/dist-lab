package input

import (
	"path/filepath"
	"strings"
)

type FileFormat string

const (
	FileFormatJSON  FileFormat = "json"
	FileFormatJSONL FileFormat = "jsonl"
	FileFormatCSV   FileFormat = "csv"
	FileFormatTSV   FileFormat = "tsv"
	FileFormatYAML  FileFormat = "yaml"
)

func DetectFileFormat(filePath string) FileFormat {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jsonl", ".ndjson":
		return FileFormatJSONL
	case ".yaml", ".yml":
		return FileFormatYAML
	case ".csv":
		return FileFormatCSV
	case ".tsv":
		return FileFormatTSV
	default:
		return FileFormatJSON
	}
}
