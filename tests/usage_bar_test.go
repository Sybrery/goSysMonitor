package tests

import (
	"math"
	"testing"

	"golangSystemMonitor/internal/monitor"
)

func TestUsageBar(t *testing.T) {
	testTable := []struct {
		percent  float64
		expected string
		wantErr  bool
	}{
		{
			percent:  0,
			expected: "..........",
			wantErr:  false,
		},
		{
			percent:  59.9,
			expected: "#####.....",
			wantErr:  false,
		},
		{
			percent:  100,
			expected: "##########",
			wantErr:  false,
		},
		{
			percent:  100.1,
			expected: "",
			wantErr:  true,
		},
		{
			percent:  -1,
			expected: "",
			wantErr:  true,
		},
		{
			percent:  math.NaN(),
			expected: "",
			wantErr:  true,
		},
	}

	for _, testCase := range testTable {
		result, err := monitor.UsageBar(testCase.percent)
		gotErr := err != nil
		if gotErr != testCase.wantErr {
			t.Errorf("UsageBar(%g) error = %v, wantErr = %t", testCase.percent, err, testCase.wantErr)
		}

		if result != testCase.expected {
			t.Errorf("UsageBar(%g) = %q, want %q", testCase.percent, result, testCase.expected)
		}
	}
}
