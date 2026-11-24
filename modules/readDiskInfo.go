package modules

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
	"golang.org/x/sys/windows"
)

type diskInformation struct {
	label            string
	capacityBytes    uint64
	activePercent    float64
	transfersPerSec  float64
	avgResponseMs    float64
	readBytesPerSec  float64
	writeBytesPerSec float64
}

func readDiskInfo(path string) (diskInformation, error) {
	label, err := getVolumeLabel(path)
	if err != nil {
		fmt.Println("Volume label: ", err)
		return diskInformation{}, err
	}
	usage, err := disk.Usage(path)
	if err != nil {
		fmt.Println("Disk usage:", err)
		return diskInformation{}, err
	}

	return diskInformation{
		label:         label,
		capacityBytes: usage.Total,
	}, nil
}

func getVolumeLabel(path string) (string, error) {
	root, err := windows.UTF16PtrFromString(path)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	var buf [windows.MAX_PATH + 1]uint16
	err = windows.GetVolumeInformation(
		root, &buf[0], uint32(len(buf)),
		nil, nil, nil, nil, 0,
	)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	name := windows.UTF16ToString(buf[:])
	if name == "" {
		name = path
	}
	return name, nil
}
