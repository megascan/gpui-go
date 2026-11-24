//go:build gpui_profiler

package gpui

import "github.com/megascan/gpui-go/scheduler"

const profilerFeatureEnabled = true

// GetAllTimings returns the task timings of every live thread.
func GetAllTimings(included TasksIncluded) []ThreadTaskTimings {
	return ThreadTaskTimingsCollect(upgradedThreadTimings(), included)
}

// GetCurrentThreadTimings returns thread's task timings.
func GetCurrentThreadTimings(thread ThreadID, included TasksIncluded) ThreadTaskTimings {
	return GetCurrentThreadTaskTimings(thread, included)
}

// TakeAllStats takes and resets the task statistics of every live thread.
func TakeAllStats(included TasksIncluded) []ThreadTaskStatistics {
	return ThreadTaskStatisticsCollectAndReset(upgradedThreadTimings(), included)
}

// UpdateRunningTask records the task that started polling.
func (t *ThreadTimings) UpdateRunningTask(spawned scheduler.SpawnTime, location scheduler.Location) {
	start := scheduler.InstantNow()
	t.Running = Some(ActiveTiming{Spawned: spawned, Location: location, Start: start})
}

// SaveTaskTiming records that the running task yielded at ended and returns its timing.
func (t *ThreadTimings) SaveTaskTiming(ended YieldTime) TaskTiming {
	running := t.Running.Take().Expect("this function is only ever called after register_task_start")
	timing := TaskTiming{
		Location: running.Location,
		Spawned:  running.Spawned,
		Start:    running.Start,
		End:      ended,
	}
	t.Stats.addYieldTiming(timing)
	t.Stats.addRuntime(timing)
	if TraceEnabled() {
		if len(t.Timings) >= maxTaskTimings {
			t.Timings = t.Timings[1:]
		}
		t.Timings = append(t.Timings, timing)
		t.TotalPushed++
	}
	return timing
}

// UpdateRunningAction records the action that started running.
func (s *ActionStatistics) UpdateRunningAction(action string, started scheduler.Instant) {
	s.running = Some(runningAction{name: action, started: started})
}

// SaveActionTiming records that the running action finished.
func (s *ActionStatistics) SaveActionTiming() {
	now := scheduler.InstantNow()
	running, ok := s.running.Take().Get()
	if !ok {
		return
	}
	timing := ActionTiming{Name: running.name, Start: running.started, End: now}
	runtime := now.Sub(running.started)
	if runtime >= s.runtimeToBeat {
		if len(s.longestRuntimes) == actionStatisticsCapacity {
			toReplace := 0
			for i, action := range s.longestRuntimes {
				if !(runtime >= action.Runtime()) {
					toReplace = i
					break
				}
			}
			s.longestRuntimes[toReplace] = timing
		} else {
			s.longestRuntimes = append(s.longestRuntimes, timing)
		}
		s.runtimeToBeat = s.longestRuntimes[0].Runtime()
		for _, action := range s.longestRuntimes[1:] {
			s.runtimeToBeat = min(s.runtimeToBeat, action.Runtime())
		}
	}
}

// TakeActionStats takes the global action statistics.
func TakeActionStats() ActionStatistics {
	actionStatistics.mu.Lock()
	defer actionStatistics.mu.Unlock()
	return actionStatistics.stats.Take()
}
