package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	OKStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	WarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
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
	Key    string
	Label  string
	Detail string
}

type SelectorModel struct {
	title    string
	items    []SelectItem
	cursor   int
	offset   int
	height   int
	selected map[int]bool
	quitting bool
	done     bool
}

const DefaultSelectorHeight = 10

func NewSelector(title string, items []SelectItem) SelectorModel {
	return SelectorModel{
		title:    title,
		items:    items,
		height:   DefaultSelectorHeight,
		selected: map[int]bool{},
	}
}

func (m SelectorModel) Init() tea.Cmd { return nil }

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = SelectorHeightForWindow(msg.Height)
		m = m.keepCursorVisible()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
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
		case "end":
			m.cursor = len(m.items) - 1
		case " ", "x":
			m.selected[m.cursor] = !m.selected[m.cursor]
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
	}
	m = m.keepCursorVisible()
	return m, nil
}

func (m SelectorModel) View() string {
	if m.quitting || m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("space toggles, a selects all, pgup/pgdown scroll, enter confirms, q exits"))
	b.WriteString("\n\n")
	visible := m.visibleCount()
	start := m.offset
	end := start + visible
	if end > len(m.items) {
		end = len(m.items)
	}
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
		b.WriteString(fmt.Sprintf("%s %s %-16s %s\n", cursor, box, item.Key, item.Label))
		if item.Detail != "" {
			b.WriteString("    " + DimStyle.Render(item.Detail) + "\n")
		}
	}
	if len(m.items) > visible {
		b.WriteString(DimStyle.Render(fmt.Sprintf("\nshowing %d-%d of %d", start+1, end, len(m.items))))
	}
	return b.String()
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
