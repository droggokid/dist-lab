package tui

import (
	"fmt"
	"os"

	"dist-lab/internal/input"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	viewFilePicker viewState = iota
	viewFields
	viewPreview
	viewAnalysis
)

type Model struct {
	state viewState

	picker    filepicker.Model
	fields    fieldsModel
	preview   viewport.Model
	analysis  viewport.Model
	valueList valueListModel
	export    exportPromptModel

	previewMode previewMode

	filePaths      []string
	fileSizes      []int64
	selectedPath   string
	rawValues      []any
	values         []any
	valuesFiltered bool
	width          int
	height         int

	parser *input.Parser
	err    error
	notice string

	fieldCount int
	docCount   int
}

func NewModel() *Model {
	fp := filepicker.New()

	wd, err := os.Getwd()
	if err == nil {
		fp.CurrentDirectory = wd
	}

	fp.ShowHidden = true
	fp.FileAllowed = true
	fp.DirAllowed = false
	fp.AutoHeight = false
	fp.SetHeight(defaultContentHeight)

	return &Model{
		state:    viewFilePicker,
		picker:   fp,
		preview:  viewport.New(defaultViewWidth, defaultContentHeight),
		analysis: viewport.New(defaultViewWidth, defaultContentHeight),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.picker.Init(), tea.WindowSize())
}

func (m *Model) changeState(state viewState) {
	m.state = state
	m.err = nil
	m.notice = ""
	m.closeExportPrompt()
	m.resizeViews()
}

func (m *Model) setError(err error) {
	m.err = err
	m.notice = ""
	m.resizeViews()
}

func (m *Model) setNotice(notice string) {
	m.notice = notice
	m.err = nil
	m.resizeViews()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViews()

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.export.active {
			return m.updatePreview(msg)
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "o":
			m.parser = nil
			m.filePaths = nil
			m.fileSizes = nil
			m.clearValues()
			m.changeState(viewFilePicker)
			return m, m.picker.Init()

		case "a":
			m.changeState(viewFilePicker)
			return m, m.picker.Init()

		case "f":
			if (m.state == viewPreview || m.state == viewAnalysis) && m.parser != nil {
				m.fields = newFieldsModel(m.parser.Fields)
				m.changeState(viewFields)
				return m, m.fields.Init()
			}
		}
	}

	switch m.state {
	case viewFilePicker:
		return m.updateFilePicker(msg)

	case viewPreview:
		return m.updatePreview(msg)

	case viewAnalysis:
		return m.updateAnalysis(msg)

	case viewFields:
		return m.updateFields(msg)

	default:
		return m, nil
	}
}

func (m *Model) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.picker, cmd = m.picker.Update(msg)

	if didSelect, path := m.picker.DidSelectFile(msg); didSelect {
		if m.parser == nil {
			m.parser = input.NewParser()
			m.filePaths = []string{}
			m.fileSizes = []int64{}
			m.selectedPath = ""
			m.clearValues()
		}

		info, err := os.Stat(path)
		if err != nil {
			m.setError(fmt.Errorf("stat file %q: %w", path, err))
			return m, cmd
		}

		m.filePaths = append(m.filePaths, path)
		m.fileSizes = append(m.fileSizes, info.Size())

		if err := m.parser.AddFile(path); err != nil {
			m.setError(err)
			return m, cmd
		}

		if len(m.parser.Fields) == 0 {
			m.fieldCount = len(m.parser.Fields)
			m.docCount = len(m.parser.Docs)
			m.setError(fmt.Errorf("no fields found in combined files"))
			return m, cmd
		}

		m.fieldCount = len(m.parser.Fields)
		m.docCount = len(m.parser.Docs)
		m.fields = newFieldsModel(m.parser.Fields)
		m.changeState(viewFields)

		return m, tea.Batch(cmd, m.fields.Init())
	}

	return m, cmd
}

func (m *Model) updateFields(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.fields, cmd = m.fields.Update(msg)

	if m.fields.Completed() {
		m.selectedPath = m.fields.SelectedPath()

		values, err := m.parser.HandleSelection(m.selectedPath, m.parser.Docs)
		if err != nil {
			m.setError(err)
			return m, cmd
		}

		m.setValues(values)
		m.changeState(viewPreview)
		m.renderValues()
	}

	return m, cmd
}

func (m *Model) View() string {
	switch m.state {
	case viewFilePicker:
		return m.filePickerView()

	case viewPreview:
		return m.previewView()

	case viewAnalysis:
		return m.analysisView()

	case viewFields:
		return m.fieldsView()

	default:
		return ""
	}
}

func (m *Model) resizeViews() {
	m.resizeFilePicker()
	m.resizeFields()
	m.resizePreview()
	m.resizeAnalysis()
	m.resizeExportPrompt()
}
