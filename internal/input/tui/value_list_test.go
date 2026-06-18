package tui

import (
	"strings"
	"testing"
)

func TestSummarizeValue(t *testing.T) {
	if got, want := summarizeValue(map[string]any{"name": "Ada"}), `{"name":"Ada"}`; got != want {
		t.Fatalf("summarizeValue(object) = %q, want %q", got, want)
	}

	long := summarizeValue(strings.Repeat("x", valueSummaryMaxLength+20))
	if len(long) != valueSummaryMaxLength {
		t.Fatalf("summarizeValue(long) length = %d, want %d", len(long), valueSummaryMaxLength)
	}
	if !strings.HasSuffix(long, "...") {
		t.Fatalf("summarizeValue(long) = %q, want ellipsis", long)
	}
}

func TestTruncateLines(t *testing.T) {
	got := truncateLines("a\nb\nc\nd", 3)
	if want := "a\nb\n..."; got != want {
		t.Fatalf("truncateLines() = %q, want %q", got, want)
	}

	if got := truncateLines("a\nb", 3); got != "a\nb" {
		t.Fatalf("truncateLines(short) = %q, want original", got)
	}
}
