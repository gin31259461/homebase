package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m SelectorModel) View() string {
	if m.quitting || m.done {
		return ""
	}
	var b strings.Builder
	contentWidth := m.listContentWidth()
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")
	help := "j/k move, gg/G jump, / filters, space toggles, a selects matches, i inspects, ctrl+d/u scroll inspect, enter confirms, q exits"
	if m.filtering {
		help = "type to filter, backspace edits, ctrl+u clears, enter keeps filter, esc clears"
	}
	appendWrapped(&b, help, contentWidth, "", DimStyle)
	if m.filtering || m.filterText != "" {
		cursor := ""
		if m.filtering {
			cursor = "_"
		}
		filter := fmt.Sprintf("filter: %s%s · %d/%d items", m.filterText, cursor, len(m.matchingItemIndices()), len(m.items))
		appendWrapped(&b, filter, contentWidth, "", DimStyle)
	}
	b.WriteString("\n")
	m.appendList(&b, contentWidth)
	if _, ok := m.currentItemIndex(); m.inspect && ok {
		m.appendInspect(&b, contentWidth)
	}
	return b.String()
}

func (m SelectorModel) appendList(b *strings.Builder, contentWidth int) {
	indices := m.matchingItemIndices()
	visible := m.visibleCount()
	start := m.offset
	end := start + visible
	if end > len(indices) {
		end = len(indices)
	}

	list := selectorListView{
		builder:      b,
		contentWidth: contentWidth,
		keyWidth:     0,
		labelWidth:   0,
	}
	list.keyWidth, list.labelWidth = selectorColumnWidths(contentWidth)
	list.scrollbar = newScrollbarRenderer(scrollbarGeometryFor(
		m.listVisualLineCount(indices, 0, len(indices), contentWidth),
		m.listVisualLineCount(indices, start, end, contentWidth),
		m.listVisualLineCount(indices, 0, start, contentWidth),
		m.scrollbarConfig,
	))

	for position := start; position < end; position++ {
		list.appendItem(m, position, indices[position])
	}
	if len(indices) == 0 {
		b.WriteString(DimStyle.Render("No items match the current filter"))
		b.WriteString("\n")
	} else if len(indices) > visible {
		b.WriteString(DimStyle.Render(fmt.Sprintf("\nshowing %d-%d of %d", start+1, end, len(indices))))
	}
}

type selectorListView struct {
	builder      *strings.Builder
	contentWidth int
	keyWidth     int
	labelWidth   int
	scrollbar    scrollbarRenderer
}

func (v *selectorListView) appendItem(m SelectorModel, position, index int) {
	item := m.items[index]
	cursor := " "
	if position == m.cursor {
		cursor = ">"
	}
	box := "[ ]"
	if m.selected[index] {
		box = "[x]"
	}
	row := strings.Join([]string{
		cursor,
		box,
		padRight(truncateCells(item.Key, v.keyWidth), v.keyWidth),
		padRight(truncateCells(item.Label, v.labelWidth), v.labelWidth),
	}, " ")
	v.appendLine(row, "")
	if item.DetailValue != "" || item.Detail != "" {
		v.appendItemDetail(item)
	}
}

func (v *selectorListView) appendItemDetail(item SelectItem) {
	prefix := "    "
	if item.DetailValue == "" {
		v.appendWrapped(item.Detail, prefix, DimStyle)
		return
	}
	detail := item.DetailValue
	if item.Detail != "" {
		detail += " - " + item.Detail
	}
	lineWidth := v.contentWidth - lipgloss.Width(prefix)
	if lineWidth < 1 {
		lineWidth = 1
	}
	lines := wrapCells(detail, lineWidth)
	remainingValue := item.DetailValue
	for _, line := range lines {
		rendered := prefix + renderDetailLine(line, &remainingValue, item.State)
		v.appendLine(rendered, "")
	}
}

func (v *selectorListView) appendWrapped(text, prefix string, style lipgloss.Style) {
	lineWidth := v.contentWidth - lipgloss.Width(prefix)
	if lineWidth < 1 {
		lineWidth = 1
	}
	for _, line := range wrapCells(text, lineWidth) {
		v.appendLine(style.Render(prefix+line), "")
	}
}

func (v *selectorListView) appendLine(text, scrollbar string) {
	if scrollbar == "" {
		scrollbar = v.scrollbar.Next()
	}
	v.builder.WriteString(padRight(text, v.contentWidth))
	v.builder.WriteString(" ")
	v.builder.WriteString(scrollbar)
	v.builder.WriteString("\n")
}

func renderDetailLine(line string, remainingValue *string, state SelectState) string {
	if *remainingValue == "" {
		return DimStyle.Render(line)
	}
	if strings.HasPrefix(*remainingValue, line) {
		*remainingValue = strings.TrimPrefix(*remainingValue, line)
		return renderStateText(line, state)
	}
	if strings.HasPrefix(line, *remainingValue) {
		value := *remainingValue
		*remainingValue = ""
		return renderStateText(value, state) + DimStyle.Render(strings.TrimPrefix(line, value))
	}
	valuePart := commonPrefix(line, *remainingValue)
	if valuePart == "" {
		*remainingValue = ""
		return DimStyle.Render(line)
	}
	*remainingValue = strings.TrimPrefix(*remainingValue, valuePart)
	return renderStateText(valuePart, state) + DimStyle.Render(strings.TrimPrefix(line, valuePart))
}

func renderStateText(text string, state SelectState) string {
	switch state {
	case SelectStateGood:
		return OKStyle.Render(text)
	case SelectStatePartial:
		return WarnStyle.Render(text)
	case SelectStateBad:
		return BadStyle.Render(text)
	case SelectStateUnknown:
		return DimStyle.Render(text)
	default:
		return text
	}
}

func commonPrefix(a, b string) string {
	ar := []rune(a)
	br := []rune(b)
	limit := len(ar)
	if len(br) < limit {
		limit = len(br)
	}
	i := 0
	for i < limit && ar[i] == br[i] {
		i++
	}
	return string(ar[:i])
}

func (m SelectorModel) listVisualLineCount(indices []int, start, end, contentWidth int) int {
	if start < 0 {
		start = 0
	}
	if end > len(indices) {
		end = len(indices)
	}
	if start > end {
		start = end
	}
	lines := 0
	for _, index := range indices[start:end] {
		lines += selectorItemLineCount(m.items[index], contentWidth)
	}
	return lines
}

func selectorItemLineCount(item SelectItem, contentWidth int) int {
	lines := 1
	if item.DetailValue == "" && item.Detail == "" {
		return lines
	}
	prefixWidth := lipgloss.Width("    ")
	lineWidth := contentWidth - prefixWidth
	if lineWidth < 1 {
		lineWidth = 1
	}
	if item.DetailValue == "" {
		return lines + len(wrapCells(item.Detail, lineWidth))
	}
	detail := item.DetailValue
	if item.Detail != "" {
		detail += " - " + item.Detail
	}
	return lines + len(wrapCells(detail, lineWidth))
}

func (m SelectorModel) appendInspect(b *strings.Builder, contentWidth int) {
	index, ok := m.currentItemIndex()
	if !ok {
		return
	}
	inspect := strings.TrimSpace(m.items[index].Inspect)
	if inspect == "" {
		return
	}
	b.WriteString("\n\n")
	b.WriteString(TitleStyle.Render("Inspect " + m.items[index].Key))
	b.WriteString("\n")
	lines := strings.Split(inspect, "\n")
	lines = wrapLines(lines, contentWidth-2)
	m.inspectOffset = m.clampInspectOffset()
	inspectEnd := m.inspectOffset + InspectHeight
	if inspectEnd > len(lines) {
		inspectEnd = len(lines)
	}
	inspectLineWidth := contentWidth - 2
	if inspectLineWidth < 1 {
		inspectLineWidth = 1
	}
	scrollbar := newScrollbarRenderer(scrollbarGeometryFor(len(lines), InspectHeight, m.inspectOffset, m.scrollbarConfig))
	for _, line := range lines[m.inspectOffset:inspectEnd] {
		b.WriteString("  ")
		b.WriteString(padRight(line, inspectLineWidth))
		b.WriteString(" ")
		b.WriteString(scrollbar.Next())
		b.WriteString("\n")
	}
	for row := inspectEnd - m.inspectOffset; row < InspectHeight; row++ {
		b.WriteString("  ")
		b.WriteString(padRight("", inspectLineWidth))
		b.WriteString(" ")
		b.WriteString(scrollbar.Next())
		b.WriteString("\n")
	}
	if len(lines) > InspectHeight {
		b.WriteString(DimStyle.Render(fmt.Sprintf("showing %d-%d of %d", m.inspectOffset+1, inspectEnd, len(lines))))
		b.WriteString("\n")
	}
}

func appendWrapped(b *strings.Builder, text string, width int, prefix string, style lipgloss.Style) {
	lineWidth := width - lipgloss.Width(prefix)
	if lineWidth < 1 {
		lineWidth = 1
	}
	for _, line := range wrapCells(text, lineWidth) {
		rendered := style.Render(prefix + line)
		b.WriteString(padRight(rendered, width))
		b.WriteString(" ")
		b.WriteString(DimStyle.Render(" "))
		b.WriteString("\n")
	}
}

func (m SelectorModel) listContentWidth() int {
	width := m.width - 2
	if width < 20 {
		return 20
	}
	if width > 120 {
		return 120
	}
	return width
}

func selectorColumnWidths(width int) (int, int) {
	available := width - 7
	if available < 2 {
		return 1, 1
	}
	keyWidth := 16
	if available < 44 {
		keyWidth = available / 3
		if keyWidth < 6 {
			keyWidth = 6
		}
	}
	if keyWidth > available-1 {
		keyWidth = available - 1
	}
	return keyWidth, available - keyWidth
}

func (m SelectorModel) clampInspectOffset() int {
	index, ok := m.currentItemIndex()
	if !m.inspect || !ok {
		return 0
	}
	lines := wrapLines(strings.Split(strings.TrimSpace(m.items[index].Inspect), "\n"), m.listContentWidth()-2)
	maxOffset := len(lines) - InspectHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.inspectOffset > maxOffset {
		return maxOffset
	}
	if m.inspectOffset < 0 {
		return 0
	}
	return m.inspectOffset
}
