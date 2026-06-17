package input

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/itchyny/gojq"
)

type JSONParser struct {
	Fields     []Field
	Docs       []any
	seenFields map[string]struct{}
}

type Field struct {
	Path string
}

func NewParser() *JSONParser {
	return &JSONParser{
		seenFields: make(map[string]struct{}),
	}
}

func (p *JSONParser) AddFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
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

		docCount++
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
	}

	sort.Slice(p.Fields, func(i, j int) bool {
		return p.Fields[i].Path < p.Fields[j].Path
	})

	return nil
}

func (p *JSONParser) HandleSelection(selectedPath string, docs []any) ([]any, error) {
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
			b.WriteString(".")
			b.WriteString(v)

		case int, float64:
			b.WriteString("[]")
		}
	}

	return b.String()
}
