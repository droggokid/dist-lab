package tui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type exportFormat string

const (
	exportFormatJSON exportFormat = "json"
	exportFormatCSV  exportFormat = "csv"
)

type exportPromptModel struct {
	active bool
	format exportFormat
	input  textinput.Model
	err    string
}

func (m *Model) openExportPrompt() tea.Cmd {
	input := textinput.New()
	input.Placeholder = "path/to/export"
	input.SetValue(m.defaultExportPath(exportFormatJSON))
	input.Width = m.exportPromptInputWidth()

	cmd := input.Focus()

	m.export = exportPromptModel{
		active: true,
		format: exportFormatJSON,
		input:  input,
	}
	m.err = nil
	m.notice = ""
	m.resizeViews()

	return cmd
}

func (m *Model) updateExportPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.closeExportPrompt()
			m.resizeViews()
			return m, nil
		case "tab":
			m.toggleExportFormat()
			m.resizeViews()
			return m, nil
		case "enter":
			m.saveExportPrompt()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.export.input, cmd = m.export.input.Update(msg)
	return m, cmd
}

func (m *Model) closeExportPrompt() {
	m.export = exportPromptModel{}
}

func (m *Model) resizeExportPrompt() {
	if !m.export.active {
		return
	}

	m.export.input.Width = m.exportPromptInputWidth()
}

func (m *Model) exportPromptInputWidth() int {
	width := m.borderedBlockWidth() - 4
	if width < 20 {
		return 20
	}

	return width
}

func (m *Model) toggleExportFormat() {
	m.export.err = ""
	oldFormat := m.export.format

	if m.export.format == exportFormatJSON {
		m.export.format = exportFormatCSV
	} else {
		m.export.format = exportFormatJSON
	}

	m.export.input.SetValue(swapExportExtension(m.export.input.Value(), oldFormat, m.export.format))
}

func swapExportExtension(path string, oldFormat exportFormat, newFormat exportFormat) string {
	oldExt := "." + string(oldFormat)
	if !strings.EqualFold(filepath.Ext(path), oldExt) {
		return path
	}

	return strings.TrimSuffix(path, filepath.Ext(path)) + "." + string(newFormat)
}

func (m *Model) saveExportPrompt() {
	path, err := m.exportValues(m.export.input.Value(), m.export.format)
	if err != nil {
		m.export.err = err.Error()
		m.resizeViews()
		return
	}

	m.closeExportPrompt()
	m.setNotice(path)
}

func (m *Model) exportPopup() string {
	lines := []string{
		"Export values",
		fmt.Sprintf("Format: %s", strings.ToUpper(string(m.export.format))),
		fmt.Sprintf("Values: %d", len(m.values)),
		"",
		"Path",
		m.export.input.View(),
		"",
		"enter save  tab format  esc cancel",
	}

	if m.export.err != "" {
		lines = append(lines, "", "Error", m.export.err)
	}

	return m.popupView(strings.Join(lines, "\n"))
}

func (m *Model) defaultExportPath(format exportFormat) string {
	name := sanitizeExportName(m.selectedPath)
	if name == "" {
		name = "values"
	}

	state := "raw"
	if m.valuesFiltered {
		state = "filtered"
	}

	return fmt.Sprintf("%s-%s.%s", name, state, format)
}

func (m *Model) exportValues(path string, format exportFormat) (string, error) {
	path, err := normalizeExportPath(path, format)
	if err != nil {
		return "", err
	}

	switch format {
	case exportFormatJSON:
		return m.exportValuesJSON(path)
	case exportFormatCSV:
		return m.exportValuesCSV(path)
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
		path += "." + string(format)
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

func (m *Model) exportValuesJSON(path string) (string, error) {
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

func (m *Model) exportValuesCSV(path string) (string, error) {
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
