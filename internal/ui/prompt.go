package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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
