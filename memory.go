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
	sm, err := mem.NewExWindows().VirtualMemory()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	bar, err := UsageBar(v.UsedPercent)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	totalMem := fmt.Sprintf("%.1f GiB", float64(v.Total)/(1<<30))
	availableMem := fmt.Sprintf("Available: %.1f GiB", float64(v.Available)/(1<<30))
	swapMem := fmt.Sprintf("Swap used: %.1f GiB", float64(s.Used)/(1<<30))
	swapTotalMem := fmt.Sprintf("%.1f GiB", float64(s.Total)/(1<<30))
	commitTotal := fmt.Sprintf("Committed: %.1f GiB", float64(sm.CommitTotal)/(1<<30))
	commitLimit := fmt.Sprintf("%.1f GiB", float64(sm.CommitLimit)/(1<<30))
	usedMem := float64(v.Used) / (1 << 30)

	fmt.Println(availableMem, "/", totalMem)
	fmt.Printf("In use: %.1f GiB [%s] %.1f%%\n", usedMem, bar, v.UsedPercent)
	fmt.Println(swapMem, "/", swapTotalMem)
	fmt.Println(commitTotal, "/", commitLimit)
}
