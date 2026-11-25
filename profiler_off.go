//go:build !gpui_profiler

package gpui

import "github.com/megascan/gpui-go/scheduler"

const profilerFeatureEnabled = false

// GetAllTimings returns no timings without the gpui_profiler build tag.
func GetAllTimings(_ TasksIncluded) []ThreadTaskTimings {
	return []ThreadTaskTimings{}
}

// GetCurrentThreadTimings returns empty timings for thread without the gpui_profiler build tag.
func GetCurrentThreadTimings(thread ThreadID, _ TasksIncluded) ThreadTaskTimings {
	return ThreadTaskTimings{
		ThreadName:  None[string](),
		ThreadID:    thread,
		Timings:     []TaskTiming{},
		Stats:       DefaultTaskStatistics(),
		TotalPushed: 0,
	}
}

// TakeAllStats returns no statistics without the gpui_profiler build tag.
func TakeAllStats(_ TasksIncluded) []ThreadTaskStatistics {
	return []ThreadTaskStatistics{}
}

// UpdateRunningTask does nothing without the gpui_profiler build tag.
func (t *ThreadTimings) UpdateRunningTask(_ scheduler.SpawnTime, _ scheduler.Location) {}

// SaveTaskTiming does nothing without the gpui_profiler build tag.
func (t *ThreadTimings) SaveTaskTiming(_ YieldTime) TaskTiming {
	return TaskTiming{}
}

// UpdateRunningAction does nothing without the gpui_profiler build tag.
func (s *ActionStatistics) UpdateRunningAction(_ string, _ scheduler.Instant) {}

// SaveActionTiming does nothing without the gpui_profiler build tag.
func (s *ActionStatistics) SaveActionTiming() {}

// TakeActionStats returns empty statistics without the gpui_profiler build tag.
func TakeActionStats() ActionStatistics {
	return DefaultActionStatistics()
}
