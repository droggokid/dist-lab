package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	valueListDetailGap       = 2
	valueListDetailMaxHeight = 8
	valueListDetailMinHeight = 3
	valueSummaryMaxLength    = 120
)

type valueItem struct {
	index   int
	summary string
}

func (i valueItem) Title() string {
	return fmt.Sprintf("%d. %s", i.index+1, i.summary)
}

func (i valueItem) Description() string {
	return ""
}

func (i valueItem) FilterValue() string {
	return i.summary
}

type valueListModel struct {
	list *list.Model
}

func newValueListModel(values []any) valueListModel {
	items := make([]list.Item, len(values))
	for i, value := range values {
		items[i] = valueItem{
			index:   i,
			summary: summarizeValue(value),
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return valueListModel{
		list: &l,
	}
}

func (m valueListModel) Update(msg tea.Msg) (valueListModel, tea.Cmd) {
	if m.list == nil {
		return m, nil
	}

	newList, cmd := m.list.Update(msg)
	m.list = &newList
	return m, cmd
}

func (m valueListModel) View() string {
	if m.list == nil {
		return ""
	}

	return m.list.View()
}

func (m valueListModel) SelectedIndex() (int, bool) {
	if m.list == nil {
		return 0, false
	}

	item, ok := m.list.SelectedItem().(valueItem)
	if !ok {
		return 0, false
	}

	return item.index, true
}

func summarizeValue(value any) string {
	rendered, err := json.Marshal(value)
	if err != nil {
		rendered = []byte(fmt.Sprint(value))
	}

	summary := strings.ReplaceAll(string(rendered), "\n", " ")
	if len(summary) <= valueSummaryMaxLength {
		return summary
	}

	return summary[:valueSummaryMaxLength-3] + "..."
}

func (m *Model) rebuildValueList(selectIndex int) {
	m.valueList = newValueListModel(m.values)
	if m.valueList.list == nil {
		return
	}

	if len(m.values) == 0 {
		selectIndex = 0
	} else if selectIndex >= len(m.values) {
		selectIndex = len(m.values) - 1
	}

	if selectIndex < 0 {
		selectIndex = 0
	}

	m.valueList.list.Select(selectIndex)
	m.resizeValueList()
}

func (m *Model) resizeValueList() {
	if m.valueList.list == nil {
		return
	}

	contentHeight := m.previewContentHeight() - valuesHeaderHeight()
	detailHeight := m.valueDetailHeight(contentHeight)
	listHeight := contentHeight
	if detailHeight > 0 {
		listHeight -= detailHeight + valueListDetailGap
	}

	if listHeight < minContentHeight {
		listHeight = minContentHeight
	}

	m.valueList.list.SetWidth(m.contentWidth())
	m.valueList.list.SetHeight(listHeight)
}

func (m *Model) valueListContent() string {
	content := valuesHeader() + m.valueList.View()
	detail := m.valueDetailView(m.valueDetailHeight(m.previewContentHeight() - valuesHeaderHeight()))
	if detail == "" {
		return content
	}

	return content + "\n\n" + detail
}

func (m *Model) valueDetailHeight(contentHeight int) int {
	if contentHeight <= minContentHeight+valueListDetailGap+valueListDetailMinHeight {
		return 0
	}

	height := contentHeight / 3
	if height < valueListDetailMinHeight {
		height = valueListDetailMinHeight
	}
	if height > valueListDetailMaxHeight {
		height = valueListDetailMaxHeight
	}

	available := contentHeight - minContentHeight - valueListDetailGap
	if height > available {
		height = available
	}
	if height < valueListDetailMinHeight {
		return 0
	}

	return height
}

func (m *Model) valueDetailView(height int) string {
	if height <= 0 {
		return ""
	}

	index, ok := m.valueList.SelectedIndex()
	if !ok || index < 0 || index >= len(m.values) {
		return truncateLines("Selected\n  [none]", height)
	}

	return truncateLines("Selected\n"+formatValue(m.values[index]), height)
}

func truncateLines(value string, maxLines int) string {
	lines := strings.Split(value, "\n")
	if len(lines) <= maxLines {
		return value
	}

	lines = lines[:maxLines]
	lines[len(lines)-1] = "..."
	return strings.Join(lines, "\n")
}
