package tui

import (
	"path/filepath"
	"strings"
)

type exportFormat string

const (
	exportFormatJSON  exportFormat = "json"
	exportFormatJSONL exportFormat = "jsonl"
	exportFormatYAML  exportFormat = "yaml"
	exportFormatCSV   exportFormat = "csv"
	exportFormatTSV   exportFormat = "tsv"
)

var exportFormats = []exportFormat{
	exportFormatJSON,
	exportFormatJSONL,
	exportFormatYAML,
	exportFormatCSV,
	exportFormatTSV,
}

func nextExportFormat(format exportFormat) exportFormat {
	for i, supported := range exportFormats {
		if format == supported {
			return exportFormats[(i+1)%len(exportFormats)]
		}
	}

	return exportFormats[0]
}

func swapExportExtension(path string, oldFormat exportFormat, newFormat exportFormat) string {
	if !exportFormatHasExtension(oldFormat, filepath.Ext(path)) {
		return path
	}

	return strings.TrimSuffix(path, filepath.Ext(path)) + "." + exportFormatExtension(newFormat)
}

func exportFormatHasExtension(format exportFormat, ext string) bool {
	for _, supportedExt := range exportFormatExtensions(format) {
		if strings.EqualFold(ext, "."+supportedExt) {
			return true
		}
	}

	return false
}

func exportFormatExtension(format exportFormat) string {
	return exportFormatExtensions(format)[0]
}

func exportFormatExtensions(format exportFormat) []string {
	switch format {
	case exportFormatJSON:
		return []string{"json"}
	case exportFormatJSONL:
		return []string{"jsonl", "ndjson"}
	case exportFormatYAML:
		return []string{"yaml", "yml"}
	case exportFormatCSV:
		return []string{"csv"}
	case exportFormatTSV:
		return []string{"tsv"}
	default:
		return []string{string(format)}
	}
}
