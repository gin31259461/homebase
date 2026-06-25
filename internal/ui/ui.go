package ui

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	OKStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	WarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	BadStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	DimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func Section(text string) {
	fmt.Printf("\n%s\n", TitleStyle.Render(text))
}

func OK(text string) {
	fmt.Printf("%s %s\n", OKStyle.Render("OK"), text)
}

func Warn(text string) {
	fmt.Printf("%s %s\n", WarnStyle.Render("WARN"), text)
}

func Note(text string) {
	fmt.Printf("%s %s\n", DimStyle.Render("NOTE"), text)
}

func Confirm(question string, def bool) bool {
	suffix := " [y/N]: "
	if def {
		suffix = " [Y/n]: "
	}
	fmt.Print(question + suffix)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

func PromptText(label, def string) string {
	finalModel, err := tea.NewProgram(newTextPrompt(label, def)).Run()
	if err == nil {
		model := finalModel.(textPromptModel)
		if model.canceled {
			return def
		}
		value := strings.TrimSpace(model.input.Value())
		if value == "" {
			return def
		}
		return value
	}
	return promptTextFallback(label, def)
}

func promptTextFallback(label, def string) string {
	if def == "" {
		fmt.Print(label + ": ")
	} else {
		fmt.Printf("%s [%s]: ", label, def)
	}
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

type textPromptModel struct {
	label    string
	def      string
	input    textinput.Model
	done     bool
	canceled bool
}

func newTextPrompt(label, def string) textPromptModel {
	input := textinput.New()
	input.SetValue(def)
	input.Placeholder = def
	input.Focus()
	input.CharLimit = 512
	input.Width = 72
	input.Prompt = "> "
	return textPromptModel{
		label: label,
		def:   def,
		input: input,
	}
}

func (m textPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textPromptModel) View() string {
	if m.done || m.canceled {
		return ""
	}
	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.label))
	if m.def != "" {
		b.WriteString(" ")
		b.WriteString(DimStyle.Render("(enter keeps current value, esc cancels)"))
	}
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	return b.String()
}

func WithSpinner(message string, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	frames := []string{"|", "/", "-", "\\"}
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case err := <-done:
			clearSpinnerLine(message)
			if err != nil {
				return err
			}
			OK(message + " complete")
			return nil
		case <-ticker.C:
			fmt.Printf("\r%s %s", DimStyle.Render(frames[frame%len(frames)]), message)
			frame++
		}
	}
}

func clearSpinnerLine(message string) {
	width := len(message) + 8
	fmt.Printf("\r%s\r", strings.Repeat(" ", width))
}

type SelectItem struct {
	Key             string
	Label           string
	DetailValue     string
	Detail          string
	Inspect         string
	State           SelectState
	DefaultSelected bool
}

type SelectState string

const (
	SelectStateNone    SelectState = ""
	SelectStateUnknown SelectState = "unknown"
	SelectStateGood    SelectState = "good"
	SelectStatePartial SelectState = "partial"
	SelectStateBad     SelectState = "bad"
)

type SelectorModel struct {
	title           string
	items           []SelectItem
	cursor          int
	offset          int
	height          int
	width           int
	selected        map[int]bool
	inspect         bool
	inspectOffset   int
	pendingG        bool
	quitting        bool
	done            bool
	scrollbarConfig ScrollbarConfig
}

type ScrollbarConfig struct {
	ShowWhenContentFits bool
	MinThumbRatio       float64
	MaxThumbRatio       float64
}

const (
	DefaultSelectorHeight = 10
	DefaultSelectorWidth  = 96
	InspectHeight         = 8
	DefaultMinThumbRatio  = 0.25
	DefaultMaxThumbRatio  = 1
)

func NewSelector(title string, items []SelectItem) SelectorModel {
	m := SelectorModel{
		title:           title,
		items:           items,
		height:          DefaultSelectorHeight,
		width:           DefaultSelectorWidth,
		selected:        map[int]bool{},
		scrollbarConfig: DefaultScrollbarConfig(),
	}
	for i, item := range items {
		if item.DefaultSelected {
			m.selected[i] = true
		}
	}
	return m
}

func NewSelectorWithScrollbar(title string, items []SelectItem, scrollbar ScrollbarConfig) SelectorModel {
	m := NewSelector(title, items)
	m.scrollbarConfig = scrollbar.normalized()
	return m
}

func DefaultScrollbarConfig() ScrollbarConfig {
	return ScrollbarConfig{
		MinThumbRatio: DefaultMinThumbRatio,
		MaxThumbRatio: DefaultMaxThumbRatio,
	}
}

func (c ScrollbarConfig) normalized() ScrollbarConfig {
	if c.MinThumbRatio <= 0 {
		c.MinThumbRatio = DefaultMinThumbRatio
	}
	if c.MaxThumbRatio <= 0 {
		c.MaxThumbRatio = DefaultMaxThumbRatio
	}
	if c.MinThumbRatio > 1 {
		c.MinThumbRatio = 1
	}
	if c.MaxThumbRatio > 1 {
		c.MaxThumbRatio = 1
	}
	if c.MinThumbRatio > c.MaxThumbRatio {
		c.MinThumbRatio = c.MaxThumbRatio
	}
	return c
}

func (m SelectorModel) Init() tea.Cmd { return nil }

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = SelectorHeightForWindow(msg.Height)
		if msg.Width > 0 {
			m.width = msg.Width
		}
		m = m.keepCursorVisible()
	case tea.KeyMsg:
		key := msg.String()
		oldCursor := m.cursor
		switch key {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+u":
			if m.inspect {
				m.inspectOffset -= InspectHeight / 2
				if m.inspectOffset < 0 {
					m.inspectOffset = 0
				}
				return m, nil
			}
		case "ctrl+d":
			if m.inspect {
				m.inspectOffset += InspectHeight / 2
				m.inspectOffset = m.clampInspectOffset()
				return m, nil
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "pgup":
			m.cursor -= m.visibleCount()
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown":
			m.cursor += m.visibleCount()
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
		case "home":
			m.cursor = 0
		case "end", "G", "shift+g":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
			}
		case "g":
			if m.pendingG {
				m.cursor = 0
				m.pendingG = false
			} else {
				m.pendingG = true
				return m, nil
			}
		case " ", "x":
			if len(m.items) > 0 {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "i":
			if len(m.items) > 0 && strings.TrimSpace(m.items[m.cursor].Inspect) != "" {
				m.inspect = !m.inspect
				m.inspectOffset = 0
			}
		case "a":
			all := len(m.selected) != len(m.items)
			m.selected = map[int]bool{}
			if all {
				for i := range m.items {
					m.selected[i] = true
				}
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
		if key != "g" {
			m.pendingG = false
		}
		if m.cursor != oldCursor {
			m.inspectOffset = 0
		}
	}
	m = m.keepCursorVisible()
	m.inspectOffset = m.clampInspectOffset()
	return m, nil
}

func (m SelectorModel) View() string {
	if m.quitting || m.done {
		return ""
	}
	var b strings.Builder
	contentWidth := m.listContentWidth()
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")
	help := "j/k move, gg/G jump, space toggles, a selects all, i inspects, ctrl+d/u scroll inspect, enter confirms, q exits"
	appendWrapped(&b, help, contentWidth, "", DimStyle)
	b.WriteString("\n")
	visible := m.visibleCount()
	start := m.offset
	end := start + visible
	if end > len(m.items) {
		end = len(m.items)
	}
	keyWidth, labelWidth := selectorColumnWidths(contentWidth)
	for i := start; i < end; i++ {
		item := m.items[i]
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		box := "[ ]"
		if m.selected[i] {
			box = "[x]"
		}
		scroll := m.scrollbar(i-start, visible)
		row := strings.Join([]string{
			cursor,
			box,
			padRight(truncateCells(item.Key, keyWidth), keyWidth),
			padRight(truncateCells(item.Label, labelWidth), labelWidth),
		}, " ")
		b.WriteString(padRight(row, contentWidth))
		b.WriteString(" ")
		b.WriteString(scroll)
		b.WriteString("\n")
		if item.DetailValue != "" || item.Detail != "" {
			appendItemDetail(&b, item, contentWidth)
		}
	}
	if len(m.items) > visible {
		b.WriteString(DimStyle.Render(fmt.Sprintf("\nshowing %d-%d of %d", start+1, end, len(m.items))))
	}
	if m.inspect && len(m.items) > 0 {
		inspect := strings.TrimSpace(m.items[m.cursor].Inspect)
		if inspect != "" {
			b.WriteString("\n\n")
			b.WriteString(TitleStyle.Render("Inspect " + m.items[m.cursor].Key))
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
			for row, line := range lines[m.inspectOffset:inspectEnd] {
				b.WriteString("  ")
				b.WriteString(padRight(line, inspectLineWidth))
				b.WriteString(" ")
				b.WriteString(m.inspectScrollbar(row, InspectHeight, len(lines)))
				b.WriteString("\n")
			}
			for row := inspectEnd - m.inspectOffset; row < InspectHeight; row++ {
				b.WriteString("  ")
				b.WriteString(padRight("", inspectLineWidth))
				b.WriteString(" ")
				b.WriteString(m.inspectScrollbar(row, InspectHeight, len(lines)))
				b.WriteString("\n")
			}
			if len(lines) > InspectHeight {
				b.WriteString(DimStyle.Render(fmt.Sprintf("showing %d-%d of %d", m.inspectOffset+1, inspectEnd, len(lines))))
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func appendItemDetail(b *strings.Builder, item SelectItem, width int) {
	prefix := "    "
	if item.DetailValue == "" {
		appendWrapped(b, item.Detail, width, prefix, DimStyle)
		return
	}
	detail := item.DetailValue
	if item.Detail != "" {
		detail += " - " + item.Detail
	}
	lineWidth := width - lipgloss.Width(prefix)
	if lineWidth < 1 {
		lineWidth = 1
	}
	lines := wrapCells(detail, lineWidth)
	remainingValue := item.DetailValue
	for _, line := range lines {
		rendered := prefix + renderDetailLine(line, &remainingValue, item.State)
		b.WriteString(padRight(rendered, width))
		b.WriteString(" ")
		b.WriteString(DimStyle.Render(" "))
		b.WriteString("\n")
	}
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

func (m SelectorModel) scrollbar(row, visible int) string {
	return renderScrollbar(row, scrollbarGeometryFor(len(m.items), visible, m.offset, m.scrollbarConfig))
}

func (m SelectorModel) inspectScrollbar(row, visible, total int) string {
	return renderScrollbar(row, scrollbarGeometryFor(total, visible, m.inspectOffset, m.scrollbarConfig))
}

type scrollbarGeometry struct {
	visible bool
	start   int
	length  int
}

func renderScrollbar(row int, geometry scrollbarGeometry) string {
	if !geometry.visible {
		return " "
	}
	if row >= geometry.start && row < geometry.start+geometry.length {
		return OKStyle.Render("|")
	}
	return DimStyle.Render(":")
}

func scrollbarGeometryFor(total, visible, offset int, config ScrollbarConfig) scrollbarGeometry {
	if visible <= 0 {
		return scrollbarGeometry{}
	}
	config = config.normalized()
	if total <= visible {
		if !config.ShowWhenContentFits {
			return scrollbarGeometry{}
		}
		return scrollbarGeometry{visible: true, length: visible}
	}

	track := visible
	minLength := ratioLength(track, config.MinThumbRatio, true)
	maxLengthCap := ratioLength(track, config.MaxThumbRatio, false)
	if maxLengthCap < minLength {
		maxLengthCap = minLength
	}
	naturalLength := int(math.Ceil(float64(track) * float64(visible) / float64(total)))
	if naturalLength < minLength {
		naturalLength = minLength
	}
	if naturalLength > maxLengthCap {
		naturalLength = maxLengthCap
	}

	maxOffset := total - visible
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	progress := float64(offset) / float64(maxOffset)
	middleFactor := 1 - math.Abs(progress*2-1)
	length := minLength + int(math.Round(float64(naturalLength-minLength)*middleFactor))
	if length < minLength {
		length = minLength
	}
	if length > naturalLength {
		length = naturalLength
	}

	startRange := track - length
	start := 0
	if startRange > 0 {
		start = int(math.Round(progress * float64(startRange)))
	}
	return scrollbarGeometry{visible: true, start: start, length: length}
}

func ratioLength(track int, ratio float64, roundUp bool) int {
	if track <= 0 {
		return 0
	}
	value := float64(track) * ratio
	length := int(math.Floor(value))
	if roundUp {
		length = int(math.Ceil(value))
	}
	if length < 1 {
		return 1
	}
	if length > track {
		return track
	}
	return length
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
	if !m.inspect || len(m.items) == 0 {
		return 0
	}
	lines := wrapLines(strings.Split(strings.TrimSpace(m.items[m.cursor].Inspect), "\n"), m.listContentWidth()-2)
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

func wrapLines(lines []string, width int) []string {
	var out []string
	for _, line := range lines {
		out = append(out, wrapCells(line, width)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapCells(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := ""
	for _, word := range words {
		parts := splitCells(word, width)
		for _, part := range parts {
			if line == "" {
				line = part
				continue
			}
			next := line + " " + part
			if lipgloss.Width(next) <= width {
				line = next
				continue
			}
			lines = append(lines, line)
			line = part
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func splitCells(text string, width int) []string {
	if lipgloss.Width(text) <= width {
		return []string{text}
	}
	var parts []string
	var current strings.Builder
	currentWidth := 0
	for _, r := range text {
		rw := lipgloss.Width(string(r))
		if currentWidth > 0 && currentWidth+rw > width {
			parts = append(parts, current.String())
			current.Reset()
			currentWidth = 0
		}
		current.WriteRune(r)
		currentWidth += rw
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func padRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	current := lipgloss.Width(s)
	if current >= width {
		return s
	}
	return s + strings.Repeat(" ", width-current)
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "."
}

func (m SelectorModel) SelectedKeys() []string {
	var keys []string
	for i, item := range m.items {
		if m.selected[i] {
			keys = append(keys, item.Key)
		}
	}
	return keys
}

func (m SelectorModel) Quitting() bool {
	return m.quitting
}

func (m SelectorModel) visibleCount() int {
	if m.height <= 0 {
		return DefaultSelectorHeight
	}
	return m.height
}

func (m SelectorModel) keepCursorVisible() SelectorModel {
	visible := m.visibleCount()
	if visible >= len(m.items) {
		m.offset = 0
		return m
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	maxOffset := len(m.items) - visible
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
	return m
}

func SelectorHeightForWindow(windowHeight int) int {
	if windowHeight <= 0 {
		return DefaultSelectorHeight
	}
	items := (windowHeight - 6) / 2
	if items < 4 {
		return 4
	}
	if items > DefaultSelectorHeight {
		return DefaultSelectorHeight
	}
	return items
}

func SelectKeys(title string, items []SelectItem) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	finalModel, err := tea.NewProgram(NewSelector(title, items)).Run()
	if err != nil {
		return NumberedSelect(title, items)
	}
	m := finalModel.(SelectorModel)
	if m.Quitting() {
		return nil, nil
	}
	return m.SelectedKeys(), nil
}
