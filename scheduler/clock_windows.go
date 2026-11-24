package scheduler

import (
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procQueryPerformanceCounter   = kernel32.NewProc("QueryPerformanceCounter")
	procQueryPerformanceFrequency = kernel32.NewProc("QueryPerformanceFrequency")
)

type performanceClock struct {
	base      time.Time
	start     int64
	frequency int64
}

var loadPerformanceClock = sync.OnceValue(func() performanceClock {
	var frequency int64
	if r, _, _ := syscall.SyscallN(procQueryPerformanceFrequency.Addr(), uintptr(unsafe.Pointer(&frequency))); r == 0 || frequency <= 0 {
		return performanceClock{}
	}
	start, ok := queryPerformanceCounter()
	if !ok {
		return performanceClock{}
	}
	return performanceClock{base: time.Now(), start: start, frequency: frequency}
})

func queryPerformanceCounter() (int64, bool) {
	var counter int64
	r, _, _ := syscall.SyscallN(procQueryPerformanceCounter.Addr(), uintptr(unsafe.Pointer(&counter)))
	return counter, r != 0
}

func monotonicNow() time.Time {
	clock := loadPerformanceClock()
	if clock.frequency == 0 {
		return time.Now()
	}
	counter, ok := queryPerformanceCounter()
	if !ok {
		return time.Now()
	}
	ticks := counter - clock.start
	seconds := ticks / clock.frequency
	remainder := ticks % clock.frequency
	return clock.base.Add(time.Duration(seconds)*time.Second + time.Duration(remainder*int64(time.Second)/clock.frequency))
}
