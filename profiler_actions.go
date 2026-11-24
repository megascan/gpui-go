package gpui

import (
	"cmp"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

const actionStatisticsCapacity = 5

type runningAction struct {
	name    string
	started scheduler.Instant
}

// ActionStatistics tracks the actions that ran the longest.
type ActionStatistics struct {
	runtimeToBeat   time.Duration
	longestRuntimes []ActionTiming
	running         Option[runningAction]
}

// GoString formats the statistics as upstream's Debug impl.
func (s ActionStatistics) GoString() string {
	running := "None"
	if r, ok := s.running.Get(); ok {
		running = fmt.Sprintf("Some((%q, %v))", r.name, r.started.Elapsed())
	}
	return fmt.Sprintf("ActionStatistics { runtime_to_beat: %v, longest_runtimes: %v, running: %s }",
		s.runtimeToBeat, s.longestRuntimes, running)
}

// String formats the statistics as upstream's Display impl.
func (s ActionStatistics) String() string {
	var f strings.Builder
	f.WriteString("Actions that blocked the longest\n")
	actions := slices.Collect(s.LongestRuntimes(true))
	slices.SortStableFunc(actions, func(a, b ActionTiming) int {
		return cmp.Compare(a.Runtime(), b.Runtime())
	})
	slices.Reverse(actions)
	for _, action := range actions {
		fmt.Fprintf(&f, "%-20s - %s\n", action.Runtime().String(), action.Name)
	}
	return f.String()
}

// DefaultActionStatistics returns upstream's Default.
func DefaultActionStatistics() ActionStatistics {
	return ActionStatistics{
		runtimeToBeat:   100 * time.Microsecond,
		longestRuntimes: make([]ActionTiming, 0, actionStatisticsCapacity),
		running:         None[runningAction](),
	}
}

// Take returns the statistics and resets them, keeping the running action.
func (s *ActionStatistics) Take() ActionStatistics {
	taken := *s
	*s = DefaultActionStatistics()
	s.running = taken.running
	return taken
}

// IsEmpty reports whether no action timing was recorded.
func (s *ActionStatistics) IsEmpty() bool {
	return len(s.longestRuntimes) == 0
}

// LongestRuntimes yields the longest action timings, followed by the running action when includeRunning.
func (s *ActionStatistics) LongestRuntimes(includeRunning bool) iter.Seq[ActionTiming] {
	longest := slices.Clone(s.longestRuntimes)
	running := s.running
	return func(yield func(ActionTiming) bool) {
		for _, timing := range longest {
			if !yield(timing) {
				return
			}
		}
		if r, ok := running.Get(); ok && includeRunning {
			yield(ActionTiming{Name: r.name, Start: r.started, End: scheduler.InstantNow()})
		}
	}
}

// ActionTiming is one run of an action handler.
type ActionTiming struct {
	Name  string
	Start scheduler.Instant
	End   scheduler.Instant
}

// String formats the timing as upstream's Debug impl.
func (t ActionTiming) String() string {
	return fmt.Sprintf("ActionTiming { name: %q, runtime: %v }", t.Name, t.Runtime())
}

// Duration returns how long the action ran.
func (t ActionTiming) Duration() time.Duration {
	return t.End.SaturatingDurationSince(t.Start)
}

// Runtime returns how long the action ran.
func (t ActionTiming) Runtime() time.Duration {
	return t.End.Sub(t.Start)
}

var actionStatistics = struct {
	mu    sync.Mutex
	stats ActionStatistics
}{stats: DefaultActionStatistics()}

func updateRunningAction(action Action, cx *App) string {
	now := scheduler.InstantNow()
	name, ok := cx.actionRegistry().tryResolveAction(reflect.TypeOf(action))
	if !ok {
		name = "un-named"
	}
	actionStatistics.mu.Lock()
	actionStatistics.stats.UpdateRunningAction(name, now)
	actionStatistics.mu.Unlock()
	return name
}

func saveActionTiming() {
	actionStatistics.mu.Lock()
	actionStatistics.stats.SaveActionTiming()
	actionStatistics.mu.Unlock()
}
