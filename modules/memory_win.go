package modules

import (
	"cmp"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func PrintMem() {
	info, err := ReadMemoryPerformance()
	if err != nil {
		fmt.Println("GetPerformanceInfo:", err)
		return
	}

	pageSize := uint64(info.pageSize)
	physicalTotalBytes := uint64(info.physicalTotal) * pageSize
	physicalTotal := float64(physicalTotalBytes) / (1 << 30)
	physicalAvailable := float64(uint64(info.physicalAvailable)*pageSize) / (1 << 30)
	physicalUsed := physicalTotal - physicalAvailable
	physicalUsedPercent := physicalUsed / physicalTotal * 100
	commitTotal := float64(uint64(info.commitTotal)*pageSize) / (1 << 30)
	commitLimit := float64(uint64(info.commitLimit)*pageSize) / (1 << 30)
	systemCache := float64(uint64(info.systemCache)*pageSize) / (1 << 30)
	pagedPool := float64(uint64(info.kernelPaged)*pageSize) / (1 << 20)
	nonPagedPool := float64(uint64(info.kernelNonPaged)*pageSize) / (1 << 20)

	bar, err := UsageBar(physicalUsedPercent)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	hardware := readMemoryHardware()
	fmt.Printf("Memory type: %s\n", cmp.Or(hardware.memoryType, "unavailable"))
	fmt.Printf("In use: %.1f GiB [%s] (%.1f%%)\n", physicalUsed, bar, physicalUsedPercent)
	fmt.Printf("Available: %.1f / %.1f GiB\n", physicalAvailable, physicalTotal)
	fmt.Printf("Committed: %.1f / %.1f GiB\n", commitTotal, commitLimit)
	fmt.Printf("Cached: %.1f GiB\n", systemCache)
	fmt.Printf("Paged pool: %.1f MiB | Non-paged pool: %.1f MiB\n", pagedPool, nonPagedPool)
	fmt.Printf("Speed: %s | Form factor: %s\n",
		cmp.Or(hardware.speed, "unavailable"), cmp.Or(hardware.formFactor, "unavailable"))

	var installedKiB uint64
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getInstalledMemory := kernel32.NewProc("GetPhysicallyInstalledSystemMemory")

	ok, _, callErr := getInstalledMemory.Call(uintptr(unsafe.Pointer(&installedKiB)))
	if ok == 0 {
		fmt.Println("GetPhysicallyInstalledSystemMemory:", callErr)
		return
	}

	reservedBytes := installedKiB*1024 - physicalTotalBytes

	fmt.Printf("Hardware reserved: %.1f MiB\n", float64(reservedBytes)/(1<<20))
}
