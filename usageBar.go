package main

import "strings"

func UsageBar(percent float64) string {
	filled := int(percent / 10)
	unfilled := 10 - filled
	bar := strings.Repeat("#", filled)
	emptyPart := strings.Repeat(".", unfilled)
	return bar + emptyPart
}
