package input

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/itchyny/gojq"
)

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

	return optionalizeArrayWildcards(path)
}

func optionalizeArrayWildcards(path string) string {
	var b strings.Builder
	for i := 0; i < len(path); {
		switch path[i] {
		case '"':
			i = copyJQStringLiteral(&b, path, i)
		case '[':
			if i+1 < len(path) && path[i+1] == ']' {
				b.WriteString("[]?")
				i += 2
				continue
			}
			b.WriteByte(path[i])
			i++
		default:
			b.WriteByte(path[i])
			i++
		}
	}

	return b.String()
}

func copyJQStringLiteral(b *strings.Builder, value string, start int) int {
	i := start
	for i < len(value) {
		b.WriteByte(value[i])
		if value[i] == '\\' && i+1 < len(value) {
			i++
			b.WriteByte(value[i])
		} else if value[i] == '"' && i != start {
			i++
			break
		}
		i++
	}

	return i
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
