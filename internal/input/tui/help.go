package tui

import "strings"

type helpSection struct {
	title string
	items []keyHelp
}

func (m *Model) openHelp() {
	if m.err != nil {
		return
	}

	m.helpActive = true
	m.notice = ""
	m.resizeViews()
}

func (m *Model) closeHelp() {
	m.helpActive = false
	m.resizeViews()
}

func (m *Model) clearTransientPopup() bool {
	switch {
	case m.err != nil:
		m.err = nil
	case m.notice != "":
		m.notice = ""
	default:
		return false
	}

	m.resizeViews()
	return true
}

func (m *Model) helpPopup() string {
	lines := []string{titleStyle.Render("Help"), m.helpContextLine()}
	for _, section := range m.helpSections() {
		lines = append(lines, "", labelStyle.Render(section.title))
		for _, item := range section.items {
			lines = append(lines, "  "+keyHelpView(item))
		}
	}

	lines = append(lines, "", helpFooter(
		keyHelp{key: "?", label: "close"},
		keyHelp{key: "esc", label: "close"},
		keyHelp{key: "q", label: "quit"},
	))

	return m.popupView(strings.Join(lines, "\n"))
}

func (m *Model) helpContextLine() string {
	switch m.state {
	case viewFilePicker:
		return "Choose a JSON, JSONL, YAML, CSV, or TSV file."
	case viewFields:
		return "Choose the field path to preview."
	case viewPreview:
		if m.previewMode == previewModeValues {
			return "Edit the current values before export or analysis."
		}
		return "Preview the current field values."
	case viewAnalysis:
		return "Inspect the current editable values."
	default:
		return ""
	}
}

func (m *Model) helpSections() []helpSection {
	switch m.state {
	case viewFilePicker:
		return []helpSection{
			{
				title: "File Picker",
				items: []keyHelp{
					{key: "up/down", label: "move"},
					{key: "enter", label: "select/open"},
					{key: "esc", label: "back"},
				},
			},
		}

	case viewFields:
		return []helpSection{
			{
				title: "Field Selection",
				items: []keyHelp{
					{key: "up/down", label: "move"},
					{key: "/", label: "filter"},
					{key: "enter", label: "select"},
					{key: "esc", label: "back"},
				},
			},
			m.fileHelpSection(),
		}

	case viewAnalysis:
		return []helpSection{
			{
				title: "Analysis",
				items: []keyHelp{
					{key: "1", label: "overview"},
					{key: "2", label: "missing"},
					{key: "3", label: "fields"},
					{key: "/", label: "filter fields"},
					{key: "n/N", label: "next/previous field"},
					{key: "enter", label: "focus field"},
				},
			},
			{
				title: "Navigation",
				items: []keyHelp{
					{key: "up/down", label: "scroll"},
					{key: "pgup/pgdn", label: "page"},
					{key: "p/esc", label: "preview"},
				},
			},
			m.fileHelpSection(),
		}

	default:
		if m.previewMode == previewModeValues {
			return []helpSection{
				{
					title: "Values",
					items: []keyHelp{
						{key: "up/down", label: "move"},
						{key: "d", label: "delete"},
						{key: "r", label: "restore"},
						{key: "v", label: "text preview"},
						m.emptyFilterHelp(),
					},
				},
				m.workflowHelpSection(),
				m.fileHelpSection(),
			}
		}

		return []helpSection{
			{
				title: "Preview",
				items: []keyHelp{
					{key: "up/down", label: "scroll"},
					{key: "pgup/pgdn", label: "page"},
					{key: "g/G", label: "top/bottom"},
					{key: "v", label: "editable list"},
					m.emptyFilterHelp(),
				},
			},
			m.workflowHelpSection(),
			m.fileHelpSection(),
		}
	}
}

func (m *Model) workflowHelpSection() helpSection {
	return helpSection{
		title: "Workflow",
		items: []keyHelp{
			{key: "i", label: "analysis"},
			{key: "x", label: "export"},
			{key: "f", label: "change field"},
		},
	}
}

func (m *Model) fileHelpSection() helpSection {
	return helpSection{
		title: "Files",
		items: []keyHelp{
			{key: "a", label: "add file"},
			{key: "o", label: "new file"},
		},
	}
}
