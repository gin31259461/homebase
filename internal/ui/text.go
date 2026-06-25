package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
