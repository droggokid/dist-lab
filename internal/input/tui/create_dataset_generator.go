package tui

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxGeneratedRows  = 100000
	maxGeneratedCells = 1000000
)

type generatedTemplate int

const (
	generatedTemplateNumeric generatedTemplate = iota
	generatedTemplateBoolean
	generatedTemplateCategorical
	generatedTemplateList
	generatedTemplateMatrix
)

type generatedValueKind int

const (
	generatedValueNumeric generatedValueKind = iota
	generatedValueBoolean
	generatedValueCategorical
)

type generatedNumberKind int

const (
	generatedNumberInteger generatedNumberKind = iota
	generatedNumberDecimal
)

type generatedNumericDistribution int

const (
	generatedNumericUniform generatedNumericDistribution = iota
	generatedNumericNormal
)

type valueGenerator func() any

type datasetGeneratorSpec struct {
	fieldName   string
	rowCount    int
	cellsPerRow int
	value       valueGenerator
}

func generateDataset(config datasetGenerationConfig, rng *rand.Rand) ([]any, error) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	spec, err := newDatasetGeneratorSpec(config, rng)
	if err != nil {
		return nil, err
	}

	values := make([]any, spec.rowCount)
	for i := range values {
		values[i] = map[string]any{
			spec.fieldName: spec.value(),
		}
	}

	return values, nil
}

func newDatasetGeneratorSpec(config datasetGenerationConfig, rng *rand.Rand) (datasetGeneratorSpec, error) {
	fieldName := strings.TrimSpace(config.fieldName)
	if fieldName == "" {
		return datasetGeneratorSpec{}, fmt.Errorf("field name is required")
	}

	rowCount, err := parseGeneratedRowCount(config.rows, rng)
	if err != nil {
		return datasetGeneratorSpec{}, err
	}

	value, cellsPerRow, err := valueGeneratorForTemplate(config, rng)
	if err != nil {
		return datasetGeneratorSpec{}, err
	}

	if err := validateGeneratedCellBudget(rowCount, cellsPerRow); err != nil {
		return datasetGeneratorSpec{}, err
	}

	return datasetGeneratorSpec{
		fieldName:   fieldName,
		rowCount:    rowCount,
		cellsPerRow: cellsPerRow,
		value:       value,
	}, nil
}

func valueGeneratorForTemplate(config datasetGenerationConfig, rng *rand.Rand) (valueGenerator, int, error) {
	switch config.template {
	case generatedTemplateNumeric, generatedTemplateBoolean, generatedTemplateCategorical:
		value, err := scalarValueGenerator(config.template.ValueKind(), config, rng)
		return value, 1, err

	case generatedTemplateList:
		length, err := parseGeneratedPositiveInt("list length", config.listLength)
		if err != nil {
			return nil, 0, err
		}

		element, err := scalarValueGenerator(config.elementKind, config, rng)
		if err != nil {
			return nil, 0, err
		}

		return func() any {
			values := make([]any, length)
			for i := range values {
				values[i] = element()
			}
			return values
		}, length, nil

	case generatedTemplateMatrix:
		rowCount, err := parseGeneratedPositiveInt("matrix rows", config.matrixRows)
		if err != nil {
			return nil, 0, err
		}

		columnCount, err := parseGeneratedPositiveInt("matrix columns", config.matrixColumns)
		if err != nil {
			return nil, 0, err
		}

		cellsPerRow, err := generatedCellProduct(rowCount, columnCount)
		if err != nil {
			return nil, 0, err
		}

		element, err := scalarValueGenerator(config.elementKind, config, rng)
		if err != nil {
			return nil, 0, err
		}

		return func() any {
			matrix := make([]any, rowCount)
			for row := range matrix {
				values := make([]any, columnCount)
				for column := range values {
					values[column] = element()
				}
				matrix[row] = values
			}
			return matrix
		}, cellsPerRow, nil

	default:
		return nil, 0, fmt.Errorf("unsupported generated template")
	}
}

func generatedCellProduct(left int, right int) (int, error) {
	if err := validateGeneratedCellBudget(left, right); err != nil {
		return 0, err
	}

	return left * right, nil
}

func validateGeneratedCellBudget(rowCount int, cellsPerRow int) error {
	if rowCount < 1 {
		return fmt.Errorf("row count must be at least 1")
	}
	if cellsPerRow < 1 {
		return fmt.Errorf("generated cell count must be at least 1")
	}
	if cellsPerRow > maxGeneratedCells || rowCount > maxGeneratedCells/cellsPerRow {
		return fmt.Errorf("generated dataset cannot exceed %d scalar cells", maxGeneratedCells)
	}

	return nil
}

func scalarValueGenerator(kind generatedValueKind, config datasetGenerationConfig, rng *rand.Rand) (valueGenerator, error) {
	switch kind {
	case generatedValueNumeric:
		return numericValueGenerator(config, rng)
	case generatedValueBoolean:
		return booleanValueGenerator(config, rng)
	case generatedValueCategorical:
		return categoricalValueGenerator(config, rng)
	default:
		return nil, fmt.Errorf("unsupported generated value type")
	}
}

func numericValueGenerator(config datasetGenerationConfig, rng *rand.Rand) (valueGenerator, error) {
	minValue, err := strconv.ParseFloat(strings.TrimSpace(config.min), 64)
	if err != nil {
		return nil, fmt.Errorf("numeric min must be a number")
	}

	maxValue, err := strconv.ParseFloat(strings.TrimSpace(config.max), 64)
	if err != nil {
		return nil, fmt.Errorf("numeric max must be a number")
	}

	if minValue > maxValue {
		return nil, fmt.Errorf("numeric min cannot be greater than max")
	}

	if config.numberKind == generatedNumberInteger {
		minInt := int64(math.Ceil(minValue))
		maxInt := int64(math.Floor(maxValue))
		if minInt > maxInt {
			return nil, fmt.Errorf("numeric range contains no integer values")
		}

		return func() any {
			var value int64
			if config.distribution == generatedNumericNormal {
				value = int64(math.Round(boundedNormal(rng, float64(minInt), float64(maxInt))))
				if value < minInt {
					value = minInt
				}
				if value > maxInt {
					value = maxInt
				}
			} else if minInt == maxInt {
				value = minInt
			} else {
				value = minInt + rng.Int63n(maxInt-minInt+1)
			}

			return int(value)
		}, nil
	}

	return func() any {
		if config.distribution == generatedNumericNormal {
			return boundedNormal(rng, minValue, maxValue)
		}
		if minValue == maxValue {
			return minValue
		}

		return minValue + rng.Float64()*(maxValue-minValue)
	}, nil
}

func boundedNormal(rng *rand.Rand, minValue float64, maxValue float64) float64 {
	if minValue == maxValue {
		return minValue
	}

	mean := (minValue + maxValue) / 2
	stddev := (maxValue - minValue) / 6
	for i := 0; i < 64; i++ {
		value := rng.NormFloat64()*stddev + mean
		if value >= minValue && value <= maxValue {
			return value
		}
	}

	value := rng.NormFloat64()*stddev + mean
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}

	return value
}

func booleanValueGenerator(config datasetGenerationConfig, rng *rand.Rand) (valueGenerator, error) {
	probability, err := strconv.ParseFloat(strings.TrimSpace(config.trueProbability), 64)
	if err != nil {
		return nil, fmt.Errorf("true chance must be a number from 0 to 1")
	}

	if probability < 0 || probability > 1 {
		return nil, fmt.Errorf("true chance must be between 0 and 1")
	}

	return func() any {
		return rng.Float64() < probability
	}, nil
}

func categoricalValueGenerator(config datasetGenerationConfig, rng *rand.Rand) (valueGenerator, error) {
	choices := parseGeneratedChoices(config.choices)
	if len(choices) == 0 {
		return nil, fmt.Errorf("at least one categorical choice is required")
	}

	weights, err := parseGeneratedWeights(config.weights, len(choices))
	if err != nil {
		return nil, err
	}

	if len(weights) == 0 {
		return func() any {
			return choices[rng.Intn(len(choices))]
		}, nil
	}

	cumulative := make([]float64, len(weights))
	totalWeight := 0.0
	for i, weight := range weights {
		totalWeight += weight
		cumulative[i] = totalWeight
	}

	return func() any {
		target := rng.Float64() * totalWeight
		index := sort.Search(len(cumulative), func(i int) bool {
			return cumulative[i] > target
		})
		if index >= len(choices) {
			index = len(choices) - 1
		}

		return choices[index]
	}, nil
}

func parseGeneratedChoices(value string) []string {
	parts := strings.Split(value, ",")
	choices := make([]string, 0, len(parts))
	for _, part := range parts {
		choice := strings.TrimSpace(part)
		if choice != "" {
			choices = append(choices, choice)
		}
	}

	return choices
}

func parseGeneratedWeights(value string, choiceCount int) ([]float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	if len(parts) != choiceCount {
		return nil, fmt.Errorf("weights must match the number of choices")
	}

	weights := make([]float64, len(parts))
	total := 0.0
	for i, part := range parts {
		weight, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, fmt.Errorf("weights must be numbers")
		}
		if weight < 0 {
			return nil, fmt.Errorf("weights cannot be negative")
		}

		weights[i] = weight
		total += weight
	}

	if total == 0 {
		return nil, fmt.Errorf("at least one weight must be greater than zero")
	}

	return weights, nil
}

func parseGeneratedRowCount(value string, rng *rand.Rand) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("row count is required")
	}

	if strings.Contains(value, "..") {
		parts := strings.Split(value, "..")
		if len(parts) != 2 {
			return 0, fmt.Errorf("row interval must look like 50..250")
		}

		minCount, err := parseGeneratedRowCountInt(parts[0])
		if err != nil {
			return 0, err
		}

		maxCount, err := parseGeneratedRowCountInt(parts[1])
		if err != nil {
			return 0, err
		}

		if minCount > maxCount {
			return 0, fmt.Errorf("row interval min cannot be greater than max")
		}

		return minCount + rng.Intn(maxCount-minCount+1), nil
	}

	return parseGeneratedRowCountInt(value)
}

func parseGeneratedRowCountInt(value string) (int, error) {
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("row count must be a whole number or interval")
	}

	if count < 1 {
		return 0, fmt.Errorf("row count must be at least 1")
	}
	if count > maxGeneratedRows {
		return 0, fmt.Errorf("row count cannot exceed %d", maxGeneratedRows)
	}

	return count, nil
}

func parseGeneratedPositiveInt(label string, value string) (int, error) {
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be a whole number", label)
	}

	if count < 1 {
		return 0, fmt.Errorf("%s must be at least 1", label)
	}

	return count, nil
}

func (t generatedTemplate) Label() string {
	switch t {
	case generatedTemplateBoolean:
		return "boolean"
	case generatedTemplateCategorical:
		return "categorical"
	case generatedTemplateList:
		return "list"
	case generatedTemplateMatrix:
		return "matrix"
	default:
		return "numeric"
	}
}

func (t generatedTemplate) ValueKind() generatedValueKind {
	switch t {
	case generatedTemplateBoolean:
		return generatedValueBoolean
	case generatedTemplateCategorical:
		return generatedValueCategorical
	default:
		return generatedValueNumeric
	}
}

func (k generatedValueKind) Label() string {
	switch k {
	case generatedValueBoolean:
		return "boolean"
	case generatedValueCategorical:
		return "categorical"
	default:
		return "numeric"
	}
}

func (k generatedNumberKind) Label() string {
	switch k {
	case generatedNumberDecimal:
		return "decimal"
	default:
		return "integer"
	}
}

func (d generatedNumericDistribution) Label() string {
	switch d {
	case generatedNumericNormal:
		return "normal"
	default:
		return "uniform"
	}
}

func nextGeneratedTemplate(value generatedTemplate, delta int) generatedTemplate {
	values := []generatedTemplate{
		generatedTemplateNumeric,
		generatedTemplateBoolean,
		generatedTemplateCategorical,
		generatedTemplateList,
		generatedTemplateMatrix,
	}

	return values[wrappedIndex(int(value), delta, len(values))]
}

func nextGeneratedValueKind(value generatedValueKind, delta int) generatedValueKind {
	values := []generatedValueKind{
		generatedValueNumeric,
		generatedValueBoolean,
		generatedValueCategorical,
	}

	return values[wrappedIndex(int(value), delta, len(values))]
}

func nextGeneratedNumberKind(value generatedNumberKind, delta int) generatedNumberKind {
	values := []generatedNumberKind{
		generatedNumberInteger,
		generatedNumberDecimal,
	}

	return values[wrappedIndex(int(value), delta, len(values))]
}

func nextGeneratedNumericDistribution(value generatedNumericDistribution, delta int) generatedNumericDistribution {
	values := []generatedNumericDistribution{
		generatedNumericUniform,
		generatedNumericNormal,
	}

	return values[wrappedIndex(int(value), delta, len(values))]
}

func adjacentExportFormat(value exportFormat, delta int) exportFormat {
	for i, format := range exportFormats {
		if format == value {
			return exportFormats[wrappedIndex(i, delta, len(exportFormats))]
		}
	}

	return exportFormats[0]
}

func wrappedIndex(index int, delta int, count int) int {
	index += delta
	for index < 0 {
		index += count
	}

	return index % count
}

func defaultGeneratedPath(fieldName string, format exportFormat) string {
	name := sanitizeExportName(fieldName)
	if name == "" {
		name = "value"
	}

	return fmt.Sprintf("generated-%s.%s", name, exportFormatExtension(format))
}
