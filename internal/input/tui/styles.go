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

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248"))
	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("29")).
			Padding(0, 1)
	headerBorderColor = lipgloss.Color("62")
	footerBorderColor = lipgloss.Color("240")
	popupBorderColor  = lipgloss.Color("62")
	errorTitleStyle   = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("203"))
	successTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("42"))
)

type statusItem struct {
	label string
	value string
}

type keyHelp struct {
	key   string
	label string
}

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
		fmt.Sprintf("%s\n%v", errorTitleStyle.Render("Error"), m.err),
	)
}

func (m *Model) noticePopup() string {
	return m.popupView(
		fmt.Sprintf("%s\n%s", successTitleStyle.Render("Saved"), m.notice),
	)
}

func (m *Model) popupView(content string) string {
	return m.borderedBlockWithStyle(content, lipgloss.RoundedBorder(), popupBorderColor)
}

func (m *Model) headerView(content string) string {
	return m.borderedBlockWithStyle(content, lipgloss.NormalBorder(), headerBorderColor)
}

func (m *Model) footerView(content string) string {
	return m.borderedBlockWithStyle(content, lipgloss.NormalBorder(), footerBorderColor)
}

func (m *Model) contentView(content string, header string, footer string) string {
	return lipgloss.NewStyle().
		Width(m.contentWidth()).
		Height(m.contentHeight(header, footer)).
		Render(content)
}

func (m *Model) borderedBlockWithStyle(content string, border lipgloss.Border, borderColor lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
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

func viewHeader(title string, lines ...string) string {
	return viewHeaderTitle(titleStyle.Render(title), lines...)
}

func viewHeaderTitle(title string, lines ...string) string {
	parts := []string{title}
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts = append(parts, line)
	}

	return strings.Join(parts, "\n")
}

func statusLine(items ...statusItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s", labelStyle.Render(item.label+":"), valueStyle.Render(item.value)))
	}

	return strings.Join(parts, "  ")
}

func badge(value string) string {
	return badgeStyle.Render(value)
}

func helpFooter(items ...keyHelp) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, keyHelpView(item))
	}

	return strings.Join(parts, "  ")
}

func keyHelpView(item keyHelp) string {
	return fmt.Sprintf("%s %s", keyStyle.Render(item.key), helpStyle.Render(item.label))
}

func screenChromeSpacing(sectionCount int) int {
	if sectionCount <= 1 {
		return 0
	}

	return strings.Count(screenSectionGap, "\n") * (sectionCount - 1)
}
