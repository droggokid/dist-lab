package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type createDatasetRow int

const (
	createRowTemplate createDatasetRow = iota
	createRowFieldName
	createRowRows
	createRowElementKind
	createRowListLength
	createRowMatrixRows
	createRowMatrixColumns
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
	elementKind  generatedValueKind
	numberKind   generatedNumberKind
	distribution generatedNumericDistribution
	format       exportFormat

	fieldName       textinput.Model
	rows            textinput.Model
	listLength      textinput.Model
	matrixRows      textinput.Model
	matrixColumns   textinput.Model
	min             textinput.Model
	max             textinput.Model
	trueProbability textinput.Model
	choices         textinput.Model
	weights         textinput.Model
	path            textinput.Model
}

type datasetGenerationConfig struct {
	template     generatedTemplate
	elementKind  generatedValueKind
	numberKind   generatedNumberKind
	distribution generatedNumericDistribution
	format       exportFormat

	fieldName       string
	rows            string
	listLength      string
	matrixRows      string
	matrixColumns   string
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
		elementKind:  generatedValueNumeric,
		numberKind:   generatedNumberInteger,
		distribution: generatedNumericUniform,
		format:       exportFormatJSON,

		fieldName:       newCreateDatasetInput("value"),
		rows:            newCreateDatasetInput("100"),
		listLength:      newCreateDatasetInput("10"),
		matrixRows:      newCreateDatasetInput("3"),
		matrixColumns:   newCreateDatasetInput("3"),
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
	lines = append(lines, helpStyle.Render("Rows can be fixed like 100 or random like 50..250. Lists and matrices wrap scalar generators."), "")

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
		elementKind:     m.elementKind,
		numberKind:      m.numberKind,
		distribution:    m.distribution,
		format:          m.format,
		fieldName:       m.fieldName.Value(),
		rows:            m.rows.Value(),
		listLength:      m.listLength.Value(),
		matrixRows:      m.matrixRows.Value(),
		matrixColumns:   m.matrixColumns.Value(),
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
	case createRowElementKind:
		m.elementKind = nextGeneratedValueKind(m.elementKind, delta)
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
	case generatedTemplateList:
		rows = append(rows, createRowElementKind, createRowListLength)
		rows = append(rows, scalarRows(m.elementKind)...)
	case generatedTemplateMatrix:
		rows = append(rows, createRowElementKind, createRowMatrixRows, createRowMatrixColumns)
		rows = append(rows, scalarRows(m.elementKind)...)
	default:
		rows = append(rows, scalarRows(m.template.ValueKind())...)
	}

	return append(rows, createRowFormat, createRowPath)
}

func scalarRows(kind generatedValueKind) []createDatasetRow {
	switch kind {
	case generatedValueBoolean:
		return []createDatasetRow{createRowTrueProbability}
	case generatedValueCategorical:
		return []createDatasetRow{createRowChoices, createRowWeights}
	default:
		return []createDatasetRow{
			createRowNumberKind,
			createRowMin,
			createRowMax,
			createRowDistribution,
		}
	}
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
	case createRowListLength:
		return &m.listLength
	case createRowMatrixRows:
		return &m.matrixRows
	case createRowMatrixColumns:
		return &m.matrixColumns
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
		&m.listLength,
		&m.matrixRows,
		&m.matrixColumns,
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
	case createRowElementKind:
		return valueStyle.Render(m.elementKind.Label())
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
	case createRowElementKind:
		return "Element"
	case createRowListLength:
		return "List length"
	case createRowMatrixRows:
		return "Matrix rows"
	case createRowMatrixColumns:
		return "Matrix columns"
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
