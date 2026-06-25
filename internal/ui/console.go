package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
