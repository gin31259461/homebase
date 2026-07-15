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
	filterText      string
	filtering       bool
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
		if key == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		if m.filtering {
			m = m.updateFilter(msg)
			return m.keepCursorVisible(), nil
		}
		indices := m.matchingItemIndices()
		oldCursor := m.cursor
		switch key {
		case "esc":
			if m.filterText != "" {
				m = m.clearFilter()
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "/":
			m.filtering = true
			m.pendingG = false
			m.inspect = false
			m.inspectOffset = 0
			return m, nil
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
			if m.cursor < len(indices)-1 {
				m.cursor++
			}
		case "pgup":
			m.cursor -= m.visibleCount()
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown":
			m.cursor += m.visibleCount()
			if m.cursor >= len(indices) {
				m.cursor = len(indices) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		case "home":
			m.cursor = 0
		case "end", "G", "shift+g":
			if len(indices) > 0 {
				m.cursor = len(indices) - 1
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
			if index, ok := m.currentItemIndex(); ok {
				if m.selected[index] {
					delete(m.selected, index)
				} else {
					m.selected[index] = true
				}
			}
		case "i":
			if index, ok := m.currentItemIndex(); ok && strings.TrimSpace(m.items[index].Inspect) != "" {
				m.inspect = !m.inspect
				m.inspectOffset = 0
			}
		case "a":
			allSelected := len(indices) > 0
			for _, index := range indices {
				if !m.selected[index] {
					allSelected = false
					break
				}
			}
			for _, index := range indices {
				if allSelected {
					delete(m.selected, index)
				} else {
					m.selected[index] = true
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

func (m SelectorModel) updateFilter(msg tea.KeyMsg) SelectorModel {
	key := msg.String()
	switch key {
	case "esc":
		return m.clearFilter()
	case "enter":
		m.filtering = false
		return m
	case "backspace", "ctrl+h", "delete":
		runes := []rune(m.filterText)
		if len(runes) > 0 {
			m.filterText = string(runes[:len(runes)-1])
			return m.resetFilterPosition()
		}
	case "ctrl+u":
		m.filterText = ""
		return m.resetFilterPosition()
	default:
		if msg.Type == tea.KeyRunes {
			m.filterText += string(msg.Runes)
			return m.resetFilterPosition()
		}
		if key == " " {
			m.filterText += " "
			return m.resetFilterPosition()
		}
	}
	return m
}

func (m SelectorModel) clearFilter() SelectorModel {
	m.filterText = ""
	m.filtering = false
	return m.resetFilterPosition()
}

func (m SelectorModel) resetFilterPosition() SelectorModel {
	m.cursor = 0
	m.offset = 0
	m.inspect = false
	m.inspectOffset = 0
	m.pendingG = false
	return m
}

func (m SelectorModel) matchingItemIndices() []int {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(m.filterText)))
	indices := make([]int, 0, len(m.items))
	for i, item := range m.items {
		if itemMatchesFilter(item, terms) {
			indices = append(indices, i)
		}
	}
	return indices
}

func itemMatchesFilter(item SelectItem, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		item.Key,
		item.Label,
		item.DetailValue,
		item.Detail,
		item.Inspect,
	}, "\n"))
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func (m SelectorModel) currentItemIndex() (int, bool) {
	indices := m.matchingItemIndices()
	if m.cursor < 0 || m.cursor >= len(indices) {
		return 0, false
	}
	return indices[m.cursor], true
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
	total := len(m.matchingItemIndices())
	if total == 0 {
		m.cursor = 0
		m.offset = 0
		return m
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	if visible >= total {
		m.offset = 0
		return m
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	maxOffset := total - visible
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
