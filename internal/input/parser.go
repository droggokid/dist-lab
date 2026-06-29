package input

import (
	"fmt"
	"os"
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
	loaded := NewParser()
	if err := loaded.loadFile(filePath); err != nil {
		return err
	}

	p.Merge(loaded)
	return nil
}

func (p *Parser) Merge(other *Parser) {
	if other == nil {
		return
	}
	if p.seenFields == nil {
		p.seenFields = make(map[string]struct{})
	}

	p.Docs = append(p.Docs, other.Docs...)
	for _, field := range other.Fields {
		if _, exists := p.seenFields[field.Path]; exists {
			continue
		}

		p.seenFields[field.Path] = struct{}{}
		p.Fields = append(p.Fields, field)
	}

	p.sortFields()
}

func (p *Parser) loadFile(filePath string) error {
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
