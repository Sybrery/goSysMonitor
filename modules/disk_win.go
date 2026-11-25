package modules

import (
	"fmt"
	"math"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

func PrintDisk() {
	partitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Println("Disk partitions:", err)
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
			fmt.Printf("Disk %s: %v\n", path, err)
			continue
		}
		disks = append(disks, info)
	}
	if len(disks) == 0 {
		fmt.Println("No readable volumes found")
		return
	}

	if err := readDiskPerformance(disks); err != nil {
		fmt.Println("Disk performance:", err)
	}

	for i, info := range disks {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("Disk %s", info.path)
		if info.model != "" {
			fmt.Printf(" (%s)", info.model)
		}
		if info.label != "" {
			fmt.Printf(" | Volume: %s", info.label)
		}
		fmt.Println()

		free := float64(min(info.freeBytes, info.capacityBytes)) / (1 << 30)
		capacity := float64(info.capacityBytes) / (1 << 30)
		fmt.Printf("Free: %.1f / %.1f GiB | Used: %.1f GiB\n", free, capacity, capacity-free)

		if info.idlePercent.available {
			activePercent := math.Max(0, math.Min(100, 100-info.idlePercent.value))
			bar, _ := UsageBar(activePercent)
			fmt.Printf("Active time: [%s] %.1f%%\n", bar, activePercent)
		} else {
			fmt.Println("Active time: unavailable")
		}

		transferRate := "unavailable"
		if info.readBytesPerSec.available && info.writeBytesPerSec.available {
			transferRate = fmt.Sprintf("%.2f MiB/s",
				(info.readBytesPerSec.value+info.writeBytesPerSec.value)/(1<<20))
		}
		fmt.Printf("Disk transfer rate: %s", transferRate)
		if info.transfersPerSec.available {
			fmt.Printf(" | %.2f ops/s", info.transfersPerSec.value)
		}
		fmt.Println()

		responseTime := "unavailable"
		if info.responseSeconds.available {
			responseTime = fmt.Sprintf("%.2f ms", info.responseSeconds.value*1000)
		}
		fmt.Printf("Average response time: %s\n", responseTime)

		readSpeed := "unavailable"
		if info.readBytesPerSec.available {
			readSpeed = fmt.Sprintf("%.2f MiB/s", info.readBytesPerSec.value/(1<<20))
		}
		writeSpeed := "unavailable"
		if info.writeBytesPerSec.available {
			writeSpeed = fmt.Sprintf("%.2f MiB/s", info.writeBytesPerSec.value/(1<<20))
		}
		fmt.Printf("Read speed: %s | Write speed: %s\n", readSpeed, writeSpeed)
	}
}
