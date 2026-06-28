package tui

import (
	"dist-lab/internal/input"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type fieldItem string

func (i fieldItem) Title() string       { return string(i) }
func (i fieldItem) Description() string { return "" }
func (i fieldItem) FilterValue() string { return string(i) }

type fieldsModel struct {
	list     *list.Model
	selected string
	done     bool
}

func newFieldsModel(fields []input.Field) fieldsModel {
	items := make([]list.Item, len(fields))
	for i, field := range fields {
		items[i] = fieldItem(field.Path)
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // We will put the help in our own footer

	return fieldsModel{
		list: &l,
	}
}

func (m fieldsModel) Init() tea.Cmd {
	return nil
}

func (m fieldsModel) Update(msg tea.Msg) (fieldsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if i, ok := m.list.SelectedItem().(fieldItem); ok {
					m.selected = string(i)
					m.done = true
					return m, nil
				}
			}
		}
	}

	var cmd tea.Cmd
	newList, cmd := m.list.Update(msg)
	m.list = &newList
	return m, cmd
}

func (m fieldsModel) View() string {
	return m.list.View()
}

func (m fieldsModel) Completed() bool {
	return m.done
}

func (m fieldsModel) SelectedPath() string {
	return m.selected
}

func (m fieldsModel) Filtering() bool {
	return m.list != nil && m.list.FilterState() == list.Filtering
}

func (m *Model) fieldsView() string {
	return m.screenView(
		m.fieldsHeader(),
		m.fields.View(),
		m.fieldsFooter(),
	)
}

func (m *Model) fieldsHeader() string {
	return viewHeader("Field Selection", m.fileInfoStatus())
}

func (m *Model) fieldsFooter() string {
	return helpFooter(
		keyHelp{key: "up/down", label: "move"},
		keyHelp{key: "/", label: "filter"},
		keyHelp{key: "enter", label: "select"},
		keyHelp{key: "esc", label: "back"},
		keyHelp{key: "?", label: "help"},
		keyHelp{key: "q", label: "quit"},
	)
}

func (m *Model) resizeFields() {
	if m.fields.list == nil {
		return
	}

	m.fields.list.SetWidth(m.contentWidth())
	m.fields.list.SetHeight(m.childContentHeight(
		m.fieldsHeader(),
		m.fieldsFooter(),
		0,
	))
}
