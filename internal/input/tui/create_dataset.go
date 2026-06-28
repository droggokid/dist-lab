package tui

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const maxGeneratedRows = 100000

type generatedTemplate int

const (
	generatedTemplateNumeric generatedTemplate = iota
	generatedTemplateBoolean
	generatedTemplateCategorical
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

type createDatasetRow int

const (
	createRowTemplate createDatasetRow = iota
	createRowFieldName
	createRowRows
	createRowNumberKind
	createRowMin
	createRowMax
	createRowDistribution
	createRowTrueProbability
	createRowChoices
	createRowWeights
	createRowFormat
	createRowPath
)

type createDatasetModel struct {
	selected createDatasetRow

	template     generatedTemplate
	numberKind   generatedNumberKind
	distribution generatedNumericDistribution
	format       exportFormat

	fieldName       textinput.Model
	rows            textinput.Model
	min             textinput.Model
	max             textinput.Model
	trueProbability textinput.Model
	choices         textinput.Model
	weights         textinput.Model
	path            textinput.Model
}

type datasetGenerationConfig struct {
	template     generatedTemplate
	numberKind   generatedNumberKind
	distribution generatedNumericDistribution
	format       exportFormat

	fieldName       string
	rows            string
	min             string
	max             string
	trueProbability string
	choices         string
	weights         string
	path            string
}

func newCreateDatasetModel() createDatasetModel {
	model := createDatasetModel{
		selected:     createRowTemplate,
		template:     generatedTemplateNumeric,
		numberKind:   generatedNumberInteger,
		distribution: generatedNumericUniform,
		format:       exportFormatJSON,

		fieldName:       newCreateDatasetInput("value"),
		rows:            newCreateDatasetInput("100"),
		min:             newCreateDatasetInput("0"),
		max:             newCreateDatasetInput("100"),
		trueProbability: newCreateDatasetInput("0.5"),
		choices:         newCreateDatasetInput("alpha,beta,gamma"),
		weights:         newCreateDatasetInput(""),
		path:            newCreateDatasetInput(defaultGeneratedPath("value", exportFormatJSON)),
	}
	model.path.Placeholder = "path/to/generated"
	model.weights.Placeholder = "optional, e.g. 3,1,1"

	return model
}

func newCreateDatasetInput(value string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.SetValue(value)
	input.Width = 40

	return input
}

func (m *Model) updateCreateDataset(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.String() == "esc" && m.clearTransientPopup() {
		return m, nil
	}
	if m.err != nil || m.notice != "" {
		return m, nil
	}

	switch key.String() {
	case "?":
		m.openHelp()
		return m, nil
	case "esc":
		m.changeState(viewStart)
		return m, nil
	case "up", "shift+tab":
		return m, m.create.moveSelection(-1)
	case "down", "tab":
		return m, m.create.moveSelection(1)
	case "left":
		if m.create.cycleSelection(-1) {
			return m, nil
		}
	case "right":
		if m.create.cycleSelection(1) {
			return m, nil
		}
	case "enter":
		return m, m.createGeneratedDataset()
	case "q":
		if m.create.selectedInput() == nil {
			return m, tea.Quit
		}
	}

	return m, m.create.updateSelectedInput(key)
}

func (m *Model) createGeneratedDataset() tea.Cmd {
	values, err := generateDataset(m.create.config(), nil)
	if err != nil {
		m.setError(err)
		return nil
	}

	path, err := writeValues(m.create.path.Value(), m.create.format, values)
	if err != nil {
		m.setError(err)
		return nil
	}

	m.resetLoadedData()
	if err := m.loadFile(path); err != nil {
		m.setError(err)
		return nil
	}

	m.changeState(viewFields)
	m.setNotice(path)

	return m.fields.Init()
}

func (m *Model) createDatasetView() string {
	return m.screenView(
		m.createDatasetHeader(),
		m.createDatasetContent(),
		m.createDatasetFooter(),
	)
}

func (m *Model) createDatasetHeader() string {
	return viewHeader(
		"Create Dataset",
		statusLine(
			statusItem{label: "Template", value: m.create.template.Label()},
			statusItem{label: "Format", value: strings.ToUpper(string(m.create.format))},
		),
		statusLine(statusItem{label: "Rows", value: m.create.rows.Value()}),
	)
}

func (m *Model) createDatasetContent() string {
	rows := m.create.visibleRows()
	lines := make([]string, 0, len(rows)+2)
	lines = append(lines, helpStyle.Render("Use 100 for a fixed row count or 50..250 for a random interval."), "")

	for _, row := range rows {
		marker := " "
		if row == m.create.selected {
			marker = ">"
		}

		label := fmt.Sprintf("%-16s", row.Label())
		lines = append(lines, fmt.Sprintf("%s %s %s", marker, labelStyle.Render(label), m.create.rowValue(row)))
	}

	return strings.Join(lines, "\n")
}

func (m *Model) createDatasetFooter() string {
	return helpFooter(
		keyHelp{key: "up/down", label: "field"},
		keyHelp{key: "left/right", label: "change"},
		keyHelp{key: "enter", label: "create"},
		keyHelp{key: "esc", label: "start"},
		keyHelp{key: "?", label: "help"},
		keyHelp{key: "q", label: "quit"},
	)
}

func (m *Model) resizeCreateDataset() {
	width := m.contentWidth() - 24
	if width < 20 {
		width = 20
	}

	m.create.setInputWidth(width)
}

func (m createDatasetModel) config() datasetGenerationConfig {
	return datasetGenerationConfig{
		template:        m.template,
		numberKind:      m.numberKind,
		distribution:    m.distribution,
		format:          m.format,
		fieldName:       m.fieldName.Value(),
		rows:            m.rows.Value(),
		min:             m.min.Value(),
		max:             m.max.Value(),
		trueProbability: m.trueProbability.Value(),
		choices:         m.choices.Value(),
		weights:         m.weights.Value(),
		path:            m.path.Value(),
	}
}

func (m *createDatasetModel) setInputWidth(width int) {
	for _, input := range m.inputs() {
		input.Width = width
	}
}

func (m *createDatasetModel) moveSelection(delta int) tea.Cmd {
	rows := m.visibleRows()
	index := m.selectedIndex(rows)
	index += delta
	if index < 0 {
		index = len(rows) - 1
	}
	if index >= len(rows) {
		index = 0
	}

	m.selected = rows[index]
	return m.focusSelected()
}

func (m *createDatasetModel) cycleSelection(delta int) bool {
	switch m.selected {
	case createRowTemplate:
		m.template = nextGeneratedTemplate(m.template, delta)
		return true
	case createRowNumberKind:
		m.numberKind = nextGeneratedNumberKind(m.numberKind, delta)
		return true
	case createRowDistribution:
		m.distribution = nextGeneratedNumericDistribution(m.distribution, delta)
		return true
	case createRowFormat:
		oldFormat := m.format
		m.format = adjacentExportFormat(m.format, delta)
		m.path.SetValue(swapExportExtension(m.path.Value(), oldFormat, m.format))
		return true
	default:
		return false
	}
}

func (m *createDatasetModel) updateSelectedInput(msg tea.Msg) tea.Cmd {
	input := m.selectedInput()
	if input == nil {
		return nil
	}

	updated, cmd := input.Update(msg)
	*input = updated
	return cmd
}

func (m *createDatasetModel) focusSelected() tea.Cmd {
	for _, input := range m.inputs() {
		input.Blur()
	}

	input := m.selectedInput()
	if input == nil {
		return nil
	}

	return input.Focus()
}

func (m *createDatasetModel) visibleRows() []createDatasetRow {
	rows := []createDatasetRow{
		createRowTemplate,
		createRowFieldName,
		createRowRows,
	}

	switch m.template {
	case generatedTemplateNumeric:
		rows = append(rows,
			createRowNumberKind,
			createRowMin,
			createRowMax,
			createRowDistribution,
		)
	case generatedTemplateBoolean:
		rows = append(rows, createRowTrueProbability)
	case generatedTemplateCategorical:
		rows = append(rows, createRowChoices, createRowWeights)
	}

	return append(rows, createRowFormat, createRowPath)
}

func (m *createDatasetModel) selectedIndex(rows []createDatasetRow) int {
	for i, row := range rows {
		if row == m.selected {
			return i
		}
	}

	m.selected = rows[0]
	return 0
}

func (m *createDatasetModel) selectedInput() *textinput.Model {
	return m.input(m.selected)
}

func (m *createDatasetModel) input(row createDatasetRow) *textinput.Model {
	switch row {
	case createRowFieldName:
		return &m.fieldName
	case createRowRows:
		return &m.rows
	case createRowMin:
		return &m.min
	case createRowMax:
		return &m.max
	case createRowTrueProbability:
		return &m.trueProbability
	case createRowChoices:
		return &m.choices
	case createRowWeights:
		return &m.weights
	case createRowPath:
		return &m.path
	default:
		return nil
	}
}

func (m *createDatasetModel) inputs() []*textinput.Model {
	return []*textinput.Model{
		&m.fieldName,
		&m.rows,
		&m.min,
		&m.max,
		&m.trueProbability,
		&m.choices,
		&m.weights,
		&m.path,
	}
}

func (m *createDatasetModel) rowValue(row createDatasetRow) string {
	if input := m.input(row); input != nil {
		return input.View()
	}

	switch row {
	case createRowTemplate:
		return valueStyle.Render(m.template.Label())
	case createRowNumberKind:
		return valueStyle.Render(m.numberKind.Label())
	case createRowDistribution:
		return valueStyle.Render(m.distribution.Label())
	case createRowFormat:
		return valueStyle.Render(strings.ToUpper(string(m.format)))
	default:
		return ""
	}
}

func (r createDatasetRow) Label() string {
	switch r {
	case createRowTemplate:
		return "Template"
	case createRowFieldName:
		return "Field"
	case createRowRows:
		return "Rows"
	case createRowNumberKind:
		return "Number kind"
	case createRowMin:
		return "Min"
	case createRowMax:
		return "Max"
	case createRowDistribution:
		return "Distribution"
	case createRowTrueProbability:
		return "True chance"
	case createRowChoices:
		return "Choices"
	case createRowWeights:
		return "Weights"
	case createRowFormat:
		return "Format"
	case createRowPath:
		return "Path"
	default:
		return ""
	}
}

func (t generatedTemplate) Label() string {
	switch t {
	case generatedTemplateBoolean:
		return "boolean"
	case generatedTemplateCategorical:
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

func generateDataset(config datasetGenerationConfig, rng *rand.Rand) ([]any, error) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	fieldName := strings.TrimSpace(config.fieldName)
	if fieldName == "" {
		return nil, fmt.Errorf("field name is required")
	}

	count, err := parseGeneratedRowCount(config.rows, rng)
	if err != nil {
		return nil, err
	}

	generator, err := valueGenerator(config, rng)
	if err != nil {
		return nil, err
	}

	values := make([]any, count)
	for i := range values {
		values[i] = map[string]any{
			fieldName: generator(),
		}
	}

	return values, nil
}

func valueGenerator(config datasetGenerationConfig, rng *rand.Rand) (func() any, error) {
	switch config.template {
	case generatedTemplateNumeric:
		return numericValueGenerator(config, rng)
	case generatedTemplateBoolean:
		return booleanValueGenerator(config, rng)
	case generatedTemplateCategorical:
		return categoricalValueGenerator(config, rng)
	default:
		return nil, fmt.Errorf("unsupported generated template")
	}
}

func numericValueGenerator(config datasetGenerationConfig, rng *rand.Rand) (func() any, error) {
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

func booleanValueGenerator(config datasetGenerationConfig, rng *rand.Rand) (func() any, error) {
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

func categoricalValueGenerator(config datasetGenerationConfig, rng *rand.Rand) (func() any, error) {
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

	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}

	return func() any {
		target := rng.Float64() * totalWeight
		seen := 0.0
		for i, weight := range weights {
			seen += weight
			if target <= seen {
				return choices[i]
			}
		}

		return choices[len(choices)-1]
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
