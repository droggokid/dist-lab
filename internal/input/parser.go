package input

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

type FileFormat string

const (
	FileFormatJSON  FileFormat = "json"
	FileFormatJSONL FileFormat = "jsonl"
	FileFormatCSV   FileFormat = "csv"
	FileFormatTSV   FileFormat = "tsv"
	FileFormatYAML  FileFormat = "yaml"
)

type Parser struct {
	Fields     []Field
	Docs       []any
	seenFields map[string]struct{}
}

type Field struct {
	Path string
}

func NewParser() *Parser {
	return &Parser{
		seenFields: make(map[string]struct{}),
	}
}

func (p *Parser) AddFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	format := DetectFileFormat(filePath)
	switch format {
	case FileFormatJSON:
		return p.addJSONFile(filePath, file)
	case FileFormatJSONL:
		return p.addJSONLinesFile(filePath, file)
	case FileFormatYAML:
		return p.addYAMLFile(filePath, file)
	case FileFormatCSV:
		return p.addDelimitedFile(filePath, file, ',')
	case FileFormatTSV:
		return p.addDelimitedFile(filePath, file, '\t')
	default:
		return fmt.Errorf("unsupported file format %q for %q", format, filePath)
	}
}

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

func (p *Parser) addJSONFile(filePath string, reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	var docCount int

	for {
		var doc any

		err := decoder.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				if docCount == 0 {
					return fmt.Errorf("file %q is empty", filePath)
				}
				break
			}

			if docCount == 0 {
				return fmt.Errorf("unsupported file format or invalid JSON in %q: %v", filePath, err)
			}
			return fmt.Errorf("parse error in %q at document %d: %w", filePath, docCount+1, err)
		}

		if err := p.addDocument(doc); err != nil {
			return err
		}
		docCount++
	}

	p.sortFields()

	return nil
}

func (p *Parser) addJSONLinesFile(filePath string, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	var docCount int
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var doc any
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return fmt.Errorf("parse error in %q at line %d: %w", filePath, lineNumber, err)
		}

		if err := p.addDocument(doc); err != nil {
			return err
		}
		docCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read JSON lines file %q: %w", filePath, err)
	}

	if docCount == 0 {
		return fmt.Errorf("file %q is empty", filePath)
	}

	p.sortFields()

	return nil
}

func (p *Parser) addYAMLFile(filePath string, reader io.Reader) error {
	decoder := yaml.NewDecoder(reader)
	var docCount int

	for {
		var doc any

		err := decoder.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				if docCount == 0 {
					return fmt.Errorf("file %q is empty", filePath)
				}
				break
			}

			if docCount == 0 {
				return fmt.Errorf("unsupported file format or invalid YAML in %q: %v", filePath, err)
			}
			return fmt.Errorf("parse error in %q at document %d: %w", filePath, docCount+1, err)
		}

		if doc == nil {
			continue
		}

		if err := p.addDocument(doc); err != nil {
			return err
		}
		docCount++
	}

	p.sortFields()

	return nil
}

func (p *Parser) addDelimitedFile(filePath string, reader io.Reader, comma rune) error {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = comma
	csvReader.FieldsPerRecord = -1

	headers, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("file %q is empty", filePath)
		}

		return fmt.Errorf("parse header in %q: %w", filePath, err)
	}
	headers = normalizeHeaders(headers)

	var rowCount int
	for {
		row, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return fmt.Errorf("parse row %d in %q: %w", rowCount+2, filePath, err)
		}

		doc := rowToDocument(headers, row)
		if err := p.addDocument(doc); err != nil {
			return err
		}
		rowCount++
	}

	if rowCount == 0 {
		return fmt.Errorf("file %q has no data rows", filePath)
	}

	p.sortFields()

	return nil
}

func normalizeHeaders(headers []string) []string {
	normalized := make([]string, len(headers))
	seen := make(map[string]int, len(headers))

	for i, header := range headers {
		name := strings.TrimSpace(strings.TrimPrefix(header, "\ufeff"))
		if name == "" {
			name = fmt.Sprintf("column_%d", i+1)
		}

		count := seen[name]
		seen[name] = count + 1
		if count > 0 {
			name = fmt.Sprintf("%s_%d", name, count+1)
		}

		normalized[i] = name
	}

	return normalized
}

func rowToDocument(headers []string, row []string) map[string]any {
	doc := make(map[string]any)
	for i, header := range headers {
		if i >= len(row) {
			doc[header] = ""
			continue
		}

		doc[header] = row[i]
	}

	for i := len(headers); i < len(row); i++ {
		doc[fmt.Sprintf("extra_%d", i-len(headers)+1)] = row[i]
	}

	return doc
}

func (p *Parser) addDocument(doc any) error {
	doc = normalizeDataValue(doc)
	p.Docs = append(p.Docs, doc)

	paths, err := runJQ("paths", doc)
	if err != nil {
		return err
	}

	for _, path := range paths {
		pathString := jqPathToGenericString(path)

		if _, exists := p.seenFields[pathString]; exists {
			continue
		}

		p.seenFields[pathString] = struct{}{}
		p.Fields = append(p.Fields, Field{
			Path: pathString,
		})
	}

	return nil
}

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

func (p *Parser) sortFields() {
	sort.Slice(p.Fields, func(i, j int) bool {
		return p.Fields[i].Path < p.Fields[j].Path
	})
}

func (p *Parser) HandleSelection(selectedPath string, docs []any) ([]any, error) {
	query := genericPathToJQ(selectedPath)

	var values []any

	for _, doc := range docs {
		docValues, err := runJQ(query, doc)
		if err != nil {
			return nil, err
		}

		values = append(values, docValues...)
	}

	return values, nil
}

func genericPathToJQ(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return "."
	}

	if !strings.HasPrefix(path, ".") {
		path = "." + path
	}

	path = strings.ReplaceAll(path, "[]", "[]?")

	return path
}

func runJQ(queryText string, input any) ([]any, error) {
	query, err := gojq.Parse(queryText)
	if err != nil {
		return nil, fmt.Errorf("parse jq query %q: %w", queryText, err)
	}

	iter := query.Run(input)

	var results []any

	for {
		value, ok := iter.Next()
		if !ok {
			break
		}

		if err, ok := value.(error); ok {
			return nil, fmt.Errorf("run jq query %q: %w", queryText, err)
		}

		results = append(results, value)
	}

	return results, nil
}

func jqPathToGenericString(path any) string {
	parts, ok := path.([]any)
	if !ok {
		return "$"
	}

	var b strings.Builder
	b.WriteString("$")

	for _, part := range parts {
		switch v := part.(type) {
		case string:
			if isSimplePathKey(v) {
				b.WriteString(".")
				b.WriteString(v)
			} else {
				b.WriteString("[")
				b.WriteString(strconv.Quote(v))
				b.WriteString("]")
			}

		case int, float64:
			b.WriteString("[]")
		}
	}

	return b.String()
}

func isSimplePathKey(value string) bool {
	if value == "" {
		return false
	}

	for i, r := range value {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}

	return true
}
