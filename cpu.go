package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func PrintCpu() {
	c1, err := cpu.Percent(time.Second, false)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	c2, err := cpu.Percent(time.Second, true)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	lp, err := cpu.Counts(true)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	cp, err := cpu.Counts(false)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ci, err := cpu.Info()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if len(ci) == 0 {
		return
	}
	if len(c1) == 0 {
		return
	}

	physicalCores := fmt.Sprintf("Physical cores: %d", cp)
	logicalProcessors := fmt.Sprintf("Logical processors: %d", lp)
	sockets := fmt.Sprintf("Sockets: %d", len(ci))
	maxMhz := fmt.Sprintf("Max frequency: %.0f MHz", ci[0].Mhz)
	totalUsage := fmt.Sprintf("Total usage: %.1f%%", c1[0])

	fmt.Println(ci[0].ModelName)
	fmt.Println(physicalCores)
	fmt.Println(logicalProcessors)
	fmt.Println(sockets)
	fmt.Println(maxMhz)
	fmt.Println(totalUsage)

	for i, usage := range c2 {
		filled := int(usage / 10)
		unfilled := 10 - filled
		bar := strings.Repeat("#", filled)
		emptyPart := strings.Repeat(".", unfilled)
		fmt.Printf("CPU %d: [%s%s] %.1f%%\n", i+1, bar, emptyPart, usage)
	}

}
