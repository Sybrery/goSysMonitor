package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strings"

	"golangSystemMonitor/internal/monitor"
	"golangSystemMonitor/internal/terminal"
)

const (
	version  = "1.0"
	helpHint = "Type \"gsm help\" to list commands."
)

type resourceCommand struct {
	flag   string
	name   string
	render func(io.Writer)
}

var resources = []resourceCommand{
	{"-c", "CPU", monitor.PrintCPU},
	{"-m", "RAM", monitor.PrintMemory},
	{"-g", "GPU", monitor.PrintGPU},
	{"-d", "disk", monitor.PrintDisk},
	{"-w", "Wi-Fi", monitor.PrintWiFi},
}

func printHelp() {
	fmt.Println("\nGolang System Monitor commands")
	for _, resource := range resources {
		fmt.Printf("  gsm %s          Show %s information\n", resource.flag, resource.name)
		fmt.Printf("  gsm %s proc     Show top %s processes\n", resource.flag, resource.name)
	}
	fmt.Print("  gsm -p          Show top RAM processes\n" +
		"  gsm help        Show commands\n" +
		"  gsm exit        Quit\n\n" +
		"Press Enter to stop a live view.\n\n")
}

func main() {
	fmt.Printf("Golang System Monitor [Version %s]\n", version)
	fmt.Println("Author: Sybrery")
	fmt.Println(helpHint)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	interactive, restoreConsole := terminal.EnableLiveConsole()
	defer restoreConsole()
	inputs := make(chan string)
	go readCommands(ctx, inputs)

	for {
		fmt.Print(">> ")
		var input string
		select {
		case <-ctx.Done():
			return
		case line, ok := <-inputs:
			if !ok {
				return
			}
			input = strings.Join(strings.Fields(line), " ")
		}
		switch input {
		case "gsm help":
			printHelp()
			continue
		case "gsm exit":
			return
		}
		command, isGsm := strings.CutPrefix(input, "gsm ")
		flag, processes := strings.CutSuffix(command, " proc")
		if input == "gsm -p" {
			flag, processes = "-m", true
		}
		index := slices.IndexFunc(resources, func(resource resourceCommand) bool {
			return resource.flag == flag
		})
		if !isGsm || index < 0 {
			fmt.Println("Error: unknown command.", helpHint)
			continue
		}
		if !processes {
			terminal.RunMonitor(ctx, inputs, interactive, input, resources[index].render)
			continue
		}
		render, closeView := monitor.OpenProcessView(flag)
		terminal.RunMonitor(ctx, inputs, interactive, input, render)
		closeView()
	}
}

func readCommands(ctx context.Context, inputs chan<- string) {
	defer close(inputs)
	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		select {
		case inputs <- reader.Text():
		case <-ctx.Done():
			return
		}
	}
	if err := reader.Err(); err != nil {
		monitor.PrintError(os.Stderr, "read input", err)
	}
}
