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
	viewStart viewState = iota
	viewFilePicker
	viewCreateDataset
	viewFields
	viewPreview
	viewAnalysis
)

type Model struct {
	state viewState

	startChoice startupChoice

	picker    filepicker.Model
	fields    fieldsModel
	preview   viewport.Model
	analysis  viewport.Model
	valueList valueListModel
	export    exportPromptModel
	create    createDatasetModel

	previewMode  previewMode
	analysisMode analysisMode

	helpActive bool

	analysisFilterActive bool
	analysisFilter       string
	analysisFieldIndex   int
	analysisFocusedField string

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
		state:       viewStart,
		picker:      fp,
		preview:     viewport.New(defaultViewWidth, defaultContentHeight),
		analysis:    viewport.New(defaultViewWidth, defaultContentHeight),
		create:      newCreateDatasetModel(),
		startChoice: startupChoiceOpen,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.picker.Init(), tea.WindowSize())
}

func (m *Model) changeState(state viewState) {
	m.state = state
	m.err = nil
	m.notice = ""
	m.helpActive = false
	m.closeExportPrompt()
	m.resizeViews()
}

func (m *Model) setError(err error) {
	m.err = err
	m.notice = ""
	m.helpActive = false
	m.resizeViews()
}

func (m *Model) setNotice(notice string) {
	m.notice = notice
	m.err = nil
	m.helpActive = false
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

		if m.helpActive {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "?", "esc":
				m.closeHelp()
				return m, nil
			default:
				return m, nil
			}
		}

		if m.state == viewStart {
			return m.updateStartup(msg)
		}

		if m.state == viewCreateDataset {
			return m.updateCreateDataset(msg)
		}

		if msg.String() == "?" {
			m.openHelp()
			return m, nil
		}

		if msg.String() == "esc" && m.clearTransientPopup() {
			return m, nil
		}

		if m.state == viewFields && m.fields.Filtering() {
			if msg.String() == "q" {
				return m, tea.Quit
			}
			break
		}

		if m.state == viewAnalysis && m.analysisFilterActive {
			if msg.String() == "q" {
				return m, tea.Quit
			}
			break
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "o":
			return m, m.openFilePicker(true)

		case "a":
			return m, m.openFilePicker(false)

		case "c":
			return m, m.openCreateDataset()

		case "f":
			if (m.state == viewPreview || m.state == viewAnalysis) && m.parser != nil {
				m.fields = newFieldsModel(m.parser.Fields)
				m.changeState(viewFields)
				return m, m.fields.Init()
			}
		}
	}

	switch m.state {
	case viewStart:
		return m.updateStartup(msg)

	case viewFilePicker:
		return m.updateFilePicker(msg)

	case viewCreateDataset:
		return m.updateCreateDataset(msg)

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
		if err := m.loadFile(path); err != nil {
			m.setError(err)
			return m, cmd
		}

		m.changeState(viewFields)

		return m, tea.Batch(cmd, m.fields.Init())
	}

	return m, cmd
}

func (m *Model) updateFields(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" && !m.fields.Filtering() {
		if m.selectedPath != "" {
			m.changeState(viewPreview)
			m.renderValues()
			return m, nil
		}

		m.parser = nil
		m.filePaths = nil
		m.fileSizes = nil
		m.clearValues()
		m.changeState(viewFilePicker)
		return m, m.picker.Init()
	}

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
	case viewStart:
		return m.startupView()

	case viewFilePicker:
		return m.filePickerView()

	case viewCreateDataset:
		return m.createDatasetView()

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
	m.resizeCreateDataset()
}

func (m *Model) openFilePicker(reset bool) tea.Cmd {
	if reset {
		m.resetLoadedData()
	}

	m.changeState(viewFilePicker)
	return m.picker.Init()
}

func (m *Model) openCreateDataset() tea.Cmd {
	m.create = newCreateDatasetModel()
	m.changeState(viewCreateDataset)
	return m.create.focusSelected()
}

func (m *Model) resetLoadedData() {
	m.parser = nil
	m.filePaths = nil
	m.fileSizes = nil
	m.selectedPath = ""
	m.fieldCount = 0
	m.docCount = 0
	m.fields = fieldsModel{}
	m.clearValues()
}

func (m *Model) ensureParser() {
	if m.parser != nil {
		return
	}

	m.parser = input.NewParser()
	m.filePaths = []string{}
	m.fileSizes = []int64{}
	m.selectedPath = ""
	m.clearValues()
}

func (m *Model) loadFile(path string) error {
	m.ensureParser()

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file %q: %w", path, err)
	}

	if err := m.parser.AddFile(path); err != nil {
		return err
	}

	m.filePaths = append(m.filePaths, path)
	m.fileSizes = append(m.fileSizes, info.Size())
	m.fieldCount = len(m.parser.Fields)
	m.docCount = len(m.parser.Docs)
	if m.fieldCount == 0 {
		return fmt.Errorf("no fields found in combined files")
	}

	m.fields = newFieldsModel(m.parser.Fields)
	return nil
}
