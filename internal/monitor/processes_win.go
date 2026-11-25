package monitor

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/windows"
)

type processView struct {
	title  string
	column string
	unit   string
	scale  float64
	sample func() (map[uint32]float64, error)
}

func OpenProcessView(resource string) (func(io.Writer), func()) {
	closeView := func() {}
	var view processView
	switch resource {
	case "-c":
		count := max(1, windows.GetActiveProcessorCount(0xffff))
		view = processView{"Top CPU processes", "CPU", "%", 1 / float64(count), func() (map[uint32]float64, error) {
			return readProcessCounter("% Processor Time", true)
		}}
	case "-m":
		view = processView{"Top RAM processes", "RAM", " MiB", 1.0 / mibibyte, func() (map[uint32]float64, error) {
			return readProcessCounter("Working Set", false)
		}}
	case "-g":
		view = processView{"Top GPU processes", "GPU", "%", 1, readGPUProcessUsage}
	case "-d", "-w":
		trace, err := openProcessTrace(resource)
		if err == nil {
			closeView = trace.close
			if resource == "-d" {
				view = processView{"Top disk processes", "Disk I/O", " MiB/s", 1.0 / mibibyte, trace.sample}
			} else {
				view = processView{"Top Wi-Fi processes", "Wi-Fi", " Kbps", 8.0 / 1000, trace.sample}
			}
		} else if resource == "-d" {
			view = processView{"Top processes by total I/O (file, network, and device)", "I/O", " MiB/s", 1.0 / mibibyte, func() (map[uint32]float64, error) {
				return readProcessCounter("IO Data Bytes/sec", true)
			}}
			return func(out io.Writer) {
				PrintError(out, "read disk activity by process", err)
				view.print(out)
			}, closeView
		} else {
			return func(out io.Writer) { PrintError(out, "read Wi-Fi activity by process", err) }, closeView
		}
	}
	return view.print, closeView
}

func (view processView) print(out io.Writer) {
	fmt.Fprintln(out, view.title)
	values, err := view.sample()
	if err != nil {
		PrintError(out, "read process activity", err)
		return
	}
	type processRow struct {
		pid   uint32
		value float64
	}
	rows := make([]processRow, 0, len(values))
	for pid, value := range values {
		if pid != 0 && value > 0 {
			rows = append(rows, processRow{pid, value})
		}
	}
	if len(rows) == 0 {
		fmt.Fprintln(out, "No process activity measured")
		return
	}

	slices.SortFunc(rows, func(a, b processRow) int {
		if order := cmp.Compare(b.value, a.value); order != 0 {
			return order
		}
		return cmp.Compare(a.pid, b.pid)
	})

	names := make(map[uint32]string)
	if processes, err := process.Processes(); err == nil {
		for _, running := range processes {
			name, _ := running.Name()
			names[uint32(running.Pid)] = name
		}
	}
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(table, "PID\tProcess\t%s\n", view.column)
	for _, row := range rows[:min(10, len(rows))] {
		value := row.value * view.scale
		if view.unit == "%" {
			value = min(100, value)
		}
		fmt.Fprintf(table, "%d\t%s\t%.1f%s\n", row.pid, cmp.Or(names[row.pid], unavailable), value, view.unit)
	}
	table.Flush()
}

func readProcessCounter(counterName string, rate bool) (map[uint32]float64, error) {
	query, err := openPerformanceQuery()
	if err != nil {
		return nil, err
	}
	defer query.close()

	counter, err := query.add("\\Process V2(*)\\" + counterName)
	var ids windows.Handle
	if err != nil {
		counter, err = query.add("\\Process(*)\\" + counterName)
		if err != nil {
			return nil, err
		}
		ids, err = query.add("\\Process(*)\\ID Process")
		if err != nil {
			return nil, err
		}
	}
	if err := query.collect(); err != nil {
		return nil, err
	}
	before := make(map[string]uint32)
	if rate {
		for _, id := range query.values(ids) {
			before[id.name] = uint32(id.value)
		}
		time.Sleep(time.Second)
		if err := query.collect(); err != nil {
			return nil, err
		}
	}
	after := make(map[string]uint32)
	for _, id := range query.values(ids) {
		after[id.name] = uint32(id.value)
	}

	values := make(map[uint32]float64)
	for _, sample := range query.values(counter) {
		pid := after[sample.name]
		if ids == 0 {
			_, suffix, found := strings.Cut(sample.name, ":")
			if !found {
				continue
			}
			number, _ := strconv.ParseUint(suffix, 10, 32)
			pid = uint32(number)
		} else if rate && before[sample.name] != pid {
			continue
		}
		if pid != 0 {
			values[pid] = sample.value
		}
	}
	return values, nil
}

func readGPUProcessUsage() (map[uint32]float64, error) {
	query, err := openPerformanceQuery()
	if err != nil {
		return nil, err
	}
	defer query.close()
	counter, err := query.add("\\GPU Engine(*)\\Utilization Percentage")
	if err != nil {
		return nil, err
	}
	if err := query.sample(); err != nil {
		return nil, err
	}

	values := make(map[uint32]float64)
	for _, sample := range query.values(counter) {
		name, _ := strings.CutPrefix(sample.name, "pid_")
		pid, _, _ := strings.Cut(name, "_")
		if number, err := strconv.ParseUint(pid, 10, 32); err == nil && number != 0 {
			id := uint32(number)
			values[id] = max(values[id], sample.value)
		}
	}
	return values, nil
}
