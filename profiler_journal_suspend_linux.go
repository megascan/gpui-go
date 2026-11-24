package gpui

import (
	"time"

	"golang.org/x/sys/unix"
)

func init() {
	systemSuspendedTime = linuxSuspendedTime
}

func linuxSuspendedTime() (time.Duration, bool) {
	var monotonic, boottime unix.Timespec
	if unix.ClockGettime(unix.CLOCK_MONOTONIC, &monotonic) != nil {
		return 0, false
	}
	if unix.ClockGettime(unix.CLOCK_BOOTTIME, &boottime) != nil {
		return 0, false
	}
	suspended := time.Duration(boottime.Nano() - monotonic.Nano())
	if suspended < 0 {
		return 0, true
	}
	return suspended, true
}
