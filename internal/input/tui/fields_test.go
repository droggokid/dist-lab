package tui

import (
	"testing"

	"dist-lab/internal/input"
)

func TestModelUpdateDoesNotStealFieldFilterText(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.state = viewFields
	m.fields = newFieldsModel([]input.Field{
		{Path: "$.alpha"},
		{Path: "$.beta"},
	})

	model, _ := m.Update(keyRunes("/"))
	updated := model.(*Model)
	if !updated.fields.Filtering() {
		t.Fatal("field filter should be active after /")
	}

	model, _ = updated.Update(keyRunes("a"))
	updated = model.(*Model)
	if updated.state != viewFields {
		t.Fatalf("state = %v, want fields", updated.state)
	}
	if !updated.fields.Filtering() {
		t.Fatal("field filter should remain active after typing a")
	}
}
