package input

import (
	"fmt"
	"strings"
)

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
