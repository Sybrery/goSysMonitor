package modules

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type performanceInformation struct {
	cb                uint32
	commitTotal       uintptr
	commitLimit       uintptr
	commitPeak        uintptr
	physicalTotal     uintptr
	physicalAvailable uintptr
	systemCache       uintptr
	kernelTotal       uintptr
	kernelPaged       uintptr
	kernelNonPaged    uintptr
	pageSize          uintptr
	handleCount       uint32
	processCount      uint32
	threadCount       uint32
}

func ReadMemoryPerformance() (performanceInformation, error) {
	var info performanceInformation
	info.cb = uint32(unsafe.Sizeof(info))

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getPerformanceInfo := kernel32.NewProc("K32GetPerformanceInfo")

	ok, _, callErr := getPerformanceInfo.Call(
		uintptr(unsafe.Pointer(&info)),
		uintptr(info.cb),
	)
	if ok == 0 {
		return info, callErr
	}
	return info, nil
}

type memoryHardware struct {
	memoryType string
	speed      string
	formFactor string
}

func readMemoryHardware() memoryHardware {
	const (
		rsmb       = 0x52534d42 // 'RSMB', the raw SMBIOS provider
		headerSize = 8
	)
	firmware := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemFirmwareTable")
	size, _, _ := firmware.Call(rsmb, 0, 0, 0)
	if size < headerSize {
		return memoryHardware{}
	}
	buf := make([]byte, size)
	read, _, _ := firmware.Call(rsmb, 0, uintptr(unsafe.Pointer(&buf[0])), size)
	if read < headerSize || read > size {
		return memoryHardware{}
	}
	length := uintptr(binary.LittleEndian.Uint32(buf[4:8]))
	if length > read-headerSize {
		return memoryHardware{}
	}
	return parseMemoryHardware(buf[headerSize : headerSize+length])
}

func parseMemoryHardware(table []byte) memoryHardware {
	memoryTypes := map[byte]string{
		0x03: "DRAM",
		0x0f: "SDRAM",
		0x12: "DDR",
		0x13: "DDR2",
		0x14: "DDR2 FB-DIMM",
		0x18: "DDR3",
		0x1a: "DDR4",
		0x1b: "LPDDR",
		0x1c: "LPDDR2",
		0x1d: "LPDDR3",
		0x1e: "LPDDR4",
		0x22: "DDR5",
		0x23: "LPDDR5",
	}
	formFactors := map[byte]string{
		0x03: "SIMM",
		0x04: "SIP",
		0x05: "Chip",
		0x06: "DIP",
		0x07: "ZIP",
		0x08: "Proprietary card",
		0x09: "DIMM",
		0x0a: "TSOP",
		0x0b: "Row of chips",
		0x0c: "RIMM",
		0x0d: "SODIMM",
		0x0e: "SRIMM",
		0x0f: "FB-DIMM",
		0x10: "Die",
		0x11: "CAMM",
		0x12: "CUDIMM",
		0x13: "CSODIMM",
	}

	var types, speeds, factors []string
	for len(table) >= 4 {
		kind, length := table[0], int(table[1])
		if kind == 127 || length < 4 || length > len(table) {
			break
		}
		// Each formatted record is followed by strings ending in two NUL bytes.
		end := bytes.Index(table[length:], []byte{0, 0})
		if end < 0 {
			break
		}
		record := table[:length]
		table = table[length+end+2:]
		// Type 17 describes a memory device. A zero size means an empty slot.
		if kind != 17 || length < 0x13 || binary.LittleEndian.Uint16(record[0x0c:0x0e]) == 0 {
			continue
		}

		types = appendMemoryValue(types, memoryTypes[record[0x12]])
		factors = appendMemoryValue(factors, formFactors[record[0x0e]])
		if speed := readMemorySpeed(record, 0x20, 0x58); speed != 0 {
			speeds = appendMemoryValue(speeds, fmt.Sprintf("%d MT/s", speed))
		} else if ratedSpeed := readMemorySpeed(record, 0x15, 0x54); ratedSpeed != 0 {
			speeds = appendMemoryValue(speeds, fmt.Sprintf("%d MT/s (rated)", ratedSpeed))
		}
	}
	return memoryHardware{
		memoryType: strings.Join(types, ", "),
		speed:      strings.Join(speeds, ", "),
		formFactor: strings.Join(factors, ", "),
	}
}

func readMemorySpeed(record []byte, offset, extendedOffset int) uint32 {
	if len(record) < offset+2 {
		return 0
	}
	speed := binary.LittleEndian.Uint16(record[offset : offset+2])
	if speed != 0xffff {
		return uint32(speed)
	}
	if len(record) < extendedOffset+4 {
		return 0
	}
	return binary.LittleEndian.Uint32(record[extendedOffset:extendedOffset+4]) & 0x7fffffff
}

func appendMemoryValue(values []string, value string) []string {
	if value == "" || slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
