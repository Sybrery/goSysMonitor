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

	totalMem := fmt.Sprintf("Total: %.1f GiB", float64(v.Total)/(1<<30))
	availableMem := fmt.Sprintf("Available: %.1f GiB", float64(v.Available)/(1<<30))
	swapMem := fmt.Sprintf("Swap used: %.1f GiB", float64(s.Used)/(1<<30))
	swapTotalMem := fmt.Sprintf("Swap capacity: %.1f GiB", float64(s.Total)/(1<<30))

	usedMem := float64(v.Used) / (1 << 30)

	fmt.Println(availableMem, "/", totalMem)
	fmt.Printf("In use: %.1f GiB [%s] %.1f%%\n", usedMem, UsageBar(v.UsedPercent), v.UsedPercent)
	fmt.Println(swapMem, "/", swapTotalMem)
}
