package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
