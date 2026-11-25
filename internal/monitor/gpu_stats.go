package monitor

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type gpuInformation struct {
	model         string
	luid          string
	pciAddress    [3]uint32
	engine3D      gpuEngineUsage
	compute       gpuEngineUsage
	copy          gpuEngineUsage
	videoEncode   gpuEngineUsage
	videoDecode   gpuEngineUsage
	videoCodec    gpuEngineUsage
	utilization   gpuEngineUsage
	dedicated     gpuMemoryUsage
	shared        gpuMemoryUsage
	capacityKnown bool
	temperature   float64
	reservedBytes uint64
	reservedKnown bool
}

type gpuEngineUsage struct {
	percent   float64
	available bool
}

func (usage *gpuEngineUsage) update(percent float64) {
	usage.percent = max(usage.percent, min(100, percent))
	usage.available = true
}

type gpuMemoryUsage struct {
	used      float64
	capacity  uint64
	available bool
}

func readGPUPerformance() ([]gpuInformation, error) {
	gpus, err := readGPUAdapters()
	if err != nil || len(gpus) == 0 {
		return gpus, err
	}
	readGPUCounters(gpus)
	return gpus, nil
}

func readGPUAdapters() ([]gpuInformation, error) {
	const (
		queryMemorySizes   = 3
		queryPCIAddress    = 6
		queryAdapterName   = 8
		queryAdapterType   = 15
		queryRegistry      = 48
		queryPerformance   = 62
		adapterRegistryKey = 1
		renderSupported    = 1
		softwareDevice     = 1 << 2
	)
	gdi := windows.NewLazySystemDLL("gdi32.dll")
	enumerate := gdi.NewProc("D3DKMTEnumAdapters2")
	if err := enumerate.Find(); err != nil {
		return nil, err
	}
	type adapter struct {
		handle    uint32
		luid      windows.LUID
		sources   uint32
		preferred uint32
	}
	var list struct {
		count    uint32
		adapters *adapter
	}
	status, _, _ := enumerate.Call(uintptr(unsafe.Pointer(&list)))
	if uint32(status) != 0 {
		return nil, fmt.Errorf("D3DKMTEnumAdapters2: 0x%08x", uint32(status))
	}
	if list.count == 0 {
		return nil, nil
	}
	adapters := make([]adapter, list.count)
	list.adapters = &adapters[0]
	status, _, _ = enumerate.Call(uintptr(unsafe.Pointer(&list)))
	if uint32(status) != 0 {
		return nil, fmt.Errorf("D3DKMTEnumAdapters2: 0x%08x", uint32(status))
	}

	query := gdi.NewProc("D3DKMTQueryAdapterInfo")
	closeAdapter := gdi.NewProc("D3DKMTCloseAdapter")
	var gpus []gpuInformation
	for _, adapter := range adapters[:list.count] {
		defer func(handle uint32) {
			closeAdapter.Call(uintptr(unsafe.Pointer(&handle)))
		}(adapter.handle)

		var flags uint32
		if !queryGPUAdapter(query, adapter.handle, queryAdapterType, unsafe.Pointer(&flags), unsafe.Sizeof(flags)) ||
			flags&renderSupported == 0 || flags&softwareDevice != 0 {
			continue
		}

		info := gpuInformation{
			luid:       fmt.Sprintf("0x%08x_0x%08x", uint32(adapter.luid.HighPart), adapter.luid.LowPart),
			pciAddress: [3]uint32{^uint32(0), ^uint32(0), ^uint32(0)},
		}
		var name [4][windows.MAX_PATH]uint16
		if queryGPUAdapter(query, adapter.handle, queryAdapterName, unsafe.Pointer(&name), unsafe.Sizeof(name)) {
			info.model = windows.UTF16ToString(name[0][:])
		}
		var address [3]uint32
		if queryGPUAdapter(query, adapter.handle, queryPCIAddress, unsafe.Pointer(&address), unsafe.Sizeof(address)) {
			info.pciAddress = address
		}
		var memory [3]uint64
		if queryGPUAdapter(query, adapter.handle, queryMemorySizes, unsafe.Pointer(&memory), unsafe.Sizeof(memory)) {
			info.dedicated.capacity = memory[0] + memory[1]
			info.shared.capacity = memory[2]
			info.capacityKnown = true

			installed := struct {
				kind      uint32
				_         uint32
				name      [windows.MAX_PATH]uint16
				valueType uint32
				_         uint32
				valueSize uint32
				status    uint32
				value     uint64
			}{kind: adapterRegistryKey, valueType: windows.REG_QWORD}
			propertyName, _ := windows.UTF16FromString("HardwareInformation.qwMemorySize")
			copy(installed.name[:], propertyName)
			if queryGPUAdapter(query, adapter.handle, queryRegistry, unsafe.Pointer(&installed), unsafe.Sizeof(installed)) &&
				installed.status == 0 && installed.valueSize == uint32(unsafe.Sizeof(installed.value)) &&
				installed.value >= info.dedicated.capacity {
				info.reservedBytes = installed.value - info.dedicated.capacity
				info.reservedKnown = true
			}
		}
		var performance struct {
			physicalIndex uint32
			_             uint32
			_             [5]uint64
			_             uint32
			_             uint32
			temperature   uint32
			_             [4]byte
		}
		if queryGPUAdapter(query, adapter.handle, queryPerformance, unsafe.Pointer(&performance), unsafe.Sizeof(performance)) &&
			performance.temperature != ^uint32(0) {
			info.temperature = float64(performance.temperature) / 10
		}
		gpus = append(gpus, info)
	}

	slices.SortStableFunc(gpus, func(a, b gpuInformation) int {
		return slices.Compare(a.pciAddress[:], b.pciAddress[:])
	})
	return gpus, nil
}

func queryGPUAdapter(query *windows.LazyProc, handle, kind uint32, data unsafe.Pointer, size uintptr) bool {
	args := struct {
		handle uint32
		kind   uint32
		data   unsafe.Pointer
		size   uint32
	}{handle, kind, data, uint32(size)}
	status, _, _ := query.Call(uintptr(unsafe.Pointer(&args)))
	return uint32(status) == 0
}

func readGPUCounters(gpus []gpuInformation) {
	query, err := openPerformanceQuery()
	if err != nil {
		return
	}
	defer query.close()

	engines, _ := query.add("\\GPU Engine(*)\\Utilization Percentage")
	dedicated, _ := query.add("\\GPU Adapter Memory(*)\\Dedicated Usage")
	shared, _ := query.add("\\GPU Adapter Memory(*)\\Shared Usage")
	if engines == 0 && dedicated == 0 && shared == 0 {
		return
	}
	if query.sample() != nil {
		return
	}

	adapters := make(map[string]*gpuInformation, len(gpus))
	for i := range gpus {
		adapters[gpus[i].luid] = &gpus[i]
	}
	addGPUEngineUsage(adapters, query.values(engines))
	for _, usage := range query.values(dedicated) {
		if gpu := gpuForCounter(adapters, usage.name); gpu != nil && usage.value >= 0 {
			gpu.dedicated.used += usage.value
			gpu.dedicated.available = true
		}
	}
	for _, usage := range query.values(shared) {
		if gpu := gpuForCounter(adapters, usage.name); gpu != nil && usage.value >= 0 {
			gpu.shared.used += usage.value
			gpu.shared.available = true
		}
	}
}

func addGPUEngineUsage(adapters map[string]*gpuInformation, values []performanceInstance) {
	engines := make(map[string]float64)
	for _, sample := range values {
		_, engine, found := strings.Cut(strings.ToLower(sample.name), "_luid_")
		if found && sample.value >= 0 {
			engines["luid_"+engine] += sample.value
		}
	}
	for engine, percent := range engines {
		gpu := gpuForCounter(adapters, engine)
		if gpu == nil {
			continue
		}
		gpu.utilization.update(percent)
		_, kind, _ := strings.Cut(engine, "_engtype_")
		kind = strings.ReplaceAll(kind, " ", "")
		switch {
		case strings.HasSuffix(kind, "3d"):
			gpu.engine3D.update(percent)
		case strings.Contains(kind, "compute"):
			gpu.compute.update(percent)
		case strings.HasPrefix(kind, "copy"):
			gpu.copy.update(percent)
		case strings.HasPrefix(kind, "videoencode"):
			gpu.videoEncode.update(percent)
		case strings.HasPrefix(kind, "videodecode"):
			gpu.videoDecode.update(percent)
		case strings.HasPrefix(kind, "videocodec"):
			gpu.videoCodec.update(percent)
		}
	}
}

func gpuForCounter(adapters map[string]*gpuInformation, name string) *gpuInformation {
	luid, _, found := strings.Cut(strings.TrimPrefix(strings.ToLower(name), "luid_"), "_phys_")
	if !found {
		return nil
	}
	return adapters[luid]
}
