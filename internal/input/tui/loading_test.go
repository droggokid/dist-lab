package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileDoesNotKeepPartialDataAfterFailure(t *testing.T) {
	dir := t.TempDir()
	first := writeLoadFileTestFile(t, dir, "first.json", `{"name":"Ada"}`)
	invalid := writeLoadFileTestFile(t, dir, "bad.json", "{\"bad\":true}\n{")
	second := writeLoadFileTestFile(t, dir, "second.json", `{"score":2}`)

	m := NewModel()
	if err := m.loadFile(first); err != nil {
		t.Fatalf("loadFile(first) error = %v", err)
	}

	if err := m.loadFile(invalid); err == nil {
		t.Fatal("loadFile(invalid) error = nil, want error")
	}
	if got := len(m.parser.Docs); got != 1 {
		t.Fatalf("len(parser.Docs) after failed load = %d, want 1", got)
	}
	if got := len(m.filePaths); got != 1 {
		t.Fatalf("len(filePaths) after failed load = %d, want 1", got)
	}
	if got := len(m.fileSizes); got != 1 {
		t.Fatalf("len(fileSizes) after failed load = %d, want 1", got)
	}
	if parserHasField(m, "$.bad") {
		t.Fatalf("failed file field was retained in %#v", m.parser.Fields)
	}

	if err := m.loadFile(second); err != nil {
		t.Fatalf("loadFile(second) error = %v", err)
	}
	if got := m.docCount; got != 2 {
		t.Fatalf("docCount after second load = %d, want 2", got)
	}
	if got := len(m.filePaths); got != 2 {
		t.Fatalf("len(filePaths) after second load = %d, want 2", got)
	}
	if !parserHasField(m, "$.name") || !parserHasField(m, "$.score") {
		t.Fatalf("successful file fields missing from %#v", m.parser.Fields)
	}
	if parserHasField(m, "$.bad") {
		t.Fatalf("failed file field was retained after second load in %#v", m.parser.Fields)
	}
}

func parserHasField(m *Model, path string) bool {
	for _, field := range m.parser.Fields {
		if field.Path == path {
			return true
		}
	}

	return false
}

func writeLoadFileTestFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
