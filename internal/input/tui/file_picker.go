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

	return fmt.Sprintf("File: %s\nSize: %s\nDocs: %d  Fields: %d", fileDesc, m.fileSizeStatus(), m.docCount, m.fieldCount)
}

func (m *Model) fileSizeStatus() string {
	if len(m.fileSizes) == 0 {
		return "none"
	}

	if len(m.fileSizes) == 1 {
		return formatByteSize(m.fileSizes[0])
	}

	return fmt.Sprintf(
		"%s total, latest %s",
		formatByteSize(totalFileSize(m.fileSizes)),
		formatByteSize(m.fileSizes[len(m.fileSizes)-1]),
	)
}

func totalFileSize(sizes []int64) int64 {
	var total int64
	for _, size := range sizes {
		total += size
	}

	return total
}

func formatByteSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%.1f PB", value/1024)
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
