package monitor

import (
	"cmp"
	"fmt"
	"io"
)

func PrintMemory(out io.Writer) {
	info, err := readMemoryPerformance()
	if err != nil {
		PrintError(out, "read RAM usage", err)
		return
	}

	pageSize := uint64(info.pageSize)
	physicalTotalBytes := uint64(info.physicalTotal) * pageSize
	physicalTotal := float64(physicalTotalBytes) / gibibyte
	physicalAvailable := float64(uint64(info.physicalAvailable)*pageSize) / gibibyte
	physicalUsed := physicalTotal - physicalAvailable
	physicalUsedPercent := physicalUsed / physicalTotal * 100
	commitTotal := float64(uint64(info.commitTotal)*pageSize) / gibibyte
	commitLimit := float64(uint64(info.commitLimit)*pageSize) / gibibyte
	systemCache := float64(uint64(info.systemCache)*pageSize) / gibibyte
	pagedPool := float64(uint64(info.kernelPaged)*pageSize) / mibibyte
	nonPagedPool := float64(uint64(info.kernelNonPaged)*pageSize) / mibibyte

	bar, err := UsageBar(physicalUsedPercent)
	if err != nil {
		PrintError(out, "format RAM usage", err)
		return
	}

	hardware := readMemoryHardware()
	fmt.Fprintf(out, "Memory type: %s\n", cmp.Or(hardware.memoryType, unavailable))
	fmt.Fprintf(out, "In use: %.1f GiB [%s] (%.1f%%)\n", physicalUsed, bar, physicalUsedPercent)
	fmt.Fprintf(out, "Available: %.1f / %.1f GiB\n", physicalAvailable, physicalTotal)
	fmt.Fprintf(out, "Committed: %.1f / %.1f GiB\n", commitTotal, commitLimit)
	fmt.Fprintf(out, "Cached: %.1f GiB\n", systemCache)
	fmt.Fprintf(out, "Paged pool: %.1f MiB | Non-paged pool: %.1f MiB\n", pagedPool, nonPagedPool)
	fmt.Fprintf(out, "Speed: %s\n", cmp.Or(hardware.speed, unavailable))
	fmt.Fprintf(out, "Form factor: %s\n", cmp.Or(hardware.formFactor, unavailable))

	installedBytes, err := readInstalledMemory()
	if err != nil {
		PrintError(out, "read installed RAM", err)
		return
	}

	reservedBytes := installedBytes - min(installedBytes, physicalTotalBytes)
	fmt.Fprintf(out, "Hardware reserved: %.1f MiB\n", float64(reservedBytes)/mibibyte)
}
