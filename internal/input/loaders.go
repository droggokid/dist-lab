package input

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

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
