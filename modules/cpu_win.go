package modules

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func PrintCpu() {
	usages, err := cpu.Percent(time.Second, true)
	if err != nil {
		fmt.Println("CPU usage:", err)
		return
	}
	if len(usages) == 0 {
		fmt.Println("CPU usage unavailable")
		return
	}
	var sum float64
	for _, usage := range usages {
		sum += usage
	}

	logicalProcessors, err := cpu.Counts(true)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	physicalCores, err := cpu.Counts(false)
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

	averageUsage := sum / float64(len(usages))

	fmt.Println(ci[0].ModelName)

	for i, usage := range usages {
		bar, err := UsageBar(usage)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("CPU %2d: [%s] %5.1f%%", i+1, bar, usage)
		if i%2 == 1 || i == len(usages)-1 {
			fmt.Println()
		} else {
			fmt.Print("    ")
		}
	}
	fmt.Println()
	fmt.Printf("Utilization: %.1f%% | Speed:\n", averageUsage)

	info, err := ReadPerformanceInfo()
	if err != nil {
		fmt.Println("GetPerformanceInfo:", err)
		return
	}
	fmt.Printf("Processes: %d, Threads: %d, Handles: %d\n", info.processCount, info.threadCount, info.handleCount)

	fmt.Printf("Max frequency: %.0f MHz\n", ci[0].Mhz)
	fmt.Printf("Sockets: %d\n", len(ci))
	fmt.Printf("Cores: %d\n", physicalCores)
	fmt.Printf("Logical processors: %d\n", logicalProcessors)

}
