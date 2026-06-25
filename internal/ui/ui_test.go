package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSelectorHeightForWindow(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{0, DefaultSelectorHeight},
		{10, 4},
		{40, DefaultSelectorHeight},
	}
	for _, tt := range tests {
		if got := SelectorHeightForWindow(tt.height); got != tt.want {
			t.Fatalf("SelectorHeightForWindow(%d) = %d; want %d", tt.height, got, tt.want)
		}
	}
}

func TestSelectorKeepsCursorVisible(t *testing.T) {
	m := NewSelector("test", []SelectItem{
		{Key: "0"}, {Key: "1"}, {Key: "2"}, {Key: "3"}, {Key: "4"},
	})
	m.height = 2
	m.cursor = 4
	m = m.keepCursorVisible()
	if m.offset != 3 {
		t.Fatalf("offset = %d; want 3", m.offset)
	}
}

func TestSelectorDefaultsSelected(t *testing.T) {
	m := NewSelector("test", []SelectItem{
		{Key: "core", DefaultSelected: true},
		{Key: "dev"},
	})
	got := m.SelectedKeys()
	if len(got) != 1 || got[0] != "core" {
		t.Fatalf("selected = %#v; want [core]", got)
	}
}

func TestNumberedSelectionKeys(t *testing.T) {
	items := []SelectItem{
		{Key: "core", DefaultSelected: true},
		{Key: "dev"},
		{Key: "apps"},
	}
	got := numberedSelectionKeys("", items)
	if len(got) != 1 || got[0] != "core" {
		t.Fatalf("defaults = %#v; want [core]", got)
	}
	got = numberedSelectionKeys("1, 2-3", items)
	if strings.Join(got, ",") != "core,dev,apps" {
		t.Fatalf("range selection = %#v; want all keys", got)
	}
	got = numberedSelectionKeys("x-3 -1 3-2 2", items)
	if len(got) != 1 || got[0] != "dev" {
		t.Fatalf("malformed ranges selected unexpected keys: %#v", got)
	}
}

func TestSelectorVimJumps(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{Key: "0"}, {Key: "1"}, {Key: "2"},
	})
	var teaModel tea.Model = model
	teaModel, _ = teaModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	model = teaModel.(SelectorModel)
	if model.cursor != 2 {
		t.Fatalf("cursor after G = %d; want 2", model.cursor)
	}
	teaModel = model
	teaModel, _ = teaModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	teaModel, _ = teaModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = teaModel.(SelectorModel)
	if model.cursor != 0 {
		t.Fatalf("cursor after gg = %d; want 0", model.cursor)
	}
}

func TestSelectorStateTextHighlight(t *testing.T) {
	tests := []struct {
		state SelectState
		text  string
	}{
		{SelectStateGood, "good"},
		{SelectStatePartial, "partial"},
		{SelectStateBad, "bad"},
		{SelectStateUnknown, "unknown"},
	}
	for _, tt := range tests {
		if got := renderStateText(tt.text, tt.state); !strings.Contains(got, tt.text) {
			t.Fatalf("state text = %q; want to contain %q", got, tt.text)
		}
	}
}

func TestSelectorHighlightsDetailValueNotTitle(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{
			Key:         "orphans",
			Label:       "Orphaned packages",
			DetailValue: "2 orphaned package(s), 2.5 MiB",
			Detail:      "pacman -Rns",
			State:       SelectStateBad,
		},
	})
	view := model.View()
	if !strings.Contains(view, "orphans") || !strings.Contains(view, "Orphaned packages") {
		t.Fatalf("view lost neutral title text:\n%s", view)
	}
	if !strings.Contains(view, "2 orphaned package(s), 2.5 MiB") {
		t.Fatalf("view lost calculated detail value:\n%s", view)
	}
	if !strings.Contains(view, "pacman -Rns") {
		t.Fatalf("view lost static detail:\n%s", view)
	}
}

func TestSelectorScrollbarDynamicLength(t *testing.T) {
	config := ScrollbarConfig{MinThumbRatio: 0.2, MaxThumbRatio: 1}

	top := scrollbarGeometryFor(20, 10, 0, config)
	if !top.visible || top.start != 0 || top.length != 2 {
		t.Fatalf("top geometry = %#v; want visible start 0 length 2", top)
	}

	middle := scrollbarGeometryFor(20, 10, 5, config)
	if !middle.visible || middle.start != 3 || middle.length != 5 {
		t.Fatalf("middle geometry = %#v; want visible start 3 length 5", middle)
	}

	aboveMiddle := scrollbarGeometryFor(20, 10, 4, config)
	if !aboveMiddle.visible || aboveMiddle.length != 5 {
		t.Fatalf("above-middle geometry = %#v; want max length 5", aboveMiddle)
	}

	belowMiddle := scrollbarGeometryFor(20, 10, 6, config)
	if !belowMiddle.visible || belowMiddle.length != 5 {
		t.Fatalf("below-middle geometry = %#v; want max length 5", belowMiddle)
	}

	bottom := scrollbarGeometryFor(20, 10, 10, config)
	if !bottom.visible || bottom.start != 8 || bottom.length != 2 {
		t.Fatalf("bottom geometry = %#v; want visible start 8 length 2", bottom)
	}
}

func TestSelectorScrollbarHiddenWhenContentFitsByDefault(t *testing.T) {
	hidden := scrollbarGeometryFor(3, 10, 0, DefaultScrollbarConfig())
	if hidden.visible {
		t.Fatalf("default fitted-content scrollbar = %#v; want hidden", hidden)
	}

	shown := scrollbarGeometryFor(3, 10, 0, ScrollbarConfig{
		ShowWhenContentFits: true,
		MinThumbRatio:       0.25,
		MaxThumbRatio:       1,
	})
	if !shown.visible || shown.start != 0 || shown.length != 10 {
		t.Fatalf("configured fitted-content scrollbar = %#v; want full-height thumb", shown)
	}
}

func TestSelectorScrollbarUsesTrackAndDynamicThumb(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{Key: "0"}, {Key: "1"}, {Key: "2"}, {Key: "3"}, {Key: "4"},
		{Key: "5"}, {Key: "6"}, {Key: "7"}, {Key: "8"}, {Key: "9"},
	})
	model.height = 4
	model.offset = 3
	thumbs := 0
	tracks := 0
	for row := 0; row < model.visibleCount(); row++ {
		scroll := model.scrollbar(row, model.visibleCount())
		if strings.Contains(scroll, "|") {
			thumbs++
		}
		if strings.Contains(scroll, ":") {
			tracks++
		}
	}
	if thumbs != 2 {
		t.Fatalf("scrollbar thumbs = %d; want 2", thumbs)
	}
	if tracks != model.visibleCount()-2 {
		t.Fatalf("scrollbar tracks = %d; want %d", tracks, model.visibleCount()-2)
	}
}

func TestSelectorScrollbarTrackContinuesThroughDetailLines(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{Key: "0", Detail: "detail text without punctuation"},
		{Key: "1"},
		{Key: "2"},
		{Key: "3"},
	})
	model.height = 2
	view := model.View()
	for _, line := range strings.Split(view, "\n") {
		if !strings.Contains(line, "detail text") {
			continue
		}
		if strings.Contains(line, ":") {
			return
		}
		t.Fatalf("detail line has no scrollbar track:\n%s", view)
	}
	t.Fatalf("view lost detail line:\n%s", view)
}

func TestSelectorScrollbarThumbContinuesThroughDetailLines(t *testing.T) {
	model := NewSelectorWithScrollbar("test", []SelectItem{
		{Key: "0", Detail: "detail text without punctuation"},
		{Key: "1"},
		{Key: "2"},
		{Key: "3"},
	}, ScrollbarConfig{
		MinThumbRatio: 1,
		MaxThumbRatio: 1,
	})
	model.height = 2
	view := model.View()
	for _, line := range strings.Split(view, "\n") {
		if !strings.Contains(line, "detail text") {
			continue
		}
		if strings.Contains(line, "|") {
			return
		}
		t.Fatalf("detail line split the scrollbar thumb:\n%s", view)
	}
	t.Fatalf("view lost detail line:\n%s", view)
}

func TestSelectorInspectCtrlScroll(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{Key: "core", Inspect: strings.Join([]string{
			"line 1", "line 2", "line 3", "line 4", "line 5", "line 6",
			"line 7", "line 8", "line 9", "line 10", "line 11", "line 12",
		}, "\n")},
	})
	var teaModel tea.Model = model
	teaModel, _ = teaModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	teaModel, _ = teaModel.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	model = teaModel.(SelectorModel)
	if model.inspectOffset != InspectHeight/2 {
		t.Fatalf("inspect offset after ctrl+d = %d; want %d", model.inspectOffset, InspectHeight/2)
	}
	teaModel = model
	teaModel, _ = teaModel.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = teaModel.(SelectorModel)
	if model.inspectOffset != 0 {
		t.Fatalf("inspect offset after ctrl+u = %d; want 0", model.inspectOffset)
	}
}

func TestSelectorInspectUsesScrollbar(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{Key: "core", Inspect: strings.Join([]string{
			"line 1", "line 2", "line 3", "line 4", "line 5", "line 6",
			"line 7", "line 8", "line 9", "line 10", "line 11", "line 12",
			"line 13", "line 14", "line 15", "line 16",
		}, "\n")},
	})
	model.inspect = true
	model.inspectOffset = 4
	view := model.View()
	if strings.Count(view, "|") == 0 || strings.Count(view, ":") == 0 {
		t.Fatalf("inspect view lost scrollbar:\n%s", view)
	}
}

func TestSelectorWrapsDescriptionToWindowWidth(t *testing.T) {
	model := NewSelector("test", []SelectItem{
		{
			Key:    "very-long-key",
			Label:  "Very long label",
			Detail: "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu",
			Inspect: strings.Join([]string{
				"alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu",
				"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			}, "\n"),
		},
	})
	model.width = 42
	model.inspect = true
	view := model.View()
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > model.width {
			t.Fatalf("line width = %d, want <= %d: %q", got, model.width, line)
		}
	}
	if !strings.Contains(view, "lambda mu") {
		t.Fatalf("wrapped view lost detail suffix:\n%s", view)
	}
}
