package monitor

import (
	"fmt"
	"math"
	"strings"
)

func UsageBar(percent float64) (string, error) {
	if math.IsNaN(percent) || percent < 0 || percent > 100 {
		return "", fmt.Errorf("usage percent out of range: %g", percent)
	}

	const width = 10
	filled := int(percent / (100 / width))
	return strings.Repeat("#", filled) + strings.Repeat(".", width-filled), nil
}
