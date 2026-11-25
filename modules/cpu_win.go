package modules

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"golang.org/x/sys/windows"
)

func PrintCpu() {
	usages, performancePercent, err := readCpuPerformance()
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
	baseGHz := ci[0].Mhz / 1000
	speed := "unavailable"
	if baseGHz > 0 && performancePercent > 0 {
		currentMHz := baseGHz * performancePercent / 100
		speed = fmt.Sprintf("%.2f GHz", currentMHz/1000)
	}
	fmt.Printf("Utilization: %.1f%% | Speed: %s\n", averageUsage, speed)

	info, err := ReadMemoryPerformance()
	if err != nil {
		fmt.Println("GetPerformanceInfo:", err)
		return
	}
	fmt.Printf("Processes: %d, Threads: %d, Handles: %d\n", info.processCount, info.threadCount, info.handleCount)

	fmt.Printf("Base speed: %.2f GHz\n", baseGHz)
	fmt.Printf("Sockets: %d\n", len(ci))
	fmt.Printf("Cores: %d\n", physicalCores)
	fmt.Printf("Logical processors: %d\n", logicalProcessors)

}

func readCpuPerformance() ([]float64, float64, error) {
	query, err := openPerformanceQuery()
	var counter windows.Handle
	if err == nil {
		defer query.close()
		counter, err = query.add(`\Processor Information(_Total)\% Processor Performance`)
		if err == nil && query.collect() != nil {
			counter = 0
		}
	}

	usages, err := cpu.Percent(time.Second, true)
	if err != nil {
		return nil, 0, err
	}
	if counter != 0 && query.collect() == nil {
		performancePercent, _ := query.value(counter)
		return usages, performancePercent, nil
	}
	return usages, 0, nil
}
