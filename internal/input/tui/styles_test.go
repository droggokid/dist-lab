package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestContentHeightUsesAvailableScreenSpace(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30

	header := viewHeader("Preview", statusLine(statusItem{label: "File", value: "data.json"}))
	footer := helpFooter(keyHelp{key: "q", label: "quit"})

	want := m.height -
		lipgloss.Height(m.headerView(header)) -
		lipgloss.Height(m.footerView(footer)) -
		screenTopPadding -
		screenChromeSpacing(screenSectionCount)

	if got := m.contentHeight(header, footer); got != want {
		t.Fatalf("contentHeight() = %d, want %d", got, want)
	}
}

func TestContentHeightAccountsForPopup(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.err = errors.New("boom")

	header := viewHeader("Preview", statusLine(statusItem{label: "File", value: "data.json"}))
	footer := helpFooter(keyHelp{key: "q", label: "quit"})

	want := m.height -
		lipgloss.Height(m.headerView(header)) -
		lipgloss.Height(m.footerView(footer)) -
		screenTopPadding -
		screenChromeSpacing(screenSectionCount) -
		lipgloss.Height(m.activePopup()) -
		popupSpacing

	if got := m.contentHeight(header, footer); got != want {
		t.Fatalf("contentHeight() = %d, want %d", got, want)
	}
}

func TestContentHeightHasMinimum(t *testing.T) {
	m := NewModel()
	m.width = 40
	m.height = 1

	header := viewHeader("Preview")
	footer := helpFooter(keyHelp{key: "q", label: "quit"})

	if got := m.contentHeight(header, footer); got != minContentHeight {
		t.Fatalf("contentHeight() = %d, want minContentHeight %d", got, minContentHeight)
	}
}
