package modules

import (
	"fmt"

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

//func readDiskInfo(path string) (diskInformation, error) {

//}

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
