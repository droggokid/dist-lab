package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

func TestHelpPopupOpensAndClosesWithoutMutatingData(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.state = viewPreview
	m.selectedPath = "$.name"
	m.setValues([]any{"Ada", "Lin"})
	m.values = []any{"Ada"}

	beforeRaw := cloneValues(m.rawValues)
	beforeValues := cloneValues(m.values)

	model, _ := m.Update(keyRunes("?"))
	updated := model.(*Model)

	if !updated.helpActive {
		t.Fatal("help should be active after ?")
	}
	if updated.state != viewPreview {
		t.Fatalf("state = %v, want preview", updated.state)
	}
	if !reflect.DeepEqual(updated.rawValues, beforeRaw) {
		t.Fatalf("rawValues changed = %#v, want %#v", updated.rawValues, beforeRaw)
	}
	if !reflect.DeepEqual(updated.values, beforeValues) {
		t.Fatalf("values changed = %#v, want %#v", updated.values, beforeValues)
	}
	if !strings.Contains(ansi.Strip(updated.activePopup()), "Help") {
		t.Fatalf("active popup should render help:\n%s", updated.activePopup())
	}

	model, _ = updated.Update(keyType(tea.KeyEsc))
	updated = model.(*Model)

	if updated.helpActive {
		t.Fatal("help should be closed after esc")
	}
	if updated.state != viewPreview {
		t.Fatalf("state after closing help = %v, want preview", updated.state)
	}
}

func TestPreviewFooterKeepsAdvancedActionsInHelp(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.state = viewPreview
	m.setValues([]any{"Ada"})

	footer := ansi.Strip(m.previewFooterText())
	for _, unwanted := range []string{"add file", "new file", "change field"} {
		if strings.Contains(footer, unwanted) {
			t.Fatalf("preview footer should not show %q:\n%s", unwanted, footer)
		}
	}
	if !strings.Contains(footer, "help") {
		t.Fatalf("preview footer should expose help:\n%s", footer)
	}

	m.openHelp()
	help := ansi.Strip(m.activePopup())
	for _, want := range []string{"add file", "new file", "change field"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help popup missing %q:\n%s", want, help)
		}
	}
}

func TestFilePickerHeaderMatchesFieldSelectionChrome(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30

	filePickerHeader := m.headerView(m.filePickerHeader())
	fieldsHeader := m.headerView(m.fieldsHeader())

	if got, want := lipgloss.Height(filePickerHeader), lipgloss.Height(fieldsHeader); got != want {
		t.Fatalf("file picker header height = %d, want field selection header height %d", got, want)
	}
	if strings.Contains(ansi.Strip(filePickerHeader), "Formats") {
		t.Fatalf("file picker header should keep format help out of the main header:\n%s", filePickerHeader)
	}
}

func TestEscClosesNoticeBeforeNavigating(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.state = viewPreview
	m.setNotice("values.json")

	model, _ := m.Update(keyType(tea.KeyEsc))
	updated := model.(*Model)

	if updated.notice != "" {
		t.Fatalf("notice = %q, want cleared", updated.notice)
	}
	if updated.state != viewPreview {
		t.Fatalf("state = %v, want preview", updated.state)
	}
}
