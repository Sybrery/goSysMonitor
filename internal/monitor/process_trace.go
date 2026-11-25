package monitor

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"os"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	diskIOProvider = windows.GUID{Data1: 0x3d6fa8d4, Data2: 0xfe05, Data3: 0x11d0, Data4: [8]byte{0x9d, 0xda, 0, 0xc0, 0x4f, 0xd7, 0xba, 0x7c}}
	threadProvider = windows.GUID{Data1: 0x3d6fa8d1, Data2: 0xfe05, Data3: 0x11d0, Data4: [8]byte{0x9d, 0xda, 0, 0xc0, 0x4f, 0xd7, 0xba, 0x7c}}
	tcpProvider    = windows.GUID{Data1: 0x9a280ac0, Data2: 0xc8e0, Data3: 0x11d1, Data4: [8]byte{0x84, 0xe2, 0, 0xc0, 0x4f, 0xb9, 0x98, 0xa2}}
	udpProvider    = windows.GUID{Data1: 0xbf3a50c5, Data2: 0xa9c9, Data3: 0x4988, Data4: [8]byte{0xa0, 0x05, 0x2d, 0xf0, 0xb7, 0xc8, 0x0f, 0x80}}
)

type processTrace struct {
	api        *windows.LazyDLL
	properties *eventTraceProperties
	controller uint64
	consumer   uint64
	done       chan struct{}
	err        error
	wifi       bool
	threads    map[uint32]uint32
	mu         sync.Mutex
	addresses  map[netip.Addr]struct{}
	totals     map[uint32]float64
	started    time.Time
}

type eventTraceProperties struct {
	size          uint32
	_             uint32
	_             uint64
	_             int64
	guid          windows.GUID
	clientContext uint32
	flags         uint32
	_             [4]uint32
	mode          uint32
	flushTimer    uint32
	enableFlags   uint32
	_             [7]uint32
	_             windows.Handle
	_             uint32
	nameOffset    uint32
	name          [128]uint16
}

const (
	eventTraceSize         = 88
	traceLogfileHeaderSize = 280
	eventHeader32Bit       = 0x0020
)

type eventTraceLogfile struct {
	_        *uint16
	name     *uint16
	_        int64
	_        uint32
	mode     uint32
	_        [eventTraceSize]byte
	_        [traceLogfileHeaderSize]byte
	_        uintptr
	_        [3]uint32
	callback uintptr
	_        uint32
	context  unsafe.Pointer
}

type eventRecord struct {
	_        uint16
	_        uint16
	flags    uint16
	_        uint16
	_        uint32
	_        uint32
	_        int64
	provider windows.GUID
	_        uint16
	version  uint8
	_        uint8
	_        uint8
	opcode   uint8
	_        uint16
	_        uint64
	_        uint64
	_        windows.GUID
	_        uint32
	_        uint16
	dataSize uint16
	_        unsafe.Pointer
	data     unsafe.Pointer
	context  unsafe.Pointer
}

var processTraceCallback = windows.NewCallback(func(event *eventRecord) uintptr {
	if event.context != nil && event.data != nil {
		trace := (*processTrace)(event.context)
		trace.record(event.provider, event.opcode, event.version, event.flags, unsafe.Slice((*byte)(event.data), event.dataSize))
	}
	return 0
})

func openProcessTrace(resource string) (*processTrace, error) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return nil, fmt.Errorf("Windows kernel tracing requires a 64-bit build")
	}
	const (
		traceDiskIO      = 0x00000100
		traceThreads     = 0x00000002
		traceNetwork     = 0x00010000
		tracedGUID       = 0x00020000
		realTimeMode     = 0x00000100
		systemLoggerMode = 0x02000000
		eventRecordMode  = 0x10000000
	)
	trace := &processTrace{
		api:     windows.NewLazySystemDLL("advapi32.dll"),
		done:    make(chan struct{}),
		wifi:    resource == "-w",
		threads: make(map[uint32]uint32),
		totals:  make(map[uint32]float64),
	}
	flags := uint32(traceDiskIO | traceThreads)
	if trace.wifi {
		flags = traceNetwork
		trace.addresses = activeWiFiAddresses()
		if len(trace.addresses) == 0 {
			return nil, fmt.Errorf("no Wi-Fi adapter with an active IP address")
		}
	}
	id, err := windows.GenerateGUID()
	if err != nil {
		return nil, err
	}
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf("GolangSystemMonitor-%d-%s", os.Getpid(), id.String()))
	properties := &eventTraceProperties{
		guid: id, clientContext: 1, flags: tracedGUID, mode: systemLoggerMode | realTimeMode,
		flushTimer: 1, enableFlags: flags,
	}
	properties.size = uint32(unsafe.Sizeof(*properties))
	properties.nameOffset = uint32(unsafe.Offsetof(properties.name))
	trace.properties = properties
	status, _, _ := trace.api.NewProc("StartTraceW").Call(
		uintptr(unsafe.Pointer(&trace.controller)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(properties)),
	)
	if uint32(status) != 0 {
		return nil, fmt.Errorf("Windows kernel tracing: %w", windows.Errno(status))
	}

	log := eventTraceLogfile{name: name, mode: eventRecordMode | realTimeMode, callback: processTraceCallback, context: unsafe.Pointer(trace)}
	handle, _, callErr := trace.api.NewProc("OpenTraceW").Call(uintptr(unsafe.Pointer(&log)))
	if handle == ^uintptr(0) {
		trace.stopController()
		return nil, fmt.Errorf("OpenTraceW: %w", callErr)
	}
	trace.consumer = uint64(handle)
	trace.started = time.Now()
	go func() {
		defer close(trace.done)
		status, _, _ := trace.api.NewProc("ProcessTrace").Call(uintptr(unsafe.Pointer(&trace.consumer)), 1, 0, 0)
		if uint32(status) != 0 {
			trace.err = fmt.Errorf("ProcessTrace: %w", windows.Errno(status))
		}
	}()
	return trace, nil
}

func (t *processTrace) stopController() {
	const stopSession = 1
	t.api.NewProc("ControlTraceW").Call(uintptr(t.controller), 0, uintptr(unsafe.Pointer(t.properties)), stopSession)
}

func (t *processTrace) close() {
	t.stopController()
	t.api.NewProc("CloseTrace").Call(uintptr(t.consumer))
	<-t.done
}

func (t *processTrace) sample() (map[uint32]float64, error) {
	timer := time.NewTimer(time.Until(t.started.Add(time.Second)))
	defer timer.Stop()
	select {
	case <-t.done:
		if t.err != nil {
			return nil, t.err
		}
		return nil, fmt.Errorf("Windows ended the tracing session")
	case <-timer.C:
	}

	var addresses map[netip.Addr]struct{}
	if t.wifi {
		addresses = activeWiFiAddresses()
	}
	t.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(t.started).Seconds()
	t.started = now
	values := t.totals
	t.totals = make(map[uint32]float64)
	t.addresses = addresses
	t.mu.Unlock()
	for pid, size := range values {
		values[pid] = size / elapsed
	}
	return values, nil
}

func (t *processTrace) record(provider windows.GUID, opcode, version uint8, flags uint16, data []byte) {
	switch provider {
	case threadProvider:
		if len(data) >= 8 && (opcode == 1 || opcode == 2 || opcode == 3) {
			tid := binary.LittleEndian.Uint32(data[4:8])
			if opcode == 2 {
				delete(t.threads, tid)
			} else {
				t.threads[tid] = binary.LittleEndian.Uint32(data[:4])
			}
		}
	case diskIOProvider:
		if version < 3 || (opcode != 10 && opcode != 11) {
			return
		}
		threadOffset := 48
		if flags&eventHeader32Bit != 0 {
			threadOffset = 40
		}
		if len(data) >= threadOffset+4 {
			pid := t.threads[binary.LittleEndian.Uint32(data[threadOffset:])]
			if pid != 0 {
				t.mu.Lock()
				t.totals[pid] += float64(binary.LittleEndian.Uint32(data[8:12]))
				t.mu.Unlock()
			}
		}
	case tcpProvider, udpProvider:
		var addrSize int
		switch opcode {
		case 10, 11:
			addrSize = 4
		case 26, 27:
			addrSize = 16
		default:
			return
		}
		if version < 2 || len(data) < 8+2*addrSize {
			return
		}
		dest, _ := netip.AddrFromSlice(data[8 : 8+addrSize])
		src, _ := netip.AddrFromSlice(data[8+addrSize : 8+2*addrSize])
		t.mu.Lock()
		_, fromWiFi := t.addresses[src.Unmap()]
		_, toWiFi := t.addresses[dest.Unmap()]
		if fromWiFi != toWiFi {
			pid := binary.LittleEndian.Uint32(data[:4])
			t.totals[pid] += float64(binary.LittleEndian.Uint32(data[4:8]))
		}
		t.mu.Unlock()
	}
}

func activeWiFiAddresses() map[netip.Addr]struct{} {
	adapters, _ := readWiFiAdapters()
	addresses := make(map[netip.Addr]struct{})
	for _, adapter := range adapters {
		if adapter.OperStatus != windows.IfOperStatusUp {
			continue
		}
		for _, ip := range wifiInterfaceIPs(adapter.InterfaceIndex) {
			if addr, valid := netip.AddrFromSlice(ip); valid {
				addresses[addr.Unmap()] = struct{}{}
			}
		}
	}
	return addresses
}
