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
	filePath string
	Fields   []Field
	Docs     []any
}

type Field struct {
	Path string
}

func NewParser(filePath string) *JSONParser {
	return &JSONParser{
		filePath: filePath,
	}
}

func (p *JSONParser) HandleDocument() error {
	file, err := os.Open(p.filePath)
	if err != nil {
		return fmt.Errorf("open file %q: %w", p.filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	seen := make(map[string]struct{})

	for {
		var doc any

		err := decoder.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("parse json %q: %w", p.filePath, err)
		}

		p.Docs = append(p.Docs, doc)

		paths, err := runJQ("paths", doc)
		if err != nil {
			return err
		}

		for _, path := range paths {
			pathString := jqPathToGenericString(path)

			if _, exists := seen[pathString]; exists {
				continue
			}

			seen[pathString] = struct{}{}
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

	if !strings.HasPrefix(strings.TrimPrefix(strings.TrimSpace(selectedPath), "$"), "[]") {
		query = ".[]? | " + query
	}

	values, err := runJQ(query, docs)
	if err != nil {
		return nil, err
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
