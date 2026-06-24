package ui

import (
	"fmt"
	"strconv"
	"strings"
)

func NumberedSelect(title string, items []SelectItem) ([]string, error) {
	Section(title)
	for i, item := range items {
		fmt.Printf("%2d. %-16s %s\n", i+1, item.Key, item.Label)
	}
	text := PromptText("Select numbers, ranges, or all", "")
	if strings.EqualFold(strings.TrimSpace(text), "all") {
		var keys []string
		for _, item := range items {
			keys = append(keys, item.Key)
		}
		return keys, nil
	}
	var keys []string
	for _, token := range strings.Fields(strings.ReplaceAll(text, ",", " ")) {
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			lo, _ := strconv.Atoi(parts[0])
			hi, _ := strconv.Atoi(parts[1])
			for n := lo; n <= hi; n++ {
				if n >= 1 && n <= len(items) {
					keys = append(keys, items[n-1].Key)
				}
			}
			continue
		}
		n, err := strconv.Atoi(token)
		if err == nil && n >= 1 && n <= len(items) {
			keys = append(keys, items[n-1].Key)
		}
	}
	return keys, nil
}
