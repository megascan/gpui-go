//go:build !windows

package scheduler

import "time"

func monotonicNow() time.Time {
	return time.Now()
}
