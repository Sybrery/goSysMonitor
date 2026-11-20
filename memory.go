package main

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/mem"
)

func PrintMem() {
	v, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	s, err := mem.SwapMemory()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	totalMem := fmt.Sprintf("Total: %.1f GB", float64(v.Total)/(1<<30))
	usedMem := fmt.Sprintf("In use: %.1f GB", float64(v.Used)/(1<<30))
	usedPercentMem := fmt.Sprintf("(%.1f%%)", float64(v.UsedPercent))
	availableMem := fmt.Sprintf("Available: %.1f GB", float64(v.Available)/(1<<30))
	committedMem := fmt.Sprintf("Committed: %.1f GB", float64(s.Used)/(1<<30))
	committedTotalMem := fmt.Sprintf("Committed Total: %.1f GB", float64(s.Total)/(1<<30))

	fmt.Println(totalMem, usedMem, usedPercentMem, availableMem, committedMem, committedTotalMem)
}
