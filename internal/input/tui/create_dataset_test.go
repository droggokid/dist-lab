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

func TestGenerateListDatasetWrapsScalarGenerator(t *testing.T) {
	values, err := generateDataset(datasetGenerationConfig{
		template:        generatedTemplateList,
		elementKind:     generatedValueBoolean,
		fieldName:       "flags",
		rows:            "2",
		listLength:      "3",
		trueProbability: "1",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset() error = %v", err)
	}

	if len(values) != 2 {
		t.Fatalf("len(values) = %d, want 2", len(values))
	}
	for _, value := range values {
		flags, ok := value.(map[string]any)["flags"].([]any)
		if !ok {
			t.Fatalf("flags type = %T, want []any", value.(map[string]any)["flags"])
		}
		if len(flags) != 3 {
			t.Fatalf("len(flags) = %d, want 3", len(flags))
		}
		for _, flag := range flags {
			if flag != true {
				t.Fatalf("flag = %v, want true", flag)
			}
		}
	}
}

func TestGenerateMatrixDatasetWrapsScalarGenerator(t *testing.T) {
	values, err := generateDataset(datasetGenerationConfig{
		template:      generatedTemplateMatrix,
		elementKind:   generatedValueNumeric,
		numberKind:    generatedNumberInteger,
		fieldName:     "matrix",
		rows:          "2",
		matrixRows:    "2",
		matrixColumns: "3",
		min:           "1",
		max:           "1",
	}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("generateDataset() error = %v", err)
	}

	if len(values) != 2 {
		t.Fatalf("len(values) = %d, want 2", len(values))
	}
	for _, value := range values {
		matrix, ok := value.(map[string]any)["matrix"].([]any)
		if !ok {
			t.Fatalf("matrix type = %T, want []any", value.(map[string]any)["matrix"])
		}
		if len(matrix) != 2 {
			t.Fatalf("len(matrix) = %d, want 2", len(matrix))
		}
		for _, row := range matrix {
			cells, ok := row.([]any)
			if !ok {
				t.Fatalf("matrix row type = %T, want []any", row)
			}
			if len(cells) != 3 {
				t.Fatalf("len(matrix row) = %d, want 3", len(cells))
			}
			for _, cell := range cells {
				if cell != 1 {
					t.Fatalf("matrix cell = %v, want 1", cell)
				}
			}
		}
	}
}

func TestGenerateDatasetRejectsOversizedCompositeData(t *testing.T) {
	_, err := generateDataset(datasetGenerationConfig{
		template:      generatedTemplateMatrix,
		elementKind:   generatedValueNumeric,
		numberKind:    generatedNumberInteger,
		fieldName:     "matrix",
		rows:          "2",
		matrixRows:    "1000",
		matrixColumns: "1000",
		min:           "0",
		max:           "1",
	}, rand.New(rand.NewSource(1)))
	if err == nil {
		t.Fatal("generateDataset() error = nil, want cell budget error")
	}
}

func TestGeneratedCellBudgetBoundaries(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	spec, err := newDatasetGeneratorSpec(datasetGenerationConfig{
		template:    generatedTemplateList,
		elementKind: generatedValueNumeric,
		numberKind:  generatedNumberInteger,
		fieldName:   "values",
		rows:        "1000",
		listLength:  "1000",
		min:         "1",
		max:         "1",
	}, rng)
	if err != nil {
		t.Fatalf("newDatasetGeneratorSpec(exact cap) error = %v", err)
	}
	if spec.rowCount != 1000 || spec.cellsPerRow != 1000 {
		t.Fatalf("spec = %#v, want rowCount=1000 cellsPerRow=1000", spec)
	}

	_, err = newDatasetGeneratorSpec(datasetGenerationConfig{
		template:    generatedTemplateList,
		elementKind: generatedValueNumeric,
		numberKind:  generatedNumberInteger,
		fieldName:   "values",
		rows:        "1001",
		listLength:  "1000",
		min:         "1",
		max:         "1",
	}, rng)
	if err == nil {
		t.Fatal("newDatasetGeneratorSpec(one over cap) error = nil, want error")
	}

	_, err = newDatasetGeneratorSpec(datasetGenerationConfig{
		template:      generatedTemplateMatrix,
		elementKind:   generatedValueNumeric,
		numberKind:    generatedNumberInteger,
		fieldName:     "matrix",
		rows:          "1",
		matrixRows:    "1000001",
		matrixColumns: "2",
		min:           "1",
		max:           "1",
	}, rng)
	if err == nil {
		t.Fatal("newDatasetGeneratorSpec(huge matrix) error = nil, want error")
	}

	_, err = newDatasetGeneratorSpec(datasetGenerationConfig{
		template:    generatedTemplateList,
		elementKind: generatedValueNumeric,
		numberKind:  generatedNumberInteger,
		fieldName:   "values",
		rows:        "100000",
		listLength:  "11",
		min:         "1",
		max:         "1",
	}, rng)
	if err == nil {
		t.Fatal("newDatasetGeneratorSpec(huge row count composite) error = nil, want error")
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

func TestGeneratedCompositeFilesRoundTripThroughParser(t *testing.T) {
	tests := []struct {
		name       string
		config     datasetGenerationConfig
		wantString string
	}{
		{
			name: "list",
			config: datasetGenerationConfig{
				template:    generatedTemplateList,
				elementKind: generatedValueNumeric,
				numberKind:  generatedNumberInteger,
				fieldName:   "value",
				rows:        "2",
				listLength:  "3",
				min:         "1",
				max:         "1",
			},
			wantString: "[1,1,1]",
		},
		{
			name: "matrix",
			config: datasetGenerationConfig{
				template:      generatedTemplateMatrix,
				elementKind:   generatedValueNumeric,
				numberKind:    generatedNumberInteger,
				fieldName:     "value",
				rows:          "2",
				matrixRows:    "2",
				matrixColumns: "2",
				min:           "1",
				max:           "1",
			},
			wantString: "[[1,1],[1,1]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := generateDataset(tt.config, rand.New(rand.NewSource(1)))
			if err != nil {
				t.Fatalf("generateDataset() error = %v", err)
			}

			for _, format := range exportFormats {
				t.Run(string(format), func(t *testing.T) {
					path, err := writeValues(filepath.Join(t.TempDir(), "generated"), format, values)
					if err != nil {
						t.Fatalf("writeValues(%s) error = %v", format, err)
					}

					parser := input.NewParser()
					if err := parser.AddFile(path); err != nil {
						t.Fatalf("AddFile(%s) error = %v", format, err)
					}
					if len(parser.Fields) == 0 {
						t.Fatalf("parser fields empty for %s", format)
					}

					selected, err := parser.HandleSelection(generatedRoundTripPath(format), parser.Docs)
					if err != nil {
						t.Fatalf("HandleSelection(%s) error = %v", format, err)
					}
					if len(selected) != 2 {
						t.Fatalf("selected len = %d, want 2 for %s", len(selected), format)
					}

					switch format {
					case exportFormatCSV, exportFormatTSV:
						if selected[0] != tt.wantString {
							t.Fatalf("selected[0] = %#v, want %q for %s", selected[0], tt.wantString, format)
						}
					default:
						if _, ok := selected[0].([]any); !ok {
							t.Fatalf("selected[0] type = %T, want []any for %s", selected[0], format)
						}
					}
				})
			}
		})
	}
}

func generatedRoundTripPath(format exportFormat) string {
	switch format {
	case exportFormatJSON, exportFormatYAML:
		return "$[].value"
	default:
		return "$.value"
	}
}

func TestCreateDatasetVisibleRowsForCompositeTemplates(t *testing.T) {
	model := newCreateDatasetModel()
	model.template = generatedTemplateList
	model.elementKind = generatedValueCategorical

	want := []createDatasetRow{
		createRowTemplate,
		createRowFieldName,
		createRowRows,
		createRowElementKind,
		createRowListLength,
		createRowChoices,
		createRowWeights,
		createRowFormat,
		createRowPath,
	}
	if got := model.visibleRows(); !reflect.DeepEqual(got, want) {
		t.Fatalf("list visibleRows() = %#v, want %#v", got, want)
	}

	model.template = generatedTemplateMatrix
	model.elementKind = generatedValueNumeric

	want = []createDatasetRow{
		createRowTemplate,
		createRowFieldName,
		createRowRows,
		createRowElementKind,
		createRowMatrixRows,
		createRowMatrixColumns,
		createRowNumberKind,
		createRowMin,
		createRowMax,
		createRowDistribution,
		createRowFormat,
		createRowPath,
	}
	if got := model.visibleRows(); !reflect.DeepEqual(got, want) {
		t.Fatalf("matrix visibleRows() = %#v, want %#v", got, want)
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
