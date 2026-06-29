package tui

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSwapExportExtension(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		oldFormat exportFormat
		newFormat exportFormat
		want      string
	}{
		{
			name:      "matching extension",
			path:      "values.json",
			oldFormat: exportFormatJSON,
			newFormat: exportFormatCSV,
			want:      "values.csv",
		},
		{
			name:      "case insensitive match",
			path:      "values.JSON",
			oldFormat: exportFormatJSON,
			newFormat: exportFormatCSV,
			want:      "values.csv",
		},
		{
			name:      "jsonl alias",
			path:      "values.ndjson",
			oldFormat: exportFormatJSONL,
			newFormat: exportFormatYAML,
			want:      "values.yaml",
		},
		{
			name:      "yaml alias",
			path:      "values.yml",
			oldFormat: exportFormatYAML,
			newFormat: exportFormatTSV,
			want:      "values.tsv",
		},
		{
			name:      "unrelated extension",
			path:      "values.txt",
			oldFormat: exportFormatJSON,
			newFormat: exportFormatCSV,
			want:      "values.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := swapExportExtension(tt.path, tt.oldFormat, tt.newFormat); got != tt.want {
				t.Fatalf("swapExportExtension() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNextExportFormatCycles(t *testing.T) {
	format := exportFormatJSON

	format = nextExportFormat(format)
	if format != exportFormatJSONL {
		t.Fatalf("nextExportFormat(JSON) = %q, want %q", format, exportFormatJSONL)
	}

	format = nextExportFormat(format)
	if format != exportFormatYAML {
		t.Fatalf("nextExportFormat(JSONL) = %q, want %q", format, exportFormatYAML)
	}

	format = nextExportFormat(format)
	if format != exportFormatCSV {
		t.Fatalf("nextExportFormat(YAML) = %q, want %q", format, exportFormatCSV)
	}

	format = nextExportFormat(format)
	if format != exportFormatTSV {
		t.Fatalf("nextExportFormat(CSV) = %q, want %q", format, exportFormatTSV)
	}

	format = nextExportFormat(format)
	if format != exportFormatJSON {
		t.Fatalf("nextExportFormat(TSV) = %q, want %q", format, exportFormatJSON)
	}
}

func TestNormalizeExportPath(t *testing.T) {
	dir := t.TempDir()

	got, err := normalizeExportPath(filepath.Join(dir, "nested", "values"), exportFormatJSON)
	if err != nil {
		t.Fatalf("normalizeExportPath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("normalizeExportPath() = %q, want absolute path", got)
	}
	if filepath.Base(got) != "values.json" {
		t.Fatalf("normalizeExportPath() base = %q, want values.json", filepath.Base(got))
	}
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Fatalf("export directory was not created: %v", err)
	}

	if _, err := normalizeExportPath("  ", exportFormatJSON); err == nil {
		t.Fatal("normalizeExportPath(empty) error = nil, want error")
	}
}

func TestDefaultExportPath(t *testing.T) {
	m := NewModel()
	m.selectedPath = "$.viewer.all_friends[].name"

	if got, want := m.defaultExportPath(exportFormatJSON), "viewer_all_friends_name-raw.json"; got != want {
		t.Fatalf("defaultExportPath(raw) = %q, want %q", got, want)
	}

	m.valuesFiltered = true
	if got, want := m.defaultExportPath(exportFormatYAML), "viewer_all_friends_name-filtered.yaml"; got != want {
		t.Fatalf("defaultExportPath(filtered) = %q, want %q", got, want)
	}
}

func TestExportValuesJSON(t *testing.T) {
	m := NewModel()
	m.values = []any{map[string]any{"name": "Ada", "active": true}}

	path, err := m.exportValues(filepath.Join(t.TempDir(), "values"), exportFormatJSON)
	if err != nil {
		t.Fatalf("exportValues(JSON) error = %v", err)
	}
	if filepath.Ext(path) != ".json" {
		t.Fatalf("exportValues(JSON) path = %q, want .json extension", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got []any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("exported JSON is invalid: %v", err)
	}

	want := []any{map[string]any{"name": "Ada", "active": true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exported JSON = %#v, want %#v", got, want)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatal("exported JSON should end with newline")
	}
}

func TestExportValuesJSONL(t *testing.T) {
	m := NewModel()
	m.values = []any{map[string]any{"name": "Ada"}, "plain"}

	path, err := m.exportValues(filepath.Join(t.TempDir(), "values"), exportFormatJSONL)
	if err != nil {
		t.Fatalf("exportValues(JSONL) error = %v", err)
	}
	if filepath.Ext(path) != ".jsonl" {
		t.Fatalf("exportValues(JSONL) path = %q, want .jsonl extension", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got, want := string(data), "{\"name\":\"Ada\"}\n\"plain\"\n"; got != want {
		t.Fatalf("JSONL export = %q, want %q", got, want)
	}
}

func TestWriteValuesJSONLReturnsCloseError(t *testing.T) {
	closeErr := errors.New("delayed close")
	file := &closeErrorBuffer{err: closeErr}

	err := writeValuesJSONLFile("values.jsonl", file, []any{"plain"})
	if !errors.Is(err, closeErr) {
		t.Fatalf("writeValuesJSONLFile() error = %v, want %v", err, closeErr)
	}
	if !strings.Contains(err.Error(), "close jsonl export") {
		t.Fatalf("writeValuesJSONLFile() error = %q, want close context", err)
	}
	if got, want := file.String(), "\"plain\"\n"; got != want {
		t.Fatalf("JSONL writes before close = %q, want %q", got, want)
	}
}

func TestExportValuesYAML(t *testing.T) {
	m := NewModel()
	m.values = []any{map[string]any{"name": "Ada", "active": true}}

	path, err := m.exportValues(filepath.Join(t.TempDir(), "values"), exportFormatYAML)
	if err != nil {
		t.Fatalf("exportValues(YAML) error = %v", err)
	}
	if filepath.Ext(path) != ".yaml" {
		t.Fatalf("exportValues(YAML) path = %q, want .yaml extension", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got []map[string]any
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("exported YAML is invalid: %v", err)
	}

	want := []map[string]any{{"name": "Ada", "active": true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exported YAML = %#v, want %#v", got, want)
	}
}

func TestExportValuesCSVFlattensObjects(t *testing.T) {
	m := NewModel()
	m.values = []any{
		map[string]any{
			"page_info": map[string]any{
				"end_cursor":    "4",
				"has_next_page": true,
			},
			"viewer": nil,
			"tags":   []any{"a", "b"},
		},
		map[string]any{
			"page_info": map[string]any{
				"end_cursor":    "5",
				"has_next_page": false,
			},
			"viewer": "Ada",
			"tags":   []any{},
		},
	}

	path, err := m.exportValues(filepath.Join(t.TempDir(), "values.csv"), exportFormatCSV)
	if err != nil {
		t.Fatalf("exportValues(CSV) error = %v", err)
	}

	records := readCSVFile(t, path)
	want := [][]string{
		{"page_info.end_cursor", "page_info.has_next_page", "tags", "viewer"},
		{"4", "true", `["a","b"]`, ""},
		{"5", "false", "[]", "Ada"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("CSV records = %#v, want %#v", records, want)
	}
}

func TestExportValuesTSVFlattensObjects(t *testing.T) {
	m := NewModel()
	m.values = []any{
		map[string]any{
			"name":  "Ada",
			"score": 1,
			"tags":  []any{"a", "b"},
		},
		map[string]any{
			"name":  "Lin",
			"score": 2,
			"tags":  []any{},
		},
	}

	path, err := m.exportValues(filepath.Join(t.TempDir(), "values"), exportFormatTSV)
	if err != nil {
		t.Fatalf("exportValues(TSV) error = %v", err)
	}
	if filepath.Ext(path) != ".tsv" {
		t.Fatalf("exportValues(TSV) path = %q, want .tsv extension", path)
	}

	records := readDelimitedFile(t, path, '\t')
	want := [][]string{
		{"name", "score", "tags"},
		{"Ada", "1", `["a","b"]`},
		{"Lin", "2", "[]"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("TSV records = %#v, want %#v", records, want)
	}
}

func TestExportValuesCSVHandlesScalars(t *testing.T) {
	m := NewModel()
	m.values = []any{"alpha", 2, true}

	path, err := m.exportValues(filepath.Join(t.TempDir(), "values.csv"), exportFormatCSV)
	if err != nil {
		t.Fatalf("exportValues(CSV) error = %v", err)
	}

	records := readCSVFile(t, path)
	want := [][]string{
		{"value"},
		{"alpha"},
		{"2"},
		{"true"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("CSV records = %#v, want %#v", records, want)
	}
}

func TestWriteValuesDelimitedReturnsCloseError(t *testing.T) {
	closeErr := errors.New("delayed close")
	file := &closeErrorBuffer{err: closeErr}

	err := writeValuesDelimitedFile("values.csv", file, ',', "csv", []any{map[string]any{"name": "Ada"}})
	if !errors.Is(err, closeErr) {
		t.Fatalf("writeValuesDelimitedFile() error = %v, want %v", err, closeErr)
	}
	if !strings.Contains(err.Error(), "close csv export") {
		t.Fatalf("writeValuesDelimitedFile() error = %q, want close context", err)
	}
	if got, want := file.String(), "name\nAda\n"; got != want {
		t.Fatalf("CSV writes before close = %q, want %q", got, want)
	}
}

func TestCSVCellFormatsValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: ""},
		{name: "string", value: "alpha", want: "alpha"},
		{name: "bool", value: true, want: "true"},
		{name: "array", value: []any{"a", "b"}, want: `["a","b"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := csvCell(tt.value)
			if err != nil {
				t.Fatalf("csvCell() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("csvCell() = %q, want %q", got, tt.want)
			}
		})
	}
}

func readCSVFile(t *testing.T, path string) [][]string {
	return readDelimitedFile(t, path, ',')
}

func readDelimitedFile(t *testing.T, path string, comma rune) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = comma
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	return records
}

type closeErrorBuffer struct {
	bytes.Buffer
	err error
}

func (b *closeErrorBuffer) Close() error {
	return b.err
}
