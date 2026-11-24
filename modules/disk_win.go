package modules

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
)

func PrintDisk() {
	label, err := getVolumeLabel(`C:\`)
	if err != nil {
		fmt.Println("Volume label: ", err)
	}
	usage, err := disk.Usage(`C:\`)
	if err != nil {
		fmt.Println("Disk usage:", err)
		return
	}
	capacityBytes := usage.Total

	fmt.Printf("Disk %s\n", label)
	fmt.Printf("Capacity: %.1f GiB\n", float64(capacityBytes)/(1<<30))
}
