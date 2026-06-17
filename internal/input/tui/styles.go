package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	defaultViewWidth     = 80
	defaultContentHeight = 12
	minContentHeight     = 3
	popupSpacing         = 2
	screenSectionGap     = "\n\n"
	screenTopPadding     = 1
	screenSectionCount   = 3
)

func (m *Model) screenView(header string, content string, footer string) string {
	return m.renderScreen(
		m.screenSections(
			m.headerView(header),
			m.contentView(content, header, footer),
			m.footerView(footer),
		),
	)
}

func (m *Model) renderScreen(sections []string) string {
	return strings.Repeat("\n", screenTopPadding) + strings.Join(sections, screenSectionGap)
}

func (m *Model) screenSections(sections ...string) []string {
	if !m.hasPopup() {
		return sections
	}

	withPopup := make([]string, 0, len(sections)+1)
	for i, section := range sections {
		withPopup = append(withPopup, section)
		if i == 0 {
			withPopup = append(withPopup, m.activePopup())
		}
	}

	return withPopup
}

func (m *Model) hasPopup() bool {
	return m.export.active || m.err != nil || m.notice != ""
}

func (m *Model) activePopup() string {
	if m.export.active {
		return m.exportPopup()
	}

	if m.err != nil {
		return m.errorPopup()
	}

	return m.noticePopup()
}

func (m *Model) errorPopup() string {
	return m.popupView(
		fmt.Sprintf("Error\n%v", m.err),
	)
}

func (m *Model) noticePopup() string {
	return m.popupView(
		fmt.Sprintf("Saved\n%s", m.notice),
	)
}

func (m *Model) popupView(content string) string {
	return m.borderedBlockWithStyle(content, lipgloss.RoundedBorder())
}

func (m *Model) headerView(content string) string {
	return m.borderedBlock(content)
}

func (m *Model) footerView(content string) string {
	return m.borderedBlock(content)
}

func (m *Model) contentView(content string, header string, footer string) string {
	return lipgloss.NewStyle().
		Width(m.contentWidth()).
		Height(m.contentHeight(header, footer)).
		Render(content)
}

func (m *Model) borderedBlock(content string) string {
	return m.borderedBlockWithStyle(content, lipgloss.NormalBorder())
}

func (m *Model) borderedBlockWithStyle(content string, border lipgloss.Border) string {
	return lipgloss.NewStyle().
		Border(border).
		Padding(0, 1).
		Width(m.borderedBlockWidth()).
		Render(content)
}

func (m *Model) borderedBlockWidth() int {
	width := m.width
	if width == 0 {
		width = defaultViewWidth
	}

	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	return contentWidth
}

func (m *Model) contentHeight(header string, footer string) int {
	if m.height == 0 {
		return defaultContentHeight
	}

	height := m.height -
		lipgloss.Height(m.headerView(header)) -
		lipgloss.Height(m.footerView(footer)) -
		screenTopPadding -
		screenChromeSpacing(screenSectionCount)

	if m.hasPopup() {
		height -= lipgloss.Height(m.activePopup()) + popupSpacing
	}

	if height < minContentHeight {
		height = minContentHeight
	}

	return height
}

func (m *Model) childContentHeight(header string, footer string, childChrome int) int {
	height := m.contentHeight(header, footer) - childChrome
	if height < minContentHeight {
		height = minContentHeight
	}

	return height
}

func (m *Model) contentWidth() int {
	if m.width == 0 {
		return defaultViewWidth
	}

	return m.width
}

func screenChromeSpacing(sectionCount int) int {
	if sectionCount <= 1 {
		return 0
	}

	return strings.Count(screenSectionGap, "\n") * (sectionCount - 1)
}
