package modules

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"golang.org/x/sys/windows"
)

type diskInformation struct {
	path             string
	label            string
	model            string
	capacityBytes    uint64
	freeBytes        uint64
	idlePercent      diskMetric
	transfersPerSec  diskMetric
	responseSeconds  diskMetric
	readBytesPerSec  diskMetric
	writeBytesPerSec diskMetric
}

type diskMetric struct {
	value     float64
	available bool
}

func readDiskInfo(path string) (diskInformation, error) {
	label, err := getVolumeLabel(path)
	if err != nil {
		return diskInformation{}, fmt.Errorf("volume label: %w", err)
	}
	usage, err := disk.Usage(path)
	if err != nil {
		return diskInformation{}, fmt.Errorf("disk usage: %w", err)
	}

	return diskInformation{
		path:          path,
		label:         label,
		model:         getVolumeDiskModels(path),
		capacityBytes: usage.Total,
		freeBytes:     usage.Free,
	}, nil
}

func getVolumeLabel(path string) (string, error) {
	root, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	var buf [windows.MAX_PATH + 1]uint16
	err = windows.GetVolumeInformation(
		root, &buf[0], uint32(len(buf)),
		nil, nil, nil, nil, 0,
	)
	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(buf[:]), nil
}

func getVolumeDiskModels(path string) string {
	root, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var name [windows.MAX_PATH + 1]uint16
	if err := windows.GetVolumeNameForVolumeMountPoint(root, &name[0], uint32(len(name))); err != nil {
		return ""
	}
	volumePath := strings.TrimSuffix(windows.UTF16ToString(name[:]), "\\")
	volume, err := openDiskDevice(volumePath)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(volume)

	const (
		getVolumeDiskExtents = 0x560000
		extentsHeaderSize    = 8
		extentSize           = 24
	)
	// DISK_EXTENT starts with the disk number, followed by two aligned int64s.
	extents := make([]byte, extentsHeaderSize+extentSize)
	var returned uint32
	for {
		err = windows.DeviceIoControl(volume, getVolumeDiskExtents, nil, 0,
			&extents[0], uint32(len(extents)), &returned, nil)
		if !errors.Is(err, windows.ERROR_MORE_DATA) {
			break
		}
		count := binary.LittleEndian.Uint32(extents[:4])
		required := extentsHeaderSize + uint64(count)*extentSize
		if required <= uint64(len(extents)) || required > 1<<31-1 {
			return ""
		}
		extents = make([]byte, required)
	}
	if err != nil || returned < extentsHeaderSize || returned > uint32(len(extents)) {
		return ""
	}

	count := binary.LittleEndian.Uint32(extents[:4])
	if count > (returned-extentsHeaderSize)/extentSize {
		return ""
	}
	var models []string
	seen := make(map[uint32]bool)
	for i := uint32(0); i < count; i++ {
		offset := extentsHeaderSize + i*extentSize
		number := binary.LittleEndian.Uint32(extents[offset : offset+4])
		if seen[number] {
			continue
		}
		seen[number] = true
		if model := getPhysicalDiskModel(number); model != "" {
			models = append(models, model)
		}
	}
	return strings.Join(models, ", ")
}

func getPhysicalDiskModel(number uint32) string {
	device, err := openDiskDevice(fmt.Sprintf("\\\\.\\PhysicalDrive%d", number))
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(device)

	const queryStorageProperty = 0x2d1400
	var query [12]byte // StorageDeviceProperty and PropertyStandardQuery are both zero.
	var header [8]byte
	var returned uint32
	if err := windows.DeviceIoControl(device, queryStorageProperty, &query[0], uint32(len(query)),
		&header[0], uint32(len(header)), &returned, nil); err != nil || returned < uint32(len(header)) {
		return ""
	}

	const productIDFieldEnd = 20
	size := binary.LittleEndian.Uint32(header[4:8])
	if size < productIDFieldEnd || size > 1<<31-1 {
		return ""
	}
	descriptor := make([]byte, size)
	if err := windows.DeviceIoControl(device, queryStorageProperty, &query[0], uint32(len(query)),
		&descriptor[0], size, &returned, nil); err != nil || returned < productIDFieldEnd || returned > size {
		return ""
	}
	offset := binary.LittleEndian.Uint32(descriptor[16:productIDFieldEnd])
	if offset < productIDFieldEnd || offset >= returned {
		return ""
	}
	model, _, _ := strings.Cut(string(descriptor[offset:returned]), "\x00")
	return strings.TrimSpace(model)
}

func openDiskDevice(path string) (windows.Handle, error) {
	devicePath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return windows.CreateFile(devicePath, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING, 0, 0)
}

func readDiskPerformance(disks []diskInformation) error {
	query, err := openPerformanceQuery()
	if err != nil {
		return err
	}
	defer query.close()

	type diskCounter struct {
		handle windows.Handle
		metric *diskMetric
	}
	var counters []diskCounter
	for i := range disks {
		info := &disks[i]
		drive := strings.TrimSuffix(info.path, "\\")
		if len(drive) != 2 || drive[1] != ':' {
			continue
		}

		fields := [...]struct {
			name   string
			metric *diskMetric
		}{
			{"% Idle Time", &info.idlePercent},
			{"Disk Transfers/sec", &info.transfersPerSec},
			{"Avg. Disk sec/Transfer", &info.responseSeconds},
			{"Disk Read Bytes/sec", &info.readBytesPerSec},
			{"Disk Write Bytes/sec", &info.writeBytesPerSec},
		}
		for _, field := range fields {
			path := "\\LogicalDisk(" + drive + ")\\" + field.name
			handle, err := query.add(path)
			if err == nil {
				counters = append(counters, diskCounter{handle: handle, metric: field.metric})
			}
		}
	}
	if len(counters) == 0 {
		return nil
	}

	for sample := 0; sample < 2; sample++ {
		if sample > 0 {
			time.Sleep(time.Second)
		}
		if err := query.collect(); err != nil {
			return err
		}
	}

	for _, counter := range counters {
		value, available := query.value(counter.handle)
		*counter.metric = diskMetric{value: value, available: available}
	}
	return nil
}
