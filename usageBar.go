package main

import (
	"fmt"
	"math"
	"strings"
)

func UsageBar(percent float64) (string, error) {
	if math.IsNaN(percent) || percent < 0 || percent > 100 {
		return "", fmt.Errorf("usage percent out of range: %g", percent)
	}

	filled := int(percent / 10)
	unfilled := 10 - filled
	bar := strings.Repeat("#", filled)
	emptyPart := strings.Repeat(".", unfilled)

	return bar + emptyPart, nil
}
