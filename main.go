package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func PrintHelp() {
	fmt.Println()
	fmt.Println("Golang System Monitor is a tool for monitoring your PC's Processes, CPU, GPU, RAM and disk usage")
	fmt.Println()
	fmt.Println("The commands are:")
	fmt.Println("                 gsm -c          show a cpu info")
	fmt.Println("                 gsm -g          show a gpu info")
	fmt.Println("                 gsm -m          show a ram info")
	fmt.Println("                 gsm -d          show a disk info")
	fmt.Println("                 gsm -p          show a processes info")
	fmt.Println("                 gsm exit         quit")
	fmt.Println()
}

func main() {
	fmt.Println("Golang System Monitor [Version 0.1]")
	fmt.Println("Author: Sybrery")
	fmt.Println("Type \"gsm help\" to list commands.")
	fmt.Printf(">> ")

	reader := bufio.NewScanner(os.Stdin)

	for reader.Scan() {
		input := strings.TrimSpace(reader.Text())

		switch input {
		case "gsm help":
			PrintHelp()
			fmt.Printf(">> ")
		case "gsm -c":
			PrintCpu()
			fmt.Printf(">> ")
		case "gsm -m":
			PrintMem()
			fmt.Printf(">> ")
		case "gsm -g":
			PrintGpu()
			fmt.Printf(">> ")
		case "gsm -d":
			PrintDisk()
			fmt.Printf(">> ")
		case "gsm -p":
			PrintProcesses()
			fmt.Printf(">> ")
		case "gsm exit":
			return
		default:
			fmt.Println("Unknown command. Type \"gsm help\" to list commands.")
			fmt.Print(">> ")
		}
	}
	if err := reader.Err(); err != nil {
		log.Fatal(err)
	}
}
