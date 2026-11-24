package modules

import (
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

func ReadPerformanceInfo() (performanceInformation, error) {
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
