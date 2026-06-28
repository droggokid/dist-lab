package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestFilterEmptyValuesRecursive(t *testing.T) {
	values := []any{
		nil,
		"",
		"  ",
		"keep",
		map[string]any{
			"nil":         nil,
			"emptyString": "",
			"blank":       " ",
			"emptySlice":  []any{nil, ""},
			"emptyObject": map[string]any{"empty": ""},
			"list":        []any{nil, "x", map[string]any{"drop": "", "keep": "y"}},
			"number":      0,
			"false":       false,
		},
		[]any{nil, ""},
		[]any{nil, "z"},
	}

	want := []any{
		"keep",
	}

	if got := filterEmptyValues(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("filterEmptyValues() = %#v, want %#v", got, want)
	}
}

func TestFilterEmptyValuesDropsWholeObjectWithNestedEmpty(t *testing.T) {
	values := []any{
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 4, "month": 9, "year": nil},
		map[string]any{"day": 5, "month": 9, "year": 1996},
	}

	want := []any{
		map[string]any{"day": 3, "month": 9, "year": 2001},
		map[string]any{"day": 5, "month": 9, "year": 1996},
	}

	if got := filterEmptyValues(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("filterEmptyValues() = %#v, want %#v", got, want)
	}
}

func TestCloneValuesDoesNotShareNestedData(t *testing.T) {
	original := []any{
		map[string]any{
			"nested": []any{
				map[string]any{"name": "Ada"},
			},
		},
	}

	cloned := cloneValues(original)
	cloned[0].(map[string]any)["nested"].([]any)[0].(map[string]any)["name"] = "Lin"

	got := original[0].(map[string]any)["nested"].([]any)[0].(map[string]any)["name"]
	if got != "Ada" {
		t.Fatalf("original nested value = %q, want Ada", got)
	}
}

func TestDeleteSelectedValueAndRestore(t *testing.T) {
	m := NewModel()
	m.setValues([]any{"a", "b", "c"})
	m.previewMode = previewModeValues

	m.deleteSelectedValue()

	if want := []any{"b", "c"}; !reflect.DeepEqual(m.values, want) {
		t.Fatalf("values after delete = %#v, want %#v", m.values, want)
	}
	if want := []any{"a", "b", "c"}; !reflect.DeepEqual(m.rawValues, want) {
		t.Fatalf("rawValues after delete = %#v, want %#v", m.rawValues, want)
	}

	m.restoreValues()

	if want := []any{"a", "b", "c"}; !reflect.DeepEqual(m.values, want) {
		t.Fatalf("values after restore = %#v, want %#v", m.values, want)
	}
}

func TestRestoreValuesKeepsFilterState(t *testing.T) {
	m := NewModel()
	m.setValues([]any{"a", "", nil, map[string]any{"keep": "b", "drop": ""}})
	m.toggleEmptyValueFilter()

	if want := []any{"a"}; !reflect.DeepEqual(m.values, want) {
		t.Fatalf("filtered values = %#v, want %#v", m.values, want)
	}

	m.values = nil
	m.restoreValues()

	if want := []any{"a"}; !reflect.DeepEqual(m.values, want) {
		t.Fatalf("restored filtered values = %#v, want %#v", m.values, want)
	}
}

func TestFormatValues(t *testing.T) {
	got := formatValues([]any{map[string]any{"name": "Ada"}, "plain"})

	if !strings.Contains(got, "1. {") {
		t.Fatalf("formatValues() = %q, want first numbered object", got)
	}
	if !strings.Contains(got, "2. \"plain\"") {
		t.Fatalf("formatValues() = %q, want second numbered string", got)
	}

	if empty := formatValues(nil); empty != "    [none]" {
		t.Fatalf("formatValues(nil) = %q, want none marker", empty)
	}
}
