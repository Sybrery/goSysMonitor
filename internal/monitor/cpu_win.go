package monitor

import (
	"fmt"
	"io"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"golang.org/x/sys/windows"
)

func PrintCPU(out io.Writer) {
	usages, performancePercent, err := readCPUPerformance()
	if err != nil {
		PrintError(out, "read CPU usage", err)
		return
	}
	if len(usages) == 0 {
		fmt.Fprintln(out, "CPU usage: unavailable")
		return
	}
	var sum float64
	for _, usage := range usages {
		sum += usage
	}

	logicalProcessors, err := cpu.Counts(true)
	if err != nil {
		PrintError(out, "count logical processors", err)
		return
	}
	physicalCores, err := cpu.Counts(false)
	if err != nil {
		PrintError(out, "count physical cores", err)
		return
	}

	processors, err := cpu.Info()
	if err != nil {
		PrintError(out, "read CPU information", err)
		return
	}
	if len(processors) == 0 {
		return
	}

	averageUsage := sum / float64(len(usages))

	fmt.Fprintln(out, processors[0].ModelName)

	for i, usage := range usages {
		bar, err := UsageBar(usage)
		if err != nil {
			PrintError(out, "format CPU usage", err)
			return
		}
		fmt.Fprintf(out, "CPU %2d: [%s] %5.1f%%", i+1, bar, usage)
		if i%2 == 1 || i == len(usages)-1 {
			fmt.Fprintln(out)
		} else {
			fmt.Fprint(out, "    ")
		}
	}
	baseGHz := processors[0].Mhz / 1000
	speed := unavailable
	if baseGHz > 0 && performancePercent > 0 {
		speed = fmt.Sprintf("%.2f GHz", baseGHz*performancePercent/100)
	}
	fmt.Fprintf(out, "Utilization: %.1f%% | Speed: %s\n", averageUsage, speed)

	info, err := readMemoryPerformance()
	if err != nil {
		PrintError(out, "read process, thread, and handle counts", err)
		return
	}
	fmt.Fprintf(out, "Processes: %d, Threads: %d, Handles: %d\n", info.processCount, info.threadCount, info.handleCount)

	fmt.Fprintf(out, "Base speed: %.2f GHz\n", baseGHz)
	fmt.Fprintf(out, "Sockets: %d\n", len(processors))
	fmt.Fprintf(out, "Cores: %d\n", physicalCores)
	fmt.Fprintf(out, "Logical processors: %d\n", logicalProcessors)
}

func readCPUPerformance() ([]float64, float64, error) {
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
