package monitor

import (
	"cmp"
	"fmt"
	"io"
)

func PrintGPU(out io.Writer) {
	gpus, err := readGPUPerformance()
	if err != nil {
		PrintError(out, "read GPU information", err)
		return
	}
	if len(gpus) == 0 {
		fmt.Fprintln(out, "No GPUs found")
		return
	}

	for i, gpu := range gpus {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "GPU %d: %s\n", i, cmp.Or(gpu.model, "Unknown adapter"))

		encodeLabel, encodeUsage := "Video encode", gpu.videoEncode
		if !gpu.videoEncode.available && gpu.videoCodec.available {
			encodeLabel, encodeUsage = "Video codec", gpu.videoCodec
		}
		engines := [...]struct {
			name  string
			usage gpuEngineUsage
		}{
			{"3D", gpu.engine3D},
			{"Copy", gpu.copy},
			{encodeLabel, encodeUsage},
			{"Video decode", gpu.videoDecode},
		}
		for _, engine := range engines {
			if engine.usage.available {
				bar, _ := UsageBar(engine.usage.percent)
				fmt.Fprintf(out, "%s: [%s] %.1f%%", engine.name, bar, engine.usage.percent)
			} else {
				fmt.Fprintf(out, "%s: unavailable", engine.name)
			}
			if engine.name == "3D" && gpu.compute.available {
				fmt.Fprintf(out, " | Compute: %.1f%%", gpu.compute.percent)
			}
			fmt.Fprintln(out)
		}

		memory := [...]struct {
			name  string
			usage gpuMemoryUsage
		}{
			{"Dedicated GPU memory", gpu.dedicated},
			{"Shared GPU memory", gpu.shared},
		}
		for _, item := range memory {
			fmt.Fprintf(out, "%s: %s", item.name, formatGPUMemory(item.usage, gpu.capacityKnown))
			if item.usage.available && gpu.capacityKnown {
				percent := 0.0
				if item.usage.capacity > 0 {
					percent = min(100, item.usage.used/float64(item.usage.capacity)*100)
				}
				bar, _ := UsageBar(percent)
				fmt.Fprintf(out, " [%s] (%.1f%%)", bar, percent)
			}
			fmt.Fprintln(out)
		}

		utilization := unavailable
		if gpu.utilization.available {
			utilization = fmt.Sprintf("%.1f%%", gpu.utilization.percent)
		}
		totalMemory := gpuMemoryUsage{
			used:      gpu.dedicated.used + gpu.shared.used,
			capacity:  gpu.dedicated.capacity + gpu.shared.capacity,
			available: gpu.dedicated.available && gpu.shared.available,
		}
		temperature := unavailable
		if gpu.temperature > 0 {
			temperature = fmt.Sprintf("%.1f °C", gpu.temperature)
		}
		reserved := unavailable
		if gpu.reservedKnown {
			reserved = fmt.Sprintf("%.1f MiB", float64(gpu.reservedBytes)/mibibyte)
		}
		fmt.Fprintf(out, "Utilization: %s | GPU memory: %s\n", utilization, formatGPUMemory(totalMemory, gpu.capacityKnown))
		fmt.Fprintf(out, "Temperature: %s\n", temperature)
		fmt.Fprintf(out, "Hardware reserved: %s\n", reserved)
	}
}

func formatGPUMemory(usage gpuMemoryUsage, capacityKnown bool) string {
	if !usage.available {
		return unavailable
	}
	usedGiB := usage.used / gibibyte
	if !capacityKnown {
		return fmt.Sprintf("%.1f GiB", usedGiB)
	}
	return fmt.Sprintf("%.1f / %.1f GiB", usedGiB, float64(usage.capacity)/gibibyte)
}
