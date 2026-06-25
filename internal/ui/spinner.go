package ui

import (
	"fmt"
	"strings"
	"time"
)

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
