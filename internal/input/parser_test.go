package input

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAddFileDiscoversFieldsAndSelectsValues(t *testing.T) {
	path := writeJSONFile(t, "data.json", strings.Join([]string{
		`{"viewer":{"name":"Ada","friends":[{"name":"Bob"},{"name":"Cam"}],"empty":null}}`,
		`{"viewer":{"name":"Lin","friends":[{"name":"Drew"}]}}`,
	}, "\n"))

	parser := NewParser()
	if err := parser.AddFile(path); err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	if got := len(parser.Docs); got != 2 {
		t.Fatalf("len(Docs) = %d, want 2", got)
	}

	assertFieldsContain(t, parser.Fields,
		"$.viewer",
		"$.viewer.name",
		"$.viewer.friends",
		"$.viewer.friends[]",
		"$.viewer.friends[].name",
		"$.viewer.empty",
	)

	names, err := parser.HandleSelection("$.viewer.name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection(viewer.name) error = %v", err)
	}
	if want := []any{"Ada", "Lin"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("HandleSelection(viewer.name) = %#v, want %#v", names, want)
	}

	friendNames, err := parser.HandleSelection("$.viewer.friends[].name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection(friends[].name) error = %v", err)
	}
	if want := []any{"Bob", "Cam", "Drew"}; !reflect.DeepEqual(friendNames, want) {
		t.Fatalf("HandleSelection(friends[].name) = %#v, want %#v", friendNames, want)
	}
}

func TestAddFileHandlesRootArrays(t *testing.T) {
	path := writeJSONFile(t, "array.json", `[{"viewer":{"name":"Ada"}},{"viewer":{"name":"Lin"}}]`)

	parser := NewParser()
	if err := parser.AddFile(path); err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	assertFieldsContain(t, parser.Fields, "$[]", "$[].viewer", "$[].viewer.name")

	values, err := parser.HandleSelection("$[].viewer.name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection($[].viewer.name) error = %v", err)
	}
	if want := []any{"Ada", "Lin"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("HandleSelection($[].viewer.name) = %#v, want %#v", values, want)
	}
}

func TestAddFileHandlesJSONLines(t *testing.T) {
	path := writeJSONFile(t, "data.jsonl", strings.Join([]string{
		`{"name":"Ada","score":1}`,
		``,
		`{"name":"Lin","score":2}`,
	}, "\n"))

	parser := NewParser()
	if err := parser.AddFile(path); err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	assertFieldsContain(t, parser.Fields, "$.name", "$.score")

	values, err := parser.HandleSelection("$.name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection($.name) error = %v", err)
	}
	if want := []any{"Ada", "Lin"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("HandleSelection($.name) = %#v, want %#v", values, want)
	}
}

func TestAddFileHandlesYAML(t *testing.T) {
	path := writeJSONFile(t, "data.yaml", strings.Join([]string{
		"viewer:",
		"  name: Ada",
		"  friends:",
		"    - name: Bob",
		"    - name: Cam",
		"column with spaces: first",
		"1: numeric key",
		"---",
		"viewer:",
		"  name: Lin",
		"  friends:",
		"    - name: Drew",
		"column with spaces: second",
	}, "\n"))

	parser := NewParser()
	if err := parser.AddFile(path); err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	assertFieldsContain(t, parser.Fields,
		"$.viewer",
		"$.viewer.name",
		"$.viewer.friends",
		"$.viewer.friends[]",
		"$.viewer.friends[].name",
		"$[\"column with spaces\"]",
		"$[\"1\"]",
	)

	names, err := parser.HandleSelection("$.viewer.name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection($.viewer.name) error = %v", err)
	}
	if want := []any{"Ada", "Lin"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("HandleSelection($.viewer.name) = %#v, want %#v", names, want)
	}

	friendNames, err := parser.HandleSelection("$.viewer.friends[].name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection(friends[].name) error = %v", err)
	}
	if want := []any{"Bob", "Cam", "Drew"}; !reflect.DeepEqual(friendNames, want) {
		t.Fatalf("HandleSelection(friends[].name) = %#v, want %#v", friendNames, want)
	}

	numericKey, err := parser.HandleSelection(`$["1"]`, parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection(numeric key) error = %v", err)
	}
	if want := []any{"numeric key", nil}; !reflect.DeepEqual(numericKey, want) {
		t.Fatalf("HandleSelection(numeric key) = %#v, want %#v", numericKey, want)
	}
}

func TestAddFileHandlesCSV(t *testing.T) {
	path := writeJSONFile(t, "data.csv", strings.Join([]string{
		"name,score,column with spaces,,name",
		"Ada,1,first,blank header,duplicate",
		"Lin,2,second,,duplicate two,extra value",
	}, "\n"))

	parser := NewParser()
	if err := parser.AddFile(path); err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	assertFieldsContain(t, parser.Fields,
		"$.name",
		"$.score",
		"$[\"column with spaces\"]",
		"$.column_4",
		"$.name_2",
		"$.extra_1",
	)

	names, err := parser.HandleSelection("$.name", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection($.name) error = %v", err)
	}
	if want := []any{"Ada", "Lin"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("HandleSelection($.name) = %#v, want %#v", names, want)
	}

	spacedValues, err := parser.HandleSelection(`$["column with spaces"]`, parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection(spaced header) error = %v", err)
	}
	if want := []any{"first", "second"}; !reflect.DeepEqual(spacedValues, want) {
		t.Fatalf("HandleSelection(spaced header) = %#v, want %#v", spacedValues, want)
	}
}

func TestAddFileHandlesTSV(t *testing.T) {
	path := writeJSONFile(t, "data.tsv", "name\tscore\nAda\t1\nLin\t2\n")

	parser := NewParser()
	if err := parser.AddFile(path); err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	values, err := parser.HandleSelection("$.score", parser.Docs)
	if err != nil {
		t.Fatalf("HandleSelection($.score) error = %v", err)
	}
	if want := []any{"1", "2"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("HandleSelection($.score) = %#v, want %#v", values, want)
	}
}

func TestAddFileErrors(t *testing.T) {
	tests := []struct {
		name       string
		fileName   string
		content    string
		wantErr    string
		wantDocErr bool
	}{
		{
			name:     "empty",
			fileName: "bad.json",
			content:  "",
			wantErr:  "is empty",
		},
		{
			name:     "invalid first document",
			fileName: "bad.json",
			content:  `{`,
			wantErr:  "unsupported file format or invalid JSON",
		},
		{
			name:       "invalid later document",
			fileName:   "bad.json",
			content:    "{\"ok\":true}\n{",
			wantErr:    "parse error",
			wantDocErr: true,
		},
		{
			name:     "invalid json lines",
			fileName: "bad.jsonl",
			content:  "{\"ok\":true}\n{",
			wantErr:  "line 2",
		},
		{
			name:     "invalid yaml",
			fileName: "bad.yaml",
			content:  "viewer: [unterminated",
			wantErr:  "invalid YAML",
		},
		{
			name:     "csv header only",
			fileName: "bad.csv",
			content:  "name,score\n",
			wantErr:  "has no data rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeJSONFile(t, tt.fileName, tt.content)
			parser := NewParser()

			err := parser.AddFile(path)
			if err == nil {
				t.Fatal("AddFile() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("AddFile() error = %q, want substring %q", err, tt.wantErr)
			}
			if tt.wantDocErr && !strings.Contains(err.Error(), "document 2") {
				t.Fatalf("AddFile() error = %q, want document number", err)
			}
		})
	}
}

func TestDetectFileFormat(t *testing.T) {
	tests := []struct {
		path string
		want FileFormat
	}{
		{path: "data.json", want: FileFormatJSON},
		{path: "data.JSON", want: FileFormatJSON},
		{path: "data.jsonl", want: FileFormatJSONL},
		{path: "data.ndjson", want: FileFormatJSONL},
		{path: "data.yaml", want: FileFormatYAML},
		{path: "data.yml", want: FileFormatYAML},
		{path: "data.csv", want: FileFormatCSV},
		{path: "data.tsv", want: FileFormatTSV},
		{path: "data.unknown", want: FileFormatJSON},
	}

	for _, tt := range tests {
		if got := DetectFileFormat(tt.path); got != tt.want {
			t.Fatalf("DetectFileFormat(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func assertFieldsContain(t *testing.T, fields []Field, paths ...string) {
	t.Helper()

	got := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		got[field.Path] = struct{}{}
	}

	for _, path := range paths {
		if _, ok := got[path]; !ok {
			t.Fatalf("field %q not found in %#v", path, fields)
		}
	}
}

func writeJSONFile(t *testing.T, name string, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
