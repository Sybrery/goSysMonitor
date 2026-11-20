package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println("Golang System Monitor [Version 0.1]")
	fmt.Println("Author: Sybrery")

	reader := bufio.NewScanner(os.Stdin)

	for reader.Scan() {
		input := strings.TrimSpace(reader.Text())

		switch input {
		case "gsm help":
			fmt.Println("info")
		case "gsm -c":
			PrintCpu()
		case "gsm -m":
			PrintMem()
		default:
			fmt.Println("invalid command")
		}
	}
	if err := reader.Err(); err != nil {
		log.Fatal(err)
	}
}
