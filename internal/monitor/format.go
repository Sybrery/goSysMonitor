package monitor

import (
	"fmt"
	"io"
)

const (
	mibibyte    = 1 << 20
	gibibyte    = 1 << 30
	unavailable = "unavailable"
)

func PrintError(out io.Writer, operation string, err error) {
	fmt.Fprintf(out, "Error: %s: %v\n", operation, err)
}
