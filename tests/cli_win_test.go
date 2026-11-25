package tests

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCLIWithRedirectedInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = ".."
	cmd.Stdin = strings.NewReader("gsm help\n  gsm   -m  \ngsm -p\ngsm -unknown\ngsm exit\n")
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run monitor: %v\n%s", err, result)
	}
	output := string(result)

	for _, text := range []string{
		"Golang System Monitor [Version 1.0]",
		"gsm -c proc",
		"gsm -m proc",
		"gsm -g proc",
		"gsm -d proc",
		"gsm -w proc",
		"Top RAM processes",
		"Error: unknown command.",
	} {
		if !strings.Contains(output, text) {
			t.Errorf("missing %q in output:\n%s", text, output)
		}
	}
	if count := strings.Count(output, "Memory type:"); count != 1 {
		t.Errorf("got %d RAM snapshots, want 1:\n%s", count, output)
	}
	if strings.Contains(output, "\x1b[") {
		t.Errorf("redirected output contains terminal control sequences:\n%s", output)
	}
}
