package tui

import (
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"dist-lab/internal/input"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelStartsAtStartupChoice(t *testing.T) {
	m := NewModel()

	if m.state != viewStart {
		t.Fatalf("state = %v, want viewStart", m.state)
	}
}

func TestStartupChoiceRoutesToOpenOrCreate(t *testing.T) {
	m := NewModel()
	model, _ := m.Update(keyType(tea.KeyEnter))
	updated := model.(*Model)
	if updated.state != viewFilePicker {
		t.Fatalf("enter on default startup choice state = %v, want file picker", updated.state)
	}

	m = NewModel()
	model, _ = m.Update(keyRunes("c"))
	updated = model.(*Model)
	if updated.state != viewCreateDataset {
		t.Fatalf("c on startup state = %v, want create dataset", updated.state)
	}
}

func TestCreateDatasetWritesAndLoadsFile(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.changeState(viewCreateDataset)
	m.create = newCreateDatasetModel()
	m.create.rows.SetValue("3")
	m.create.path.SetValue(filepath.Join(t.TempDir(), "generated"))

	model, _ := m.updateCreateDataset(keyType(tea.KeyEnter))
	updated := model.(*Model)

	if updated.state != viewFields {
		t.Fatalf("state = %v, want fields", updated.state)
	}
	if len(updated.filePaths) != 1 {
		t.Fatalf("filePaths = %#v, want one generated file", updated.filePaths)
	}
	if filepath.Ext(updated.filePaths[0]) != ".json" {
		t.Fatalf("generated path = %q, want .json extension", updated.filePaths[0])
	}
	if updated.fieldCount == 0 {
		t.Fatal("generated file should discover fields")
	}
	if _, err := os.Stat(updated.filePaths[0]); err != nil {
		t.Fatalf("generated file missing: %v", err)
	}
}

func TestGenerateNumericDatasetRespectsBounds(t *testing.T) {
	values, err := generateDataset(datasetGenerationConfig{
		template:   generatedTemplateNumeric,
		numberKind: generatedNumberInteger,
		fieldName:  "score",
		rows:       "25",
		min:        "5",
		max:        "7",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset() error = %v", err)
	}
	if len(values) != 25 {
		t.Fatalf("len(values) = %d, want 25", len(values))
	}

	for _, value := range values {
		score, ok := value.(map[string]any)["score"].(int)
		if !ok {
			t.Fatalf("score type = %T, want int", value.(map[string]any)["score"])
		}
		if score < 5 || score > 7 {
			t.Fatalf("score = %d, want within 5..7", score)
		}
	}
}

func TestGenerateBooleanProbabilityEdges(t *testing.T) {
	falseValues, err := generateDataset(datasetGenerationConfig{
		template:        generatedTemplateBoolean,
		fieldName:       "active",
		rows:            "5",
		trueProbability: "0",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset(false) error = %v", err)
	}
	for _, value := range falseValues {
		if got := value.(map[string]any)["active"]; got != false {
			t.Fatalf("active = %v, want false", got)
		}
	}

	trueValues, err := generateDataset(datasetGenerationConfig{
		template:        generatedTemplateBoolean,
		fieldName:       "active",
		rows:            "5",
		trueProbability: "1",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset(true) error = %v", err)
	}
	for _, value := range trueValues {
		if got := value.(map[string]any)["active"]; got != true {
			t.Fatalf("active = %v, want true", got)
		}
	}
}

func TestGenerateCategoricalDatasetUsesChoicesAndWeights(t *testing.T) {
	values, err := generateDataset(datasetGenerationConfig{
		template:  generatedTemplateCategorical,
		fieldName: "kind",
		rows:      "10",
		choices:   "alpha,beta,gamma",
		weights:   "0,1,0",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset() error = %v", err)
	}

	for _, value := range values {
		if got := value.(map[string]any)["kind"]; got != "beta" {
			t.Fatalf("kind = %v, want beta", got)
		}
	}
}

func TestGenerateDatasetSupportsRandomRowInterval(t *testing.T) {
	values, err := generateDataset(datasetGenerationConfig{
		template:   generatedTemplateNumeric,
		numberKind: generatedNumberInteger,
		fieldName:  "value",
		rows:       "5..10",
		min:        "1",
		max:        "1",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset() error = %v", err)
	}

	if len(values) < 5 || len(values) > 10 {
		t.Fatalf("len(values) = %d, want within 5..10", len(values))
	}
}

func TestGenerateDatasetValidation(t *testing.T) {
	tests := []struct {
		name   string
		config datasetGenerationConfig
	}{
		{
			name: "empty field",
			config: datasetGenerationConfig{
				template: generatedTemplateBoolean,
				rows:     "1",
			},
		},
		{
			name: "bad rows",
			config: datasetGenerationConfig{
				template:  generatedTemplateBoolean,
				fieldName: "value",
				rows:      "0",
			},
		},
		{
			name: "bad numeric range",
			config: datasetGenerationConfig{
				template:   generatedTemplateNumeric,
				numberKind: generatedNumberInteger,
				fieldName:  "value",
				rows:       "1",
				min:        "10",
				max:        "1",
			},
		},
		{
			name: "bad boolean probability",
			config: datasetGenerationConfig{
				template:        generatedTemplateBoolean,
				fieldName:       "value",
				rows:            "1",
				trueProbability: "2",
			},
		},
		{
			name: "bad weights",
			config: datasetGenerationConfig{
				template:  generatedTemplateCategorical,
				fieldName: "value",
				rows:      "1",
				choices:   "a,b",
				weights:   "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := generateDataset(tt.config, rand.New(rand.NewSource(1))); err == nil {
				t.Fatal("generateDataset() error = nil, want error")
			}
		})
	}
}

func TestGeneratedFilesLoadThroughParser(t *testing.T) {
	values := []any{
		map[string]any{"value": 1},
		map[string]any{"value": 2},
	}

	for _, format := range exportFormats {
		t.Run(string(format), func(t *testing.T) {
			path, err := writeValues(filepath.Join(t.TempDir(), "generated"), format, values)
			if err != nil {
				t.Fatalf("writeValues() error = %v", err)
			}

			parser := input.NewParser()
			if err := parser.AddFile(path); err != nil {
				t.Fatalf("AddFile(%s) error = %v", format, err)
			}

			if len(parser.Fields) == 0 {
				t.Fatalf("parser fields empty for %s", format)
			}

			paths := make([]string, 0, len(parser.Fields))
			for _, field := range parser.Fields {
				paths = append(paths, field.Path)
			}
			if !containsValueField(paths) {
				t.Fatalf("fields = %#v, want a value field", paths)
			}
		})
	}
}

func TestParseGeneratedChoices(t *testing.T) {
	got := parseGeneratedChoices(" alpha, , beta,gamma ")
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseGeneratedChoices() = %#v, want %#v", got, want)
	}
}

func containsValueField(paths []string) bool {
	for _, path := range paths {
		if path == "$.value" || path == "$[].value" || strings.HasSuffix(path, ".value") {
			return true
		}
	}

	return false
}
