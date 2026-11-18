package main

import (
	"fmt"
	"log"

	"github.com/shirou/gopsutil/v4/mem"
)

var variant int

func main() {
	for {
		fmt.Println("------------------------")
		fmt.Println("-----System Monitor-----")
		fmt.Println("------------------------")
		fmt.Println("1. Processes")
		fmt.Println("2. Perfomance")
		fmt.Println("3. Exit")

		fmt.Printf("Your choice: ")
		fmt.Scan(&variant)

		if variant == 1 {
			fmt.Println("all processes")
		} else if variant == 2 {
			v, err := mem.VirtualMemory()
			if err != nil {
				log.Fatal(err)
			}

			s, err := mem.SwapMemory()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Total: %v, Used:%v, UsedPercent:%f%%, Available:%v, Committed Used:%v, Committed Total:%v\n", v.Total, v.Used, v.UsedPercent, v.Available, s.Used, s.Total)
		} else if variant == 3 {
			break
		} else {
			fmt.Println("Your input data is invalid, try again")
		}
	}
}
