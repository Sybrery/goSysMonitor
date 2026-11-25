package modules

import (
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

type performanceQuery struct {
	pdh    *windows.LazyDLL
	handle windows.Handle
}

func openPerformanceQuery() (performanceQuery, error) {
	pdh := windows.NewLazySystemDLL("pdh.dll")
	open := pdh.NewProc("PdhOpenQueryW")
	if err := open.Find(); err != nil {
		return performanceQuery{}, err
	}
	var handle windows.Handle
	status, _, _ := open.Call(0, 0, uintptr(unsafe.Pointer(&handle)))
	if uint32(status) != 0 {
		return performanceQuery{}, fmt.Errorf("PdhOpenQueryW: 0x%08x", uint32(status))
	}
	return performanceQuery{pdh: pdh, handle: handle}, nil
}

func (q performanceQuery) close() {
	q.pdh.NewProc("PdhCloseQuery").Call(uintptr(q.handle))
}

func (q performanceQuery) add(path string) (windows.Handle, error) {
	counterPath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var counter windows.Handle
	status, _, _ := q.pdh.NewProc("PdhAddEnglishCounterW").Call(
		uintptr(q.handle), uintptr(unsafe.Pointer(counterPath)), 0, uintptr(unsafe.Pointer(&counter)),
	)
	if uint32(status) != 0 {
		return 0, fmt.Errorf("PdhAddEnglishCounterW: 0x%08x", uint32(status))
	}
	return counter, nil
}

func (q performanceQuery) collect() error {
	status, _, _ := q.pdh.NewProc("PdhCollectQueryData").Call(uintptr(q.handle))
	if uint32(status) != 0 {
		return fmt.Errorf("PdhCollectQueryData: 0x%08x", uint32(status))
	}
	return nil
}

func (q performanceQuery) value(counter windows.Handle) (float64, bool) {
	const (
		pdhFmtDouble   = 0x0200
		pdhFmtNoScale  = 0x1000
		pdhFmtNoCap100 = 0x8000
	)
	// The double in PDH_FMT_COUNTERVALUE starts at byte 8.
	var result struct {
		status uint32
		_      uint32
		value  float64
	}
	status, _, _ := q.pdh.NewProc("PdhGetFormattedCounterValue").Call(
		uintptr(counter), pdhFmtDouble|pdhFmtNoScale|pdhFmtNoCap100, 0, uintptr(unsafe.Pointer(&result)),
	)
	// Counter status 0 and 1 indicate valid and new data, respectively.
	if uint32(status) != 0 || result.status > 1 || math.IsNaN(result.value) || math.IsInf(result.value, 0) {
		return 0, false
	}
	return result.value, true
}
