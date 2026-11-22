package main

import (
	"fmt"
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

func PrintMem() {
	var info performanceInformation
	info.cb = uint32(unsafe.Sizeof(info))

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getPerformanceInfo := kernel32.NewProc("K32GetPerformanceInfo")

	ok, _, callErr := getPerformanceInfo.Call(
		uintptr(unsafe.Pointer(&info)),
		uintptr(info.cb),
	)
	if ok == 0 {
		fmt.Println("GetPerformanceInfo:", callErr)
		return
	}

	physicalTotal := float64(info.physicalTotal*info.pageSize) / (1 << 30)
	physicalAvailable := float64(info.physicalAvailable*info.pageSize) / (1 << 30)
	physicalUsed := physicalTotal - physicalAvailable
	physicalUsedPercent := physicalUsed / physicalTotal * 100
	commitTotal := float64(info.commitTotal*info.pageSize) / (1 << 30)
	commitLimit := float64(info.commitLimit*info.pageSize) / (1 << 30)
	systemCache := float64(info.systemCache*info.pageSize) / (1 << 30)
	pagedPool := float64(info.kernelPaged*info.pageSize) / (1 << 20)
	nonPagedPool := float64(info.kernelNonPaged*info.pageSize) / (1 << 20)

	bar, err := UsageBar(physicalUsedPercent)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("In use: %.1f GiB [%s] (%.1f%%)\n", physicalUsed, bar, physicalUsedPercent)
	fmt.Printf("Available: %.1f / %.1f GiB\n", physicalAvailable, physicalTotal)
	fmt.Printf("Committed: %.1f / %.1f GiB\n", commitTotal, commitLimit)
	fmt.Printf("Cached: %.1f GiB\n", systemCache)
	fmt.Printf("Paged pool: %.1f MiB | Non-paged pool: %.1f MiB\n", pagedPool, nonPagedPool)

	var installedKiB uint64
	getInstalledMemory := kernel32.NewProc("GetPhysicallyInstalledSystemMemory")

	ok, _, callErr = getInstalledMemory.Call(uintptr(unsafe.Pointer(&installedKiB)))
	if ok == 0 {
		fmt.Println("GetPhysicallyInstalledSystemMemory:", callErr)
		return
	}

	reservedBytes := installedKiB*1024 - uint64(info.physicalTotal)*uint64(info.pageSize)

	fmt.Printf("Hardware reserved: %.1f MiB\n", float64(reservedBytes)/(1<<20))
}
