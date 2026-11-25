package terminal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

func EnableLiveConsole() (bool, func()) {
	var inputMode, outputMode uint32
	var screen windows.ConsoleScreenBufferInfo
	console := windows.Handle(os.Stdout.Fd())
	if windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &inputMode) != nil ||
		windows.GetConsoleMode(console, &outputMode) != nil ||
		windows.GetConsoleScreenBufferInfo(console, &screen) != nil ||
		windows.SetConsoleMode(console, outputMode|windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
		return false, func() {}
	}
	return true, func() {
		windows.SetConsoleMode(console, outputMode)
	}
}

func RunMonitor(ctx context.Context, inputs <-chan string, interactive bool, command string, render func(io.Writer)) {
	if !interactive {
		render(os.Stdout)
		return
	}

	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")
	writeLiveFrame(os.Stdout, command, "")
	last := refreshMonitor(ctx, inputs, os.Stdout, command, render)
	fmt.Printf("\x1b[H\x1b[J>> %s\n", command)
	if last != "" {
		fmt.Println(strings.TrimRight(last, "\r\n"))
	}
}

func refreshMonitor(ctx context.Context, inputs <-chan string, out *os.File, command string, render func(io.Writer)) string {
	var frame bytes.Buffer
	var last string
	for {
		started := time.Now()
		frame.Reset()
		render(&frame)
		select {
		case <-ctx.Done():
			return last
		case <-inputs:
			if last == "" {
				return frame.String()
			}
			return last
		default:
		}

		last = frame.String()
		writeLiveFrame(out, command, last)

		next := time.NewTimer(time.Until(started.Add(time.Second)))
		select {
		case <-ctx.Done():
			next.Stop()
			return last
		case <-inputs:
			next.Stop()
			return last
		case <-next.C:
		}
	}
}

func writeLiveFrame(out *os.File, command, frame string) {
	var info windows.ConsoleScreenBufferInfo
	if windows.GetConsoleScreenBufferInfo(windows.Handle(out.Fd()), &info) != nil {
		return
	}
	width := int(info.Window.Right) - int(info.Window.Left) + 1
	height := int(info.Window.Bottom) - int(info.Window.Top) + 1

	lines := []string{">> " + command}
	if frame != "" && height > 2 {
		body := strings.Split(strings.TrimRight(frame, "\r\n"), "\n")
		lines = append(lines, body[:min(len(body), height-2)]...)
	}
	if height > 1 {
		lines = append(lines, "Press Enter to stop.")
	}

	var screen strings.Builder
	screen.WriteString("\x1b[H\x1b[J")
	lineWidth := width - 1
	for i, line := range lines {
		fmt.Fprintf(&screen, "\x1b[%d;1H%s", i+1, fitMonitorLine(strings.TrimSuffix(line, "\r"), lineWidth))
	}
	screen.WriteString("\x1b[H")
	fmt.Fprint(out, screen.String())
}

func fitMonitorLine(line string, columns int) string {
	for i, ch := range line {
		width := 1
		if ch > 127 {
			width = 2
		}
		if columns < width {
			return line[:i]
		}
		columns -= width
	}
	return line
}
