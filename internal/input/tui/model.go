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
)

type Model struct {
	state viewState

	picker  filepicker.Model
	fields  fieldsModel
	preview viewport.Model

	filePath     string
	selectedPath string
	width        int
	height       int

	parser *input.JSONParser
	err    error

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
		state:   viewFilePicker,
		picker:  fp,
		preview: viewport.New(defaultViewWidth, defaultContentHeight),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.picker.Init(), tea.WindowSize())
}

func (m *Model) changeState(state viewState) {
	m.state = state
	m.err = nil
	m.resizeViews()
}

func (m *Model) setError(err error) {
	m.err = err
	m.resizeViews()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViews()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "o":
			m.changeState(viewFilePicker)
			return m, m.picker.Init()

		case "f":
			if m.state == viewPreview && m.parser != nil {
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
		m.filePath = path
		m.parser = nil
		m.selectedPath = ""
		m.fieldCount = 0
		m.docCount = 0
		m.err = nil
		m.preview.SetContent("")

		parser := input.NewParser(path)

		if err := parser.HandleDocument(); err != nil {
			m.setError(err)
			return m, cmd
		}

		if len(parser.Fields) == 0 {
			m.parser = nil
			m.fieldCount = 0
			m.docCount = len(parser.Docs)
			m.setError(fmt.Errorf("no fields found in %s", path))
			return m, cmd
		}

		m.parser = parser
		m.fieldCount = len(parser.Fields)
		m.docCount = len(parser.Docs)
		m.fields = newFieldsModel(parser.Fields)
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

		m.preview.SetContent(formatValues(values))
		m.preview.GotoTop()
		m.changeState(viewPreview)
	}

	return m, cmd
}

func (m *Model) View() string {
	switch m.state {
	case viewFilePicker:
		return m.filePickerView()

	case viewPreview:
		return m.previewView()

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
}
