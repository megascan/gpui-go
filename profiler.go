package gpui

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"math/bits"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
	"weak"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// ThreadID identifies a dispatcher thread whose task timings are tracked (std::thread::ThreadId).
type ThreadID uint64

var (
	nextThreadID atomic.Uint64
	threadNames  sync.Map
)

// NewThreadID allocates the identity of a dispatcher thread, with an optional name.
func NewThreadID(name Option[string]) ThreadID {
	id := ThreadID(nextThreadID.Add(1))
	if n, ok := name.Get(); ok {
		threadNames.Store(id, n)
	}
	return id
}

func threadName(id ThreadID) Option[string] {
	if n, ok := threadNames.Load(id); ok {
		return Some(n.(string))
	}
	return None[string]()
}

func hashThreadID(id ThreadID) uint64 {
	h := fnv.New64a()
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(id))
	h.Write(b[:])
	return h.Sum64()
}

// YieldTime is the moment a task yielded.
type YieldTime struct {
	scheduler.Instant
}

// TaskTiming is one poll of a task.
type TaskTiming struct {
	Location scheduler.Location
	Spawned  scheduler.SpawnTime
	Start    scheduler.Instant
	End      YieldTime
}

// String formats the timing as upstream's Debug impl.
func (t TaskTiming) String() string {
	return fmt.Sprintf("TaskTiming { location: %v, since_spawned: %v, last_poll_duration: %v, total_runtime: %v }",
		t.Location, t.Spawned.Instant.Elapsed(), t.PollDuration(), t.sinceSpawn())
}

// ActiveTiming is the task currently being polled.
type ActiveTiming struct {
	Location scheduler.Location
	Spawned  scheduler.SpawnTime
	Start    scheduler.Instant
}

// TaskTimingPlaceholder returns a task timing with a duration of zero. Any task will replace this in history.
func TaskTimingPlaceholder() TaskTiming {
	now := scheduler.InstantNow()
	return TaskTiming{
		Location: scheduler.CallerLocation(1),
		Spawned:  scheduler.SpawnTime{Instant: now},
		Start:    now,
		End:      YieldTime{now},
	}
}

// PollDuration returns how long the poll ran.
func (t TaskTiming) PollDuration() time.Duration {
	return t.End.Instant.Sub(t.Start)
}

func (t TaskTiming) sinceSpawn() time.Duration {
	return t.End.Instant.Sub(t.Spawned.Instant)
}

// ThreadTaskTimings are the timings recorded on one thread.
type ThreadTaskTimings struct {
	ThreadName  Option[string]
	ThreadID    ThreadID
	Timings     []TaskTiming
	Stats       TaskStatistics
	TotalPushed uint64
}

// UpgradedThreadTimings is a live per-thread timing handle.
type UpgradedThreadTimings struct {
	ThreadID ThreadID
	Timings  *GuardedTaskTimings
}

// ThreadTaskTimingsCollect converts upgraded per-thread timings into their structured format.
func ThreadTaskTimingsCollect(timings []UpgradedThreadTimings, included TasksIncluded) []ThreadTaskTimings {
	out := make([]ThreadTaskTimings, 0, len(timings))
	for _, entry := range timings {
		t := entry.Timings.Lock()
		threadName := t.ThreadName
		totalPushed := t.TotalPushed
		vec := make([]TaskTiming, 0, len(t.Timings)+1)
		vec = append(vec, t.Timings...)
		if running, ok := t.Running.Get(); included == TasksIncludedCompletedAndRunning && ok {
			vec = append(vec, TaskTiming{
				Location: running.Location,
				Spawned:  running.Spawned,
				Start:    running.Start,
				End:      YieldTime{scheduler.InstantNow()},
			})
		}
		out = append(out, ThreadTaskTimings{
			ThreadName:  threadName,
			ThreadID:    entry.ThreadID,
			Timings:     vec,
			Stats:       t.Stats,
			TotalPushed: totalPushed,
		})
		entry.Timings.Unlock()
	}
	return out
}

// ThreadTaskStatistics are the statistics of one thread.
type ThreadTaskStatistics struct {
	ThreadName Option[string]
	ThreadID   ThreadID
	Stats      TaskStatistics
}

// ThreadTaskStatisticsCollectAndReset takes each thread's statistics, resetting them.
func ThreadTaskStatisticsCollectAndReset(timings []UpgradedThreadTimings, includeRunning TasksIncluded) []ThreadTaskStatistics {
	out := make([]ThreadTaskStatistics, 0, len(timings))
	for _, entry := range timings {
		t := entry.Timings.Lock()
		threadName := t.ThreadName
		stats := t.Stats
		t.Stats = DefaultTaskStatistics()
		if running, ok := t.Running.Get(); includeRunning == TasksIncludedCompletedAndRunning && ok {
			end := YieldTime{scheduler.InstantNow()}
			timing := TaskTiming{
				Location: running.Location,
				Spawned:  running.Spawned,
				Start:    running.Start,
				End:      end,
			}
			stats.addRuntime(timing)
			stats.addYieldTiming(timing)
		}
		entry.Timings.Unlock()
		out = append(out, ThreadTaskStatistics{
			ThreadName: threadName,
			ThreadID:   entry.ThreadID,
			Stats:      stats,
		})
	}
	return out
}

// SerializedLocation is the serializable variant of a source location.
type SerializedLocation struct {
	File   SharedString `json:"file"`
	Line   uint32       `json:"line"`
	Column uint32       `json:"column"`
}

func serializedLocationFrom(value scheduler.Location) SerializedLocation {
	return SerializedLocation{
		File: SharedString(value.File()),
		Line: uint32(value.Line()),
	}
}

// SerializedTaskTiming is the serializable variant of TaskTiming.
type SerializedTaskTiming struct {
	Location SerializedLocation `json:"location"`
	Start    uint64             `json:"start"`
	Duration uint64             `json:"duration"`
}

// SerializedTaskTimingConvert converts timings into their serializable format; anchor should be
// earlier than all timings.
func SerializedTaskTimingConvert(anchor scheduler.Instant, timings []TaskTiming) []SerializedTaskTiming {
	serialized := make([]SerializedTaskTiming, 0, len(timings))
	for _, timing := range timings {
		serialized = append(serialized, SerializedTaskTimingFrom(anchor, timing))
	}
	return serialized
}

// SerializedTaskTimingFrom converts one timing; anchor should be earlier than all timings.
func SerializedTaskTimingFrom(anchor scheduler.Instant, timing TaskTiming) SerializedTaskTiming {
	start := uint64(timing.Start.SaturatingDurationSince(anchor))
	duration := uint64(timing.End.Instant.SaturatingDurationSince(timing.Start))
	return SerializedTaskTiming{
		Location: serializedLocationFrom(timing.Location),
		Start:    start,
		Duration: duration,
	}
}

// SerializedThreadTaskTimings is the serializable variant of ThreadTaskTimings.
type SerializedThreadTaskTimings struct {
	ThreadName Option[string]         `json:"thread_name"`
	ThreadID   uint64                 `json:"thread_id"`
	Timings    []SerializedTaskTiming `json:"timings"`
}

// SerializedThreadTaskTimingsConvert converts timings into their serializable format; anchor
// should be earlier than all timings.
func SerializedThreadTaskTimingsConvert(anchor scheduler.Instant, timings ThreadTaskTimings) SerializedThreadTaskTimings {
	return SerializedThreadTaskTimings{
		ThreadName: timings.ThreadName,
		ThreadID:   hashThreadID(timings.ThreadID),
		Timings:    SerializedTaskTimingConvert(anchor, timings.Timings),
	}
}

// ThreadTimingsDelta are the timings of one thread not seen before.
type ThreadTimingsDelta struct {
	ThreadID   uint64
	ThreadName Option[string]
	NewTimings []SerializedTaskTiming
}

// ProfilingCollector tracks which timing events have already been seen so that callers can
// request only unseen events.
type ProfilingCollector struct {
	startupTime scheduler.Instant
	cursors     map[ThreadID]uint64
}

// NewProfilingCollector creates a collector anchored at startupTime.
func NewProfilingCollector(startupTime scheduler.Instant) *ProfilingCollector {
	return &ProfilingCollector{startupTime: startupTime, cursors: map[ThreadID]uint64{}}
}

// StartupTime returns the collector's anchor.
func (c *ProfilingCollector) StartupTime() scheduler.Instant {
	return c.startupTime
}

// CollectUnseen returns the timings recorded since the previous call.
func (c *ProfilingCollector) CollectUnseen(allTimings []ThreadTaskTimings) []ThreadTimingsDelta {
	deltas := make([]ThreadTimingsDelta, 0, len(allTimings))
	for _, thread := range allTimings {
		hashedID := hashThreadID(thread.ThreadID)
		prevCursor := c.cursors[thread.ThreadID]
		bufferLen := uint64(len(thread.Timings))
		bufferStart := saturatingSubU64(thread.TotalPushed, bufferLen)
		var slice []TaskTiming
		if prevCursor < bufferStart {
			slice = thread.Timings
		} else {
			skip := min(prevCursor-bufferStart, uint64(len(thread.Timings)))
			slice = thread.Timings[skip:]
		}
		c.cursors[thread.ThreadID] = thread.TotalPushed
		if len(slice) == 0 {
			continue
		}
		deltas = append(deltas, ThreadTimingsDelta{
			ThreadID:   hashedID,
			ThreadName: thread.ThreadName,
			NewTimings: SerializedTaskTimingConvert(c.startupTime, slice),
		})
	}
	return deltas
}

// Reset forgets every cursor.
func (c *ProfilingCollector) Reset() {
	clear(c.cursors)
}

func saturatingSubU64(a, b uint64) uint64 {
	if b > a {
		return 0
	}
	return a - b
}

const maxTaskTimings = (16 * 1024 * 1024) / int(unsafe.Sizeof(TaskTiming{}))

// GuardedTaskTimings is a ThreadTimings behind a lock.
type GuardedTaskTimings struct {
	mu      sync.Mutex
	timings ThreadTimings
}

// Lock acquires the lock and returns the guarded timings.
func (g *GuardedTaskTimings) Lock() *ThreadTimings {
	g.mu.Lock()
	return &g.timings
}

// Unlock releases the lock.
func (g *GuardedTaskTimings) Unlock() {
	g.mu.Unlock()
}

// GlobalThreadTimings is a registry entry for one thread's timings.
type GlobalThreadTimings struct {
	ThreadID ThreadID
	Timings  weak.Pointer[GuardedTaskTimings]
}

// TaskStatistics tracks the tasks that blocked and ran the longest.
type TaskStatistics struct {
	PollTimeToBeat   time.Duration
	RuntimeToBeat    time.Duration
	LongestPollTimes [5]TaskTiming
	LongestRuntimes  [5]TaskTiming
}

// String formats the statistics as upstream's Display impl.
func (s TaskStatistics) String() string {
	var f strings.Builder
	f.WriteString("Tasks that blocked the longest\n")
	for _, timing := range s.LongestPollTimes {
		fmt.Fprintf(&f, "%-20s - %s:%d\n", timing.PollDuration().String(), timing.Location.File(), 0)
	}
	f.WriteString("Tasks that ran the longest\n")
	for _, timing := range s.LongestRuntimes {
		fmt.Fprintf(&f, "%-20s - %s:%d\n", timing.sinceSpawn().String(), timing.Location.File(), 0)
	}
	return f.String()
}

// DefaultTaskStatistics returns upstream's Default.
func DefaultTaskStatistics() TaskStatistics {
	placeholder := TaskTimingPlaceholder()
	s := TaskStatistics{
		PollTimeToBeat: 100 * time.Microsecond,
		RuntimeToBeat:  100 * time.Microsecond,
	}
	for i := range 5 {
		s.LongestPollTimes[i] = placeholder
		s.LongestRuntimes[i] = placeholder
	}
	return s
}

func positionMinBySinceSpawn(timings *[5]TaskTiming) int {
	index := 0
	for i := 1; i < len(timings); i++ {
		if timings[i].sinceSpawn() < timings[index].sinceSpawn() {
			index = i
		}
	}
	return index
}

func minSinceSpawn(timings *[5]TaskTiming) time.Duration {
	return timings[positionMinBySinceSpawn(timings)].sinceSpawn()
}

func (s *TaskStatistics) addYieldTiming(task TaskTiming) {
	yieldedAfter := task.PollDuration()
	if yieldedAfter >= s.PollTimeToBeat {
		toReplace := positionMinBySinceSpawn(&s.LongestPollTimes)
		s.LongestPollTimes[toReplace] = task
		s.PollTimeToBeat = minSinceSpawn(&s.LongestPollTimes)
	}
}

func (s *TaskStatistics) addRuntime(task TaskTiming) {
	runtime := task.sinceSpawn()
	if runtime >= s.RuntimeToBeat {
		toReplace := positionMinBySinceSpawn(&s.LongestRuntimes)
		s.LongestRuntimes[toReplace] = task
		s.RuntimeToBeat = minSinceSpawn(&s.LongestRuntimes)
	}
}

var globalThreadTimings struct {
	mu      sync.Mutex
	entries []GlobalThreadTimings
}

func upgradedThreadTimings() []UpgradedThreadTimings {
	globalThreadTimings.mu.Lock()
	defer globalThreadTimings.mu.Unlock()
	out := make([]UpgradedThreadTimings, 0, len(globalThreadTimings.entries))
	for _, t := range globalThreadTimings.entries {
		if timings := t.Timings.Value(); timings != nil {
			out = append(out, UpgradedThreadTimings{ThreadID: t.ThreadID, Timings: timings})
		}
	}
	return out
}

var threadTimingsByThread sync.Map

func threadTimingsFor(thread ThreadID) *GuardedTaskTimings {
	if timings, ok := threadTimingsByThread.Load(thread); ok {
		return timings.(*GuardedTaskTimings)
	}
	timings := &GuardedTaskTimings{timings: NewThreadTimings(threadName(thread), thread)}
	actual, loaded := threadTimingsByThread.LoadOrStore(thread, timings)
	if loaded {
		return actual.(*GuardedTaskTimings)
	}
	globalThreadTimings.mu.Lock()
	globalThreadTimings.entries = append(globalThreadTimings.entries, GlobalThreadTimings{
		ThreadID: thread,
		Timings:  weak.Make(timings),
	})
	globalThreadTimings.mu.Unlock()
	return timings
}

// ReleaseThreadTimings drops thread's timings at thread exit, removing them from the registry.
func ReleaseThreadTimings(thread ThreadID) {
	timings, ok := threadTimingsByThread.LoadAndDelete(thread)
	threadNames.Delete(thread)
	if !ok {
		return
	}
	timings.(*GuardedTaskTimings).timings.Release()
}

// ThreadTimings are the task timings of one thread.
type ThreadTimings struct {
	ThreadName  Option[string]
	ThreadID    ThreadID
	Timings     []TaskTiming
	Running     Option[ActiveTiming]
	Stats       TaskStatistics
	TotalPushed uint64
}

// NewThreadTimings creates empty timings for a thread.
func NewThreadTimings(threadName Option[string], threadID ThreadID) ThreadTimings {
	return ThreadTimings{
		ThreadName:  threadName,
		ThreadID:    threadID,
		Stats:       DefaultTaskStatistics(),
		TotalPushed: 0,
		Running:     None[ActiveTiming](),
	}
}

// GetThreadTaskTimings returns the thread's timings, including the running task when asked.
func (t *ThreadTimings) GetThreadTaskTimings(includes TasksIncluded) ThreadTaskTimings {
	timings := slices.Clone(t.Timings)
	if running, ok := t.Running.Get(); ok && includes == TasksIncludedCompletedAndRunning {
		timings = append(timings, TaskTiming{
			Spawned:  running.Spawned,
			Location: running.Location,
			Start:    running.Start,
			End:      YieldTime{scheduler.InstantNow()},
		})
	}
	if timings == nil {
		timings = []TaskTiming{}
	}
	return ThreadTaskTimings{
		ThreadName:  t.ThreadName,
		ThreadID:    t.ThreadID,
		Timings:     timings,
		Stats:       t.Stats,
		TotalPushed: t.TotalPushed,
	}
}

// Release removes the timings from the global registry, as upstream's Drop.
func (t *ThreadTimings) Release() {
	globalThreadTimings.mu.Lock()
	defer globalThreadTimings.mu.Unlock()
	entries := globalThreadTimings.entries
	for index, e := range entries {
		if e.ThreadID == t.ThreadID {
			last := len(entries) - 1
			entries[index] = entries[last]
			entries[last] = GlobalThreadTimings{}
			globalThreadTimings.entries = entries[:last]
			return
		}
	}
}

// UpdateRunningTask records that thread started polling a task.
func UpdateRunningTask(thread ThreadID, spawned scheduler.SpawnTime, location scheduler.Location) {
	if profilerFeatureEnabled {
		beginForegroundTurn()
	}
	timings := threadTimingsFor(thread)
	timings.Lock().UpdateRunningTask(spawned, location)
	timings.Unlock()
}

// SaveTaskTiming records that thread's running task yielded.
func SaveTaskTiming(thread ThreadID) {
	yieldedAt := YieldTime{scheduler.InstantNow()}
	timings := threadTimingsFor(thread)
	timing := timings.Lock().SaveTaskTiming(yieldedAt)
	timings.Unlock()
	if profilerFeatureEnabled {
		recordTaskPoll(timing)
	}
}

// GetCurrentThreadTaskTimings returns thread's timings.
func GetCurrentThreadTaskTimings(thread ThreadID, includeRunning TasksIncluded) ThreadTaskTimings {
	timings := threadTimingsFor(thread)
	defer timings.Unlock()
	return timings.Lock().GetThreadTaskTimings(includeRunning)
}

const (
	traceSettingEnabled uint64 = 1 << 63
	traceScopeCountMask        = traceSettingEnabled - 1
)

var traceState atomic.Uint64

// SetTraceEnabled enables or disables profiler trace collection at runtime. When transitioning
// from enabled to disabled the per-thread task buffers and the frame-event buffer are cleared.
// Active trace scopes keep collection enabled until the last scope ends. Calls with the current
// setting are a no-op and return false.
func SetTraceEnabled(enabled bool) bool {
	state := traceState.Load()
	for {
		wasEnabled := state&traceSettingEnabled != 0
		if wasEnabled == enabled {
			return false
		}
		var nextState uint64
		if enabled {
			nextState = state | traceSettingEnabled
		} else {
			nextState = state & traceScopeCountMask
		}
		if traceState.CompareAndSwap(state, nextState) {
			if nextState == 0 {
				clearTraceBuffers()
			}
			return true
		}
		state = traceState.Load()
	}
}

type traceGuard struct {
	ended *atomic.Bool
}

func traceStateTryUpdate(f func(state uint64) (uint64, bool)) (uint64, bool) {
	state := traceState.Load()
	for {
		next, ok := f(state)
		if !ok {
			return state, false
		}
		if traceState.CompareAndSwap(state, next) {
			return state, true
		}
		state = traceState.Load()
	}
}

func traceScope() traceGuard {
	_, incremented := traceStateTryUpdate(func(state uint64) (uint64, bool) {
		return state + 1, state&traceScopeCountMask < traceScopeCountMask
	})
	if !incremented {
		panic("too many active profiler trace scopes")
	}
	return traceGuard{ended: new(atomic.Bool)}
}

func (g traceGuard) End() {
	if g.ended.Swap(true) {
		return
	}
	previous, ok := traceStateTryUpdate(func(state uint64) (uint64, bool) {
		return state - 1, state&traceScopeCountMask > 0
	})
	switch {
	case ok && previous == 1:
		clearTraceBuffers()
	case !ok && gpuiutil.DebugAssertions:
		panic("profiler trace scope count underflowed")
	}
}

// TraceEnabled returns whether profiler trace collection is enabled.
func TraceEnabled() bool {
	return traceState.Load() != 0
}

func clearTraceBuffers() {
	for _, entry := range upgradedThreadTimings() {
		t := entry.Timings.Lock()
		t.Timings = nil
		t.TotalPushed = 0
		entry.Timings.Unlock()
	}
	if profilerFeatureEnabled {
		frameTimings.mu.Lock()
		frameTimings.timings = nil
		frameTimings.totalPushed = 0
		frameTimings.mu.Unlock()
	}
}

// FrameTiming is the timing of a single drawn window frame.
type FrameTiming struct {
	WindowID        WindowID
	DirtyAt         Option[scheduler.Instant]
	Invalidations   uint64
	DrawStart       scheduler.Instant
	DrawEnd         scheduler.Instant
	RefreshInterval Option[time.Duration]
	SignalAt        Option[scheduler.Instant]
	Phases          DrawPhases
}

// DrawPhases is how long each phase of a window draw took.
type DrawPhases struct {
	RequestLayout time.Duration
	Prepaint      time.Duration
	Paint         time.Duration
}

// ViewPhase is a phase of drawing a view.
type ViewPhase uint8

const (
	// ViewPhaseRequestLayout renders the view and requests its layout.
	ViewPhaseRequestLayout ViewPhase = iota
	// ViewPhasePrepaint prepaints the view's elements.
	ViewPhasePrepaint
	// ViewPhasePaint paints the view's elements.
	ViewPhasePaint
)

// AsStr returns the phase's name, e.g. "prepaint".
func (p ViewPhase) AsStr() string {
	switch p {
	case ViewPhaseRequestLayout:
		return "request_layout"
	case ViewPhasePrepaint:
		return "prepaint"
	default:
		return "paint"
	}
}

// ViewTimingFloor is the least own work of a view phase that is recorded.
const ViewTimingFloor = 500 * time.Microsecond

// ViewTiming is one phase of drawing one view.
type ViewTiming struct {
	Name         string
	Phase        ViewPhase
	Start        scheduler.Instant
	End          scheduler.Instant
	SelfDuration time.Duration
}

// FrameBudgetFloor is the least work a frame can take before it counts as late: one refresh at 120 Hz.
const FrameBudgetFloor = 8_333_333 * time.Nanosecond

// UnknownRefreshFrameBudget is the frame budget when the refresh interval is unknown: one refresh at 60 Hz.
const UnknownRefreshFrameBudget = 16_666_667 * time.Nanosecond

// WorkStart returns when the frame's work began: the later of its signal and its first invalidation.
func (f FrameTiming) WorkStart() (scheduler.Instant, bool) {
	signalAt, ok := f.SignalAt.Get()
	if !ok {
		return scheduler.Instant{}, false
	}
	if dirtyAt, ok := f.DirtyAt.Get(); ok {
		if dirtyAt.After(signalAt) {
			return dirtyAt, true
		}
	}
	return signalAt, true
}

// Work returns the main-thread time from WorkStart until the frame was submitted at submittedAt.
func (f FrameTiming) Work(submittedAt scheduler.Instant) (time.Duration, bool) {
	start, ok := f.WorkStart()
	if !ok {
		return 0, false
	}
	return submittedAt.SaturatingDurationSince(start), true
}

// IsLate reports whether the frame's work exceeded its Budget, when its work is known.
func (f FrameTiming) IsLate(submittedAt scheduler.Instant) (bool, bool) {
	work, ok := f.Work(submittedAt)
	if !ok {
		return false, false
	}
	return work > f.Budget(), true
}

// Budget returns the work a frame can take before it counts as late.
func (f FrameTiming) Budget() time.Duration {
	if interval, ok := f.RefreshInterval.Get(); ok {
		return max(interval, FrameBudgetFloor)
	}
	return UnknownRefreshFrameBudget
}

// MissedRefreshes returns how many refreshes the frame missed when submitted at submittedAt.
func (f FrameTiming) MissedRefreshes(submittedAt scheduler.Instant) (uint64, bool) {
	refreshInterval, ok := f.RefreshInterval.Get()
	if !ok {
		return 0, false
	}
	work, ok := f.Work(submittedAt)
	if !ok {
		return 0, false
	}
	w, r := uint64(work), uint64(refreshInterval)
	refreshes := w / r
	if w%r != 0 {
		refreshes++
	}
	refreshes = max(refreshes, 1)
	return refreshes - 1, true
}

// DrawDuration returns the time spent inside Window::draw.
func (f FrameTiming) DrawDuration() time.Duration {
	return f.DrawEnd.SaturatingDurationSince(f.DrawStart)
}

// DirtyToDrawDuration returns the time from the first invalidation to the end of the draw.
func (f FrameTiming) DirtyToDrawDuration() (time.Duration, bool) {
	dirtyAt, ok := f.DirtyAt.Get()
	if !ok {
		return 0, false
	}
	return f.DrawEnd.SaturatingDurationSince(dirtyAt), true
}

// PresentTiming is the work spent submitting a window frame to the platform.
type PresentTiming struct {
	WindowID          WindowID
	PresentStart      scheduler.Instant
	PresentEnd        scheduler.Instant
	AnimationInterval Option[time.Duration]
}

// PresentDuration returns the time spent submitting the frame.
func (p PresentTiming) PresentDuration() time.Duration {
	return p.PresentEnd.SaturatingDurationSince(p.PresentStart)
}

// FrameEvent is a frame event observed by the profiler: FrameEventDraw or FrameEventPresent.
type FrameEvent interface {
	isFrameEvent()
}

// FrameEventDraw is a drawn window frame.
type FrameEventDraw struct {
	FrameTiming
}

// FrameEventPresent is a presented newly drawn window frame.
type FrameEventPresent struct {
	PresentTiming
}

func (FrameEventDraw) isFrameEvent()    {}
func (FrameEventPresent) isFrameEvent() {}

// FrameDurationSnapshot is a snapshot of a window's frame-duration histograms.
type FrameDurationSnapshot struct {
	DirtyToPresentHistogram  *Histogram
	DrawDurationHistogram    *Histogram
	PresentIntervalHistogram *Histogram
	ByRefreshInterval        []RefreshIntervalFrames
}

// FrameDurationBucketsMs are the upper bounds, in milliseconds, of the duration buckets.
var FrameDurationBucketsMs = [9]uint64{4, 8, 16, 33, 50, 100, 250, 500, 1000}

// FrameWorkBucketsPercent are the upper bounds, in percent of the refresh interval, of the work buckets.
var FrameWorkBucketsPercent = [7]uint64{25, 50, 75, 100, 150, 200, 400}

// HistogramBucketCounts counts the values up to and including each bound, plus the values above the last.
func HistogramBucketCounts(histogram *Histogram, upperBounds []uint64) []uint64 {
	var counted uint64
	counts := make([]uint64, 0, len(upperBounds)+1)
	for _, bound := range upperBounds {
		cumulative := histogram.CountBetween(0, bound)
		counts = append(counts, saturatingSubU64(cumulative, counted))
		counted = max(counted, cumulative)
	}
	return append(counts, saturatingSubU64(histogram.Len(), counted))
}

// RefreshIntervalFrames are the statistics of frames drawn while the display had one refresh interval.
type RefreshIntervalFrames struct {
	RefreshInterval   Option[time.Duration]
	FramesDrawn       uint64
	DrawDuration      *Histogram
	FramesOnTime      uint64
	FramesLate        uint64
	MissedRefreshes   uint64
	FramesUnmeasured  uint64
	Work              *Histogram
	InputFramesOnTime uint64
	InputFramesLate   uint64
	InputWork         *Histogram
}

func newRefreshIntervalFrames(refreshInterval Option[time.Duration], emptyHistogram *Histogram) RefreshIntervalFrames {
	return RefreshIntervalFrames{
		RefreshInterval: refreshInterval,
		DrawDuration:    emptyHistogram.Clone(),
		Work:            emptyHistogram.Clone(),
		InputWork:       emptyHistogram.Clone(),
	}
}

func (r RefreshIntervalFrames) clone() RefreshIntervalFrames {
	r.DrawDuration = r.DrawDuration.Clone()
	r.Work = r.Work.Clone()
	r.InputWork = r.InputWork.Clone()
	return r
}

// Since returns the frames recorded since earlier, a previous snapshot of the same statistics.
func (r RefreshIntervalFrames) Since(earlier RefreshIntervalFrames) RefreshIntervalFrames {
	subtract := func(current, earlier *Histogram) *Histogram {
		delta := current.Clone()
		if err := delta.Subtract(earlier); err != nil {
			slog.Error(fmt.Sprintf("failed to subtract frame histograms: %v", err))
			return current.Clone()
		}
		return delta
	}
	return RefreshIntervalFrames{
		RefreshInterval:   r.RefreshInterval,
		FramesDrawn:       saturatingSubU64(r.FramesDrawn, earlier.FramesDrawn),
		DrawDuration:      subtract(r.DrawDuration, earlier.DrawDuration),
		FramesOnTime:      saturatingSubU64(r.FramesOnTime, earlier.FramesOnTime),
		FramesLate:        saturatingSubU64(r.FramesLate, earlier.FramesLate),
		MissedRefreshes:   saturatingSubU64(r.MissedRefreshes, earlier.MissedRefreshes),
		FramesUnmeasured:  saturatingSubU64(r.FramesUnmeasured, earlier.FramesUnmeasured),
		Work:              subtract(r.Work, earlier.Work),
		InputFramesOnTime: saturatingSubU64(r.InputFramesOnTime, earlier.InputFramesOnTime),
		InputFramesLate:   saturatingSubU64(r.InputFramesLate, earlier.InputFramesLate),
		InputWork:         subtract(r.InputWork, earlier.InputWork),
	}
}

func (r *RefreshIntervalFrames) recordPresentedFrame(frame *FrameTiming, presentation *PresentTiming, respondedToInput bool) {
	submittedAt := presentation.PresentStart
	work, okWork := frame.Work(submittedAt)
	isLate, okLate := frame.IsLate(submittedAt)
	if !okWork || !okLate {
		r.FramesUnmeasured++
		return
	}
	recordDuration(r.Work, work)
	if respondedToInput {
		recordDuration(r.InputWork, work)
	}
	missed, _ := frame.MissedRefreshes(submittedAt)
	r.MissedRefreshes += missed
	input := uint64(0)
	if respondedToInput {
		input = 1
	}
	if isLate {
		r.FramesLate++
		r.InputFramesLate += input
	} else {
		r.FramesOnTime++
		r.InputFramesOnTime += input
	}
}

func durationNanos(d time.Duration) uint64 {
	if d < 0 {
		return 0
	}
	return uint64(d)
}

func recordDuration(histogram *Histogram, duration time.Duration) {
	if err := histogram.Record(durationNanos(duration)); err != nil {
		slog.Error(fmt.Sprintf("failed to record frame duration: %v", err))
	}
}

// InputLatencySnapshot is a snapshot of a window's input-latency histograms.
type InputLatencySnapshot struct {
	LatencyHistogram        *Histogram
	EventsPerFrameHistogram *Histogram
	MidDrawEventsDropped    uint64
}

type windowActivity struct {
	draw      bool
	startedAt scheduler.Instant
	kind      string
}

type activeAction struct {
	name  string
	start scheduler.Instant
}

// WindowProfiler collects profiling information for one window.
type WindowProfiler struct {
	windowID                 WindowID
	activeActivities         []windowActivity
	activeActions            []activeAction
	dirtyToPresentHistogram  *Histogram
	drawDurationHistogram    *Histogram
	presentIntervalHistogram *Histogram
	firstInputAt             Option[scheduler.Instant]
	pendingInputCount        uint64
	inputLatencyHistogram    *Histogram
	eventsPerFrameHistogram  *Histogram
	midDrawEventsDropped     uint64
	lastPresentAt            Option[scheduler.Instant]
	animatingAtLastPresent   bool
	pendingFrame             Option[FrameTiming]
	refreshInterval          Option[time.Duration]
	frameSignalAt            Option[scheduler.Instant]
	byRefreshInterval        []RefreshIntervalFrames
	emptyHistogram           *Histogram
	drawPhases               DrawPhases
	nestedViewTime           []time.Duration
	closed                   bool
}

// NewWindowProfiler creates a profiler for a window.
func NewWindowProfiler(windowID WindowID) (*WindowProfiler, error) {
	newHistogram := func(what string) (*Histogram, error) {
		h, err := NewHistogram(3)
		if err != nil {
			return nil, fmt.Errorf("Failed to create %s histogram: %w", what, err)
		}
		return h, nil
	}
	p := &WindowProfiler{windowID: windowID}
	var err error
	if p.dirtyToPresentHistogram, err = newHistogram("dirty-to-present"); err != nil {
		return nil, err
	}
	if p.drawDurationHistogram, err = newHistogram("draw duration"); err != nil {
		return nil, err
	}
	if p.presentIntervalHistogram, err = newHistogram("present interval"); err != nil {
		return nil, err
	}
	if p.inputLatencyHistogram, err = newHistogram("input latency"); err != nil {
		return nil, err
	}
	if p.eventsPerFrameHistogram, err = newHistogram("events per frame"); err != nil {
		return nil, err
	}
	if p.emptyHistogram, err = newHistogram("frame"); err != nil {
		return nil, err
	}
	recordFramePending(windowID, scheduler.InstantNow())
	return p, nil
}

// BeginInput records the beginning of an input dispatch of the given kind.
func (p *WindowProfiler) BeginInput(kind string) {
	beginForegroundTurn()
	p.activeActivities = append(p.activeActivities, windowActivity{startedAt: scheduler.InstantNow(), kind: kind})
}

func (p *WindowProfiler) popActivity() (windowActivity, bool) {
	n := len(p.activeActivities)
	if n == 0 {
		return windowActivity{}, false
	}
	a := p.activeActivities[n-1]
	p.activeActivities = p.activeActivities[:n-1]
	return a, true
}

// EndInput records the end of an input dispatch.
func (p *WindowProfiler) EndInput(causedInvalidation bool) {
	activity, ok := p.popActivity()
	if !ok || activity.draw {
		if gpuiutil.DebugAssertions {
			panic("input activity must be the current window activity")
		}
		endForegroundTurn()
		return
	}
	startedAt, kind := activity.startedAt, activity.kind
	if !powerInterruptedSince(startedAt) {
		recordInput(InputTiming{
			Kind:               kind,
			Start:              startedAt,
			End:                scheduler.InstantNow(),
			CausedInvalidation: causedInvalidation,
		})
	}
	endForegroundTurn()

	if !causedInvalidation || !frameSampleIsValid(p.windowID, startedAt) {
		return
	}
	if at, ok := p.firstInputAt.Get(); ok && !frameSampleIsValid(p.windowID, at) {
		p.firstInputAt = None[scheduler.Instant]()
		p.pendingInputCount = 0
	}
	arrivedDuringDraw := slices.ContainsFunc(p.activeActivities, func(a windowActivity) bool { return a.draw })
	if arrivedDuringDraw {
		p.midDrawEventsDropped++
	} else {
		if p.firstInputAt.IsNone() {
			p.firstInputAt = Some(startedAt)
		}
		p.pendingInputCount++
	}
}

// BeginActionHandler records the beginning of an action handler.
func (p *WindowProfiler) BeginActionHandler(action Action, cx *App) {
	beginForegroundTurn()
	name := updateRunningAction(action, cx)
	p.activeActions = append(p.activeActions, activeAction{name: name, start: scheduler.InstantNow()})
}

// EndActionHandler records the end of the current action handler.
func (p *WindowProfiler) EndActionHandler() {
	saveActionTiming()
	n := len(p.activeActions)
	if n == 0 {
		if gpuiutil.DebugAssertions {
			panic("action handler must be begun before it ends")
		}
		endForegroundTurn()
		return
	}
	a := p.activeActions[n-1]
	p.activeActions = p.activeActions[:n-1]
	if !powerInterruptedSince(a.start) {
		recordAction(ActionTiming{Name: a.name, Start: a.start, End: scheduler.InstantNow()})
	}
	endForegroundTurn()
}

// RecordFrameSignal records when the frame signal for the next drawn frame fired.
func (p *WindowProfiler) RecordFrameSignal(signalAt Option[scheduler.Instant]) {
	p.frameSignalAt = signalAt
}

// SetRefreshInterval sets the refresh interval of the window's display.
func (p *WindowProfiler) SetRefreshInterval(refreshInterval Option[time.Duration]) {
	p.refreshInterval = refreshInterval
}

// BeginDraw records the beginning of a window draw.
func (p *WindowProfiler) BeginDraw() {
	beginForegroundTurn()
	startedAt := scheduler.InstantNow()
	recordFramePending(p.windowID, startedAt)
	p.activeActivities = append(p.activeActivities, windowActivity{draw: true, startedAt: startedAt})
}

// RecordDrawPhases records the phases of the window draw in progress.
func (p *WindowProfiler) RecordDrawPhases(phases DrawPhases) {
	p.drawPhases = phases
}

// BeginView records the start of one phase of drawing a view, returning its start time.
func (p *WindowProfiler) BeginView() scheduler.Instant {
	p.nestedViewTime = append(p.nestedViewTime, 0)
	return scheduler.InstantNow()
}

// EndView records the end of a view phase begun at start.
func (p *WindowProfiler) EndView(start scheduler.Instant, name string, phase ViewPhase) {
	end := scheduler.InstantNow()
	duration := end.SaturatingDurationSince(start)
	var nested time.Duration
	if n := len(p.nestedViewTime); n > 0 {
		nested = p.nestedViewTime[n-1]
		p.nestedViewTime = p.nestedViewTime[:n-1]
	}
	if n := len(p.nestedViewTime); n > 0 {
		p.nestedViewTime[n-1] += duration
	}
	selfDuration := max(duration-nested, 0)
	if selfDuration >= ViewTimingFloor {
		recordView(ViewTiming{Name: name, Phase: phase, Start: start, End: end, SelfDuration: selfDuration})
	}
}

// EndDraw records the end of a window draw and returns the draw duration.
func (p *WindowProfiler) EndDraw(dirtyAt Option[scheduler.Instant], invalidations uint64) time.Duration {
	activity, ok := p.popActivity()
	if !ok || !activity.draw {
		if gpuiutil.DebugAssertions {
			panic("draw activity must be the current window activity")
		}
		endForegroundTurn()
		return 0
	}
	drawStart := activity.startedAt
	drawEnd := scheduler.InstantNow()
	signalAt := p.frameSignalAt.Take()
	if at, ok := dirtyAt.Get(); ok && !frameSampleIsValid(p.windowID, at) {
		dirtyAt = None[scheduler.Instant]()
	}
	phases := p.drawPhases
	p.drawPhases = DrawPhases{}
	frameTiming := FrameTiming{
		WindowID:        p.windowID,
		DirtyAt:         dirtyAt,
		Invalidations:   invalidations,
		DrawStart:       drawStart,
		DrawEnd:         drawEnd,
		RefreshInterval: p.refreshInterval,
		SignalAt:        signalAt,
		Phases:          phases,
	}
	drawDuration := frameTiming.DrawDuration()
	if !powerInterruptedSince(drawStart) {
		p.recordDrawTiming(frameTiming)
	}
	endForegroundTurn()
	return drawDuration
}

// RecordPresent records that a frame was presented.
func (p *WindowProfiler) RecordPresent(presentStart, presentEnd scheduler.Instant, windowActive, nextFrameScheduled bool) {
	p.recordPresentAt(presentStart, presentEnd, windowActive, nextFrameScheduled)
}

// InputLatencySnapshot returns a snapshot of the current input-latency histograms.
func (p *WindowProfiler) InputLatencySnapshot() InputLatencySnapshot {
	return InputLatencySnapshot{
		LatencyHistogram:        p.inputLatencyHistogram.Clone(),
		EventsPerFrameHistogram: p.eventsPerFrameHistogram.Clone(),
		MidDrawEventsDropped:    p.midDrawEventsDropped,
	}
}

// FrameDurationSnapshot returns a snapshot of the current frame-duration histograms.
func (p *WindowProfiler) FrameDurationSnapshot() FrameDurationSnapshot {
	byRefreshInterval := make([]RefreshIntervalFrames, len(p.byRefreshInterval))
	for i, frames := range p.byRefreshInterval {
		byRefreshInterval[i] = frames.clone()
	}
	return FrameDurationSnapshot{
		DirtyToPresentHistogram:  p.dirtyToPresentHistogram.Clone(),
		DrawDurationHistogram:    p.drawDurationHistogram.Clone(),
		PresentIntervalHistogram: p.presentIntervalHistogram.Clone(),
		ByRefreshInterval:        byRefreshInterval,
	}
}

// Close records that the window closed, as upstream's Drop.
func (p *WindowProfiler) Close() {
	if p.closed {
		return
	}
	p.closed = true
	recordWindowClosed(p.windowID)
}

func (p *WindowProfiler) framesFor(refreshInterval Option[time.Duration]) *RefreshIntervalFrames {
	index := slices.IndexFunc(p.byRefreshInterval, func(frames RefreshIntervalFrames) bool {
		return frames.RefreshInterval == refreshInterval
	})
	if index < 0 {
		p.byRefreshInterval = append(p.byRefreshInterval, newRefreshIntervalFrames(refreshInterval, p.emptyHistogram))
		index = len(p.byRefreshInterval) - 1
	}
	return &p.byRefreshInterval[index]
}

func (p *WindowProfiler) recordPresentAt(presentStart, presentEnd scheduler.Instant, windowActive, nextFrameScheduled bool) {
	respondedToInput := false
	if firstInputAt, ok := p.firstInputAt.Take().Get(); ok && frameSampleIsValid(p.windowID, firstInputAt) {
		respondedToInput = true
		latencyNanos := durationNanos(presentEnd.SaturatingDurationSince(firstInputAt))
		_ = p.inputLatencyHistogram.Record(latencyNanos)
		if p.pendingInputCount > 0 {
			_ = p.eventsPerFrameHistogram.Record(p.pendingInputCount)
		}
	}
	p.pendingInputCount = 0

	frame := p.pendingFrame.Take()
	if f, ok := frame.Get(); ok {
		if !frameSampleIsValid(p.windowID, f.DrawStart) {
			frame = None[FrameTiming]()
		} else {
			if at, ok := f.DirtyAt.Get(); ok && !frameSampleIsValid(p.windowID, at) {
				f.DirtyAt = None[scheduler.Instant]()
			}
			frame = Some(f)
		}
	}
	animationInterval := None[time.Duration]()
	if frame.IsSome() && p.animatingAtLastPresent && windowActive {
		if lastPresentAt, ok := p.lastPresentAt.Get(); ok && frameSampleIsValid(p.windowID, lastPresentAt) {
			animationInterval = Some(presentEnd.SaturatingDurationSince(lastPresentAt))
		}
	}
	presentTiming := PresentTiming{
		WindowID:          p.windowID,
		PresentStart:      presentStart,
		PresentEnd:        presentEnd,
		AnimationInterval: animationInterval,
	}
	recordPresent(presentTiming, frame)

	f, ok := frame.Get()
	if !ok {
		return
	}
	p.framesFor(f.RefreshInterval).recordPresentedFrame(&f, &presentTiming, respondedToInput)

	if dirtyAt, ok := f.DirtyAt.Get(); ok {
		if err := p.dirtyToPresentHistogram.Record(durationNanos(presentEnd.SaturatingDurationSince(dirtyAt))); err != nil {
			slog.Error(fmt.Sprintf("failed to record dirty-to-present frame timing: %v", err))
		}
	}
	if interval, ok := animationInterval.Get(); ok {
		_ = p.presentIntervalHistogram.Record(durationNanos(interval))
	}
	RecordFrameEvent(FrameEventPresent{presentTiming})

	p.lastPresentAt = Some(presentEnd)
	p.animatingAtLastPresent = nextFrameScheduled && windowActive
}

func (p *WindowProfiler) recordDrawTiming(timing FrameTiming) {
	p.recordDrawDuration(timing.DrawDuration())
	frames := p.framesFor(timing.RefreshInterval)
	frames.FramesDrawn++
	recordDuration(frames.DrawDuration, timing.DrawDuration())
	p.pendingFrame = Some(timing)
	RecordFrameEvent(FrameEventDraw{timing})
	recordDraw(timing)
}

func (p *WindowProfiler) recordDrawDuration(duration time.Duration) {
	_ = p.drawDurationHistogram.Record(durationNanos(duration))
}

var maxFrameTimings = (16 * 1024 * 1024) / int(unsafe.Sizeof(FrameEventDraw{}))

var frameTimings struct {
	mu          sync.Mutex
	timings     []FrameEvent
	totalPushed uint64
}

// RecordFrameEvent records a frame event. No-op unless profiler tracing is enabled via SetTraceEnabled.
func RecordFrameEvent(event FrameEvent) {
	if !TraceEnabled() {
		return
	}
	frameTimings.mu.Lock()
	defer frameTimings.mu.Unlock()
	if len(frameTimings.timings) >= maxFrameTimings {
		frameTimings.timings[0] = nil
		frameTimings.timings = frameTimings.timings[1:]
	}
	frameTimings.timings = append(frameTimings.timings, event)
	frameTimings.totalPushed++
}

// FrameTimingCollector drains frame events recorded after it was created.
type FrameTimingCollector struct {
	cursor uint64
}

// NewFrameTimingCollector creates a collector that only sees frame events recorded from now on.
func NewFrameTimingCollector() *FrameTimingCollector {
	frameTimings.mu.Lock()
	defer frameTimings.mu.Unlock()
	return &FrameTimingCollector{cursor: frameTimings.totalPushed}
}

// CollectUnseen returns frame events recorded since the previous call or the collector's creation.
func (c *FrameTimingCollector) CollectUnseen() []FrameEvent {
	frameTimings.mu.Lock()
	defer frameTimings.mu.Unlock()
	bufferLen := uint64(len(frameTimings.timings))
	bufferStart := saturatingSubU64(frameTimings.totalPushed, bufferLen)
	skip := min(saturatingSubU64(c.cursor, bufferStart), bufferLen)
	unseen := slices.Clone(frameTimings.timings[skip:])
	if unseen == nil {
		unseen = []FrameEvent{}
	}
	c.cursor = frameTimings.totalPushed
	return unseen
}

const (
	histogramOriginalMin uint64 = math.MaxUint64
	histogramOriginalMax uint64 = 0
)

// Histogram is a port of the hdrhistogram crate's Histogram<u64> as the profiler uses it.
type Histogram struct {
	autoResize                  bool
	highestTrackableValue       uint64
	lowestDiscernibleValue      uint64
	significantValueDigits      uint8
	bucketCount                 uint8
	subBucketCount              uint32
	leadingZeroCountBase        uint8
	subBucketHalfCountMagnitude uint8
	unitMagnitude               uint8
	subBucketHalfCount          uint32
	subBucketMask               uint64
	unitMagnitudeMask           uint64
	maxValue                    uint64
	minNonZeroValue             uint64
	totalCount                  uint64
	counts                      []uint64
}

var (
	errHistogramSigFigExceedsMax                   = errors.New("sigfig exceeds max")
	errHistogramValueOutOfRangeResizeDisabled      = errors.New("value out of range and resize disabled")
	errHistogramSubtrahendValueExceedsMinuendRange = errors.New("subtrahend value exceeds minuend range")
	errHistogramSubtrahendCountExceedsMinuendCount = errors.New("subtrahend count exceeds minuend count")
)

// NewHistogram constructs an auto-resizing histogram with a lowest discernible value of 1.
func NewHistogram(sigfig uint8) (*Histogram, error) {
	h, err := newHistogramWithBounds(1, 2, sigfig)
	if err != nil {
		return nil, err
	}
	h.autoResize = true
	return h, nil
}

func newHistogramWithBounds(low, high uint64, sigfig uint8) (*Histogram, error) {
	if sigfig > 5 {
		return nil, errHistogramSigFigExceedsMax
	}
	largest := 2 * uint32(math.Pow10(int(sigfig)))
	unitMagnitude := uint8(math.Floor(math.Log2(float64(low))))
	unitMagnitudeMask := uint64(1)<<unitMagnitude - 1
	subBucketCountMagnitude := uint8(math.Ceil(math.Log2(float64(largest))))
	subBucketHalfCountMagnitude := subBucketCountMagnitude - 1
	subBucketCount := uint32(1) << subBucketCountMagnitude
	subBucketHalfCount := subBucketCount / 2
	subBucketMask := (uint64(subBucketCount) - 1) << unitMagnitude
	h := &Histogram{
		highestTrackableValue:       high,
		lowestDiscernibleValue:      low,
		significantValueDigits:      sigfig,
		subBucketCount:              subBucketCount,
		leadingZeroCountBase:        64 - unitMagnitude - subBucketCountMagnitude,
		subBucketHalfCountMagnitude: subBucketHalfCountMagnitude,
		unitMagnitude:               unitMagnitude,
		subBucketHalfCount:          subBucketHalfCount,
		subBucketMask:               subBucketMask,
		unitMagnitudeMask:           unitMagnitudeMask,
		maxValue:                    histogramOriginalMax,
		minNonZeroValue:             histogramOriginalMin,
	}
	h.resize(high)
	return h, nil
}

// Clone returns a deep copy.
func (h *Histogram) Clone() *Histogram {
	c := *h
	c.counts = slices.Clone(h.counts)
	return &c
}

// Len returns the total number of samples recorded.
func (h *Histogram) Len() uint64 { return h.totalCount }

// IsEmpty reports whether no values were recorded.
func (h *Histogram) IsEmpty() bool { return h.totalCount == 0 }

func (h *Histogram) indexFor(value uint64) int {
	bucketIndex := h.bucketFor(value)
	subBucketIndex := h.subBucketFor(value, bucketIndex)
	bucketBaseIndex := (int(bucketIndex) + 1) << h.subBucketHalfCountMagnitude
	offsetInBucket := int(subBucketIndex) - int(h.subBucketHalfCount)
	return bucketBaseIndex + offsetInBucket
}

func (h *Histogram) indexForOrLast(value uint64) int {
	return min(h.indexFor(value), h.lastIndex())
}

func (h *Histogram) lastIndex() int {
	return len(h.counts) - 1
}

// Record records value, growing the histogram as needed.
func (h *Histogram) Record(value uint64) error {
	if i := h.indexFor(value); i < len(h.counts) {
		h.counts[i] = saturatingAddU64(h.counts[i], 1)
	} else {
		if !h.autoResize {
			return errHistogramValueOutOfRangeResizeDisabled
		}
		h.resize(value)
		h.highestTrackableValue = h.highestEquivalent(h.valueFor(h.lastIndex()))
		h.counts[h.indexFor(value)]++
	}
	h.updateMinMax(value)
	h.totalCount = saturatingAddU64(h.totalCount, 1)
	return nil
}

func saturatingAddU64(a, b uint64) uint64 {
	s, carry := bits.Add64(a, b, 0)
	if carry != 0 {
		return math.MaxUint64
	}
	return s
}

// Subtract subtracts subtrahend's counts from h.
func (h *Histogram) Subtract(subtrahend *Histogram) error {
	if subtrahend.IsEmpty() {
		return nil
	}
	top := h.highestEquivalent(h.valueFor(h.lastIndex()))
	if top < h.highestEquivalent(subtrahend.Max()) {
		return errHistogramSubtrahendValueExceedsMinuendRange
	}
	oldMinHighestEquiv := h.highestEquivalent(h.Min())
	oldMaxLowestEquiv := h.lowestEquivalent(h.Max())
	needsRestat := h.totalCount == math.MaxUint64
	for i, otherCount := range subtrahend.counts {
		if otherCount == 0 {
			continue
		}
		otherValue := subtrahend.valueFor(i)
		index := h.indexFor(otherValue)
		if index >= len(h.counts) {
			panic(fmt.Sprintf("Tried to subtract value outside of range: %d", otherValue))
		}
		if h.counts[index] < otherCount {
			return errHistogramSubtrahendCountExceedsMinuendCount
		}
		h.counts[index] -= otherCount
		if otherValue <= oldMinHighestEquiv || otherValue >= oldMaxLowestEquiv {
			needsRestat = true
		}
		if !needsRestat {
			if h.totalCount < otherCount {
				panic("total count underflow on subtraction")
			}
			h.totalCount -= otherCount
		}
	}
	if needsRestat {
		h.restat(len(h.counts))
	}
	return nil
}

// Min returns the lowest recorded value level, or 0 when empty.
func (h *Histogram) Min() uint64 {
	if h.totalCount == 0 || h.counts[0] != 0 {
		return 0
	}
	return h.MinNz()
}

// Max returns the highest recorded value level.
func (h *Histogram) Max() uint64 {
	if h.maxValue == histogramOriginalMax {
		return histogramOriginalMax
	}
	return h.highestEquivalent(h.maxValue)
}

// MinNz returns the lowest recorded non-zero value level.
func (h *Histogram) MinNz() uint64 {
	if h.minNonZeroValue == histogramOriginalMin {
		return histogramOriginalMin
	}
	return h.lowestEquivalent(h.minNonZeroValue)
}

// Mean returns the mean of all recorded values.
func (h *Histogram) Mean() float64 {
	if h.totalCount == 0 {
		return 0
	}
	total := 0.0
	for i, count := range h.counts {
		if count == 0 {
			continue
		}
		total += float64(h.medianEquivalent(h.valueFor(i))) * float64(count) / float64(h.totalCount)
	}
	return total
}

// ValueAtQuantile returns the value at the given quantile.
func (h *Histogram) ValueAtQuantile(quantile float64) uint64 {
	if quantile > 1.0 {
		quantile = 1.0
	}
	countAtQuantile := uint64(math.Ceil(quantile * float64(h.totalCount)))
	if countAtQuantile == 0 {
		countAtQuantile = 1
	}
	var totalToCurrentIndex uint64
	for i, count := range h.counts {
		totalToCurrentIndex += count
		if totalToCurrentIndex >= countAtQuantile {
			valueAtIndex := h.valueFor(i)
			if quantile == 0.0 {
				return h.lowestEquivalent(valueAtIndex)
			}
			return h.highestEquivalent(valueAtIndex)
		}
	}
	return 0
}

// CountBetween returns the count of values between low and high, inclusive at value resolution.
func (h *Histogram) CountBetween(low, high uint64) uint64 {
	lowIndex := h.indexForOrLast(low)
	highIndex := h.indexForOrLast(high)
	var t uint64
	for i := lowIndex; i <= highIndex; i++ {
		t = saturatingAddU64(t, h.counts[i])
	}
	return t
}

func (h *Histogram) lowestEquivalent(value uint64) uint64 {
	bucketIndex := h.bucketFor(value)
	return h.valueFromLoc(bucketIndex, h.subBucketFor(value, bucketIndex))
}

func (h *Histogram) highestEquivalent(value uint64) uint64 {
	if value == math.MaxUint64 {
		return math.MaxUint64
	}
	return h.nextNonEquivalent(value) - 1
}

func (h *Histogram) medianEquivalent(value uint64) uint64 {
	return h.lowestEquivalent(value) + h.equivalentRange(value)>>1
}

func (h *Histogram) nextNonEquivalent(value uint64) uint64 {
	return saturatingAddU64(h.lowestEquivalent(value), h.equivalentRange(value))
}

func (h *Histogram) equivalentRange(value uint64) uint64 {
	return uint64(1) << (h.unitMagnitude + h.bucketFor(value))
}

func (h *Histogram) valueFor(index int) uint64 {
	bucketIndex := (index >> h.subBucketHalfCountMagnitude) - 1
	subBucketIndex := (uint32(index) & (h.subBucketHalfCount - 1)) + h.subBucketHalfCount
	if bucketIndex < 0 {
		subBucketIndex -= h.subBucketHalfCount
		bucketIndex = 0
	}
	return h.valueFromLoc(uint8(bucketIndex), subBucketIndex)
}

func (h *Histogram) bucketFor(value uint64) uint8 {
	return h.leadingZeroCountBase - uint8(bits.LeadingZeros64(value|h.subBucketMask))
}

func (h *Histogram) subBucketFor(value uint64, bucketIndex uint8) uint32 {
	return uint32(value >> (bucketIndex + h.unitMagnitude))
}

func (h *Histogram) valueFromLoc(bucketIndex uint8, subBucketIndex uint32) uint64 {
	return uint64(subBucketIndex) << (bucketIndex + h.unitMagnitude)
}

func (h *Histogram) bucketsToCover(value uint64) uint8 {
	smallestUntrackableValue := uint64(h.subBucketCount) << h.unitMagnitude
	bucketsNeeded := uint8(1)
	for smallestUntrackableValue <= value {
		if smallestUntrackableValue > math.MaxUint64/2 {
			return bucketsNeeded + 1
		}
		smallestUntrackableValue <<= 1
		bucketsNeeded++
	}
	return bucketsNeeded
}

func (h *Histogram) resize(high uint64) {
	if high < 2*h.lowestDiscernibleValue {
		panic("highest trackable value must be >= (2 * lowest discernible value)")
	}
	bucketsNeeded := h.bucketsToCover(high)
	length := int((uint32(bucketsNeeded) + 1) * h.subBucketHalfCount)
	h.bucketCount = bucketsNeeded
	h.highestTrackableValue = high
	if length > len(h.counts) {
		h.counts = append(h.counts, make([]uint64, length-len(h.counts))...)
	} else {
		h.counts = h.counts[:length]
	}
}

func (h *Histogram) updateMax(value uint64) {
	internalValue := value | h.unitMagnitudeMask
	if internalValue > h.maxValue {
		h.maxValue = internalValue
	}
}

func (h *Histogram) updateMin(value uint64) {
	if value <= h.unitMagnitudeMask {
		return
	}
	internalValue := value &^ h.unitMagnitudeMask
	if internalValue < h.minNonZeroValue {
		h.minNonZeroValue = internalValue
	}
}

func (h *Histogram) updateMinMax(value uint64) {
	if value > h.maxValue {
		h.updateMax(value)
	}
	if value < h.minNonZeroValue && value != 0 {
		h.updateMin(value)
	}
}

func (h *Histogram) restat(lengthToScan int) {
	h.maxValue = histogramOriginalMax | h.unitMagnitudeMask
	h.minNonZeroValue = histogramOriginalMin
	maxIndex, minIndex := -1, -1
	var totalCount uint64
	for i := range lengthToScan {
		count := h.counts[i]
		if count == 0 {
			continue
		}
		totalCount = saturatingAddU64(totalCount, count)
		maxIndex = i
		if minIndex < 0 && i != 0 {
			minIndex = i
		}
	}
	if maxIndex >= 0 {
		h.updateMax(h.highestEquivalent(h.valueFor(maxIndex)))
	}
	if minIndex >= 0 {
		h.updateMin(h.valueFor(minIndex))
	}
	h.totalCount = totalCount
}
