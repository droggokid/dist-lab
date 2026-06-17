package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func helpFooter(actions ...string) string {
	actions = append(actions, "q quit")
	return strings.Join(actions, "  ")
}

func (m *Model) fileInfoStatus() string {
	var fileDesc string
	if len(m.filePaths) == 0 {
		fileDesc = "none"
	} else if len(m.filePaths) == 1 {
		fileDesc = m.filePaths[0]
	} else {
		fileDesc = fmt.Sprintf("%d files (latest: %s)", len(m.filePaths), m.filePaths[len(m.filePaths)-1])
	}

	return fmt.Sprintf("File: %s\nDocs: %d  Fields: %d", fileDesc, m.docCount, m.fieldCount)
}

func (m *Model) filePickerView() string {
	content := m.picker.View()

	return m.screenView(
		m.filePickerHeader(),
		content,
		m.filePickerFooter(),
	)
}

func (m *Model) filePickerHeader() string {
	return "File Picker\nChoose a JSON file"
}

func (m *Model) filePickerFooter() string {
	return helpFooter("enter select/open", "esc back")
}

func (m *Model) resizeFilePicker() {
	height := m.childContentHeight(
		m.filePickerHeader(),
		m.filePickerFooter(),
		0,
	)

	m.picker.SetHeight(height)
	m.picker, _ = m.picker.Update(tea.WindowSizeMsg{
		Width:  m.contentWidth(),
		Height: height,
	})
}
