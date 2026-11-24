package modules

import (
	"fmt"
)

func PrintDisk() {
	info, err := readDiskInfo("C:\\")
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Printf("Disk %s\n", info.label)
	fmt.Printf("Capacity: %.1f GiB\n", float64(info.capacityBytes)/(1<<30))
}
