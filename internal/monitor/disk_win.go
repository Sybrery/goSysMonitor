package monitor

import (
	"fmt"
	"io"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

func PrintDisk(out io.Writer) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		PrintError(out, "list disk volumes", err)
	}

	var disks []diskInformation
	for _, partition := range partitions {
		path := partition.Mountpoint
		if path == "" {
			continue
		}
		if !strings.HasSuffix(path, "\\") {
			path += "\\"
		}
		info, err := readDiskInfo(path)
		if err != nil {
			PrintError(out, "read disk "+path, err)
			continue
		}
		disks = append(disks, info)
	}
	if len(disks) == 0 {
		fmt.Fprintln(out, "No readable volumes found")
		return
	}

	if err := readDiskPerformance(disks); err != nil {
		PrintError(out, "read disk performance", err)
	}

	for i, info := range disks {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "Disk %s", info.path)
		if info.model != "" {
			fmt.Fprintf(out, " (%s)", info.model)
		}
		if info.label != "" {
			fmt.Fprintf(out, " | Volume: %s", info.label)
		}
		fmt.Fprintln(out)

		free := float64(min(info.freeBytes, info.capacityBytes)) / gibibyte
		capacity := float64(info.capacityBytes) / gibibyte
		fmt.Fprintf(out, "Free: %.1f / %.1f GiB | Used: %.1f GiB\n", free, capacity, capacity-free)

		if info.idlePercent.available {
			activePercent := max(0, min(100, 100-info.idlePercent.value))
			bar, _ := UsageBar(activePercent)
			fmt.Fprintf(out, "Active time: [%s] %.1f%%\n", bar, activePercent)
		} else {
			fmt.Fprintln(out, "Active time: unavailable")
		}

		transferRate := unavailable
		if info.readBytesPerSec.available && info.writeBytesPerSec.available {
			rate := (info.readBytesPerSec.value + info.writeBytesPerSec.value) / mibibyte
			transferRate = fmt.Sprintf("%.2f MiB/s", rate)
		}
		fmt.Fprintf(out, "Disk transfer rate: %s", transferRate)
		if info.transfersPerSec.available {
			fmt.Fprintf(out, " | %.2f ops/s", info.transfersPerSec.value)
		}
		fmt.Fprintln(out)

		responseTime := formatDiskMetric(info.responseSeconds, 1000, "ms")
		fmt.Fprintf(out, "Average response time: %s\n", responseTime)

		readSpeed := formatDiskMetric(info.readBytesPerSec, 1.0/mibibyte, "MiB/s")
		writeSpeed := formatDiskMetric(info.writeBytesPerSec, 1.0/mibibyte, "MiB/s")
		fmt.Fprintf(out, "Read speed: %s | Write speed: %s\n", readSpeed, writeSpeed)
	}
}

func formatDiskMetric(metric diskMetric, scale float64, unit string) string {
	if !metric.available {
		return unavailable
	}
	return fmt.Sprintf("%.2f %s", metric.value*scale, unit)
}
