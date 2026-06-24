package ui

import "testing"

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
