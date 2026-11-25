package monitor

import (
	"fmt"
	"math"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type performanceQuery struct {
	pdh    *windows.LazyDLL
	handle windows.Handle
}

const (
	pdhFmtDouble   = 0x0200
	pdhFmtNoScale  = 0x1000
	pdhFmtNoCap100 = 0x8000
	pdhDoubleFlags = pdhFmtDouble | pdhFmtNoScale | pdhFmtNoCap100
)

type pdhDoubleValue struct {
	status uint32
	_      uint32
	value  float64
}

func (v pdhDoubleValue) valid() bool {
	return v.status <= 1 && !math.IsNaN(v.value) && !math.IsInf(v.value, 0)
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

func (q performanceQuery) sample() error {
	if err := q.collect(); err != nil {
		return err
	}
	time.Sleep(time.Second)
	return q.collect()
}

func (q performanceQuery) value(counter windows.Handle) (float64, bool) {
	var result pdhDoubleValue
	status, _, _ := q.pdh.NewProc("PdhGetFormattedCounterValue").Call(
		uintptr(counter), pdhDoubleFlags, 0, uintptr(unsafe.Pointer(&result)),
	)
	if uint32(status) != 0 || !result.valid() {
		return 0, false
	}
	return result.value, true
}

type performanceInstance struct {
	name  string
	value float64
}

func (q performanceQuery) values(counter windows.Handle) []performanceInstance {
	if counter == 0 {
		return nil
	}
	const pdhMoreData = 0x800007d2
	formatted := q.pdh.NewProc("PdhGetFormattedCounterArrayW")
	var size, count uint32
	status, _, _ := formatted.Call(uintptr(counter), pdhDoubleFlags,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if uint32(status) != pdhMoreData || size == 0 {
		return nil
	}

	buffer := make([]byte, size)
	status, _, _ = formatted.Call(uintptr(counter), pdhDoubleFlags,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buffer[0])))
	type arrayItem struct {
		name   *uint16
		_      [8 - unsafe.Sizeof(uintptr(0))]byte
		result pdhDoubleValue
	}
	if uint32(status) != 0 || size > uint32(len(buffer)) || uintptr(count) > uintptr(size)/unsafe.Sizeof(arrayItem{}) {
		return nil
	}

	var values []performanceInstance
	for _, item := range unsafe.Slice((*arrayItem)(unsafe.Pointer(&buffer[0])), count) {
		if item.name != nil && item.result.valid() {
			values = append(values, performanceInstance{windows.UTF16PtrToString(item.name), item.result.value})
		}
	}
	runtime.KeepAlive(buffer)
	return values
}
