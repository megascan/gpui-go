package gpui

import (
	"fmt"
	"os"
	"reflect"
	"runtime/metrics"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

var benchDispatchers sync.Map

// BenchPlatform returns a benchmark platform backed by the calling goroutine's shared
// ThreadedDispatcher, so background work runs with production concurrency in real time. The
// dispatcher is cached per goroutine and reused across benchmark invocations; RunBench runs
// every invocation on one goroutine so worker and timer goroutines persist for the process.
//
// Text is shaped with textSystem. headlessRendererFactory supplies a renderer for benchmark
// windows; when nil, presenting discards the scene.
func BenchPlatform(headlessRendererFactory func() (PlatformHeadlessRenderer, error), textSystem PlatformTextSystem) Platform {
	id := goroutineID()
	cached, ok := benchDispatchers.Load(id)
	if !ok {
		cached, _ = benchDispatchers.LoadOrStore(id, NewThreadedDispatcher())
	}
	dispatcher := cached.(*ThreadedDispatcher)
	backgroundExecutor := NewBackgroundExecutor(dispatcher)
	foregroundExecutor := NewForegroundExecutor(dispatcher)
	return TestPlatformWithPlatform(backgroundExecutor, foregroundExecutor, textSystem, headlessRendererFactory)
}

const defaultBenchFps uint64 = 120

const nanosPerSecond uint64 = 1_000_000_000

// ForegroundWorkSummary is aggregate statistics for total foreground executor work observed
// during a measured interval, returned by BenchReport.ForegroundWork.
type ForegroundWorkSummary struct {
	// Count is the number of foreground work items recorded: task polls, action handlers,
	// input dispatches, and folded sub-floor poll flushes.
	Count uint64
	// Total is the sum of every recorded item's duration.
	Total time.Duration
	// Max is the longest single recorded item.
	Max time.Duration
	// P50 is the 50th percentile duration.
	P50 time.Duration
	// P90 is the 90th percentile duration.
	P90 time.Duration
	// P95 is the 95th percentile duration.
	P95 time.Duration
	// P99 is the 99th percentile duration.
	P99 time.Duration
	// FrameBudgetOverrunsTotal is how many whole frame budgets were exceeded in total, summed
	// across every recorded item.
	FrameBudgetOverrunsTotal uint64
	// FrameBudgetOverrunsMax is how many whole frame budgets the longest item exceeded.
	FrameBudgetOverrunsMax uint64
}

// BenchReport is a small report produced by GPUI benchmarks. Copies share their state.
type BenchReport struct {
	frameSnapshot    *windowFrameSnapshot
	metrics          *MetricReport
	frameBudgetNanos uint64
}

// DefaultBenchReport returns a report at the default frame rate (120 fps).
func DefaultBenchReport() BenchReport {
	return BenchReportWithFps(defaultBenchFps)
}

// BenchReportWithFps creates a report whose per-frame budget is one frame at fps when counting
// frame budget overruns.
func BenchReportWithFps(fps uint64) BenchReport {
	if fps == 0 {
		panic("frame rate must be greater than zero")
	}
	return BenchReportWithFrameBudgetNanos(nanosPerSecond / fps)
}

// BenchReportWithFrameBudgetNanos creates a report that treats frameBudgetNanos as the
// per-frame budget when counting frame budget overruns.
func BenchReportWithFrameBudgetNanos(frameBudgetNanos uint64) BenchReport {
	if frameBudgetNanos == 0 {
		panic("frame budget must be at least one nanosecond")
	}
	return BenchReport{
		frameSnapshot:    newWindowFrameSnapshot(),
		metrics:          NewMetricReport(),
		frameBudgetNanos: frameBudgetNanos,
	}
}

// Metrics returns the per-iteration secondary metrics recorded alongside the primary
// measurement.
func (r BenchReport) Metrics() *MetricReport {
	return r.metrics
}

func (r BenchReport) recordFrameTimings(events []FrameEvent) {
	snapshot := r.frameSnapshot
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	for _, event := range events {
		switch event := event.(type) {
		case FrameEventDraw:
			_ = snapshot.draw.Record(uint64(event.DrawDuration().Nanoseconds()))
			if dirtyToDraw, ok := event.DirtyToDrawDuration(); ok {
				_ = snapshot.dirtyToDraw.Record(uint64(dirtyToDraw.Nanoseconds()))
			}
			if event.Invalidations > 0 {
				_ = snapshot.invalidationsPerFrame.Record(event.Invalidations)
			}
		case FrameEventPresent:
			if animationInterval, ok := event.AnimationInterval.Get(); ok {
				_ = snapshot.presentInterval.Record(uint64(animationInterval.Nanoseconds()))
			}
		}
	}
}

func (r BenchReport) recordForegroundEvents(events []ForegroundEvent) {
	snapshot := r.frameSnapshot
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	for _, event := range events {
		var duration time.Duration
		switch event := event.(type) {
		case ForegroundEventDraw, ForegroundEventPresent:
			continue
		case ForegroundEventSmallPolls:
			duration = event.Summary.Total
		default:
			duration = event.Duration()
		}
		snapshot.foregroundWork.record(duration)
	}
}

func (r BenchReport) totalBudgetOverruns(histogram *Histogram) uint64 {
	var total uint64
	for index, count := range histogram.counts {
		if count == 0 {
			continue
		}
		value := histogram.highestEquivalent(histogram.valueFor(index))
		total += r.budgetOverruns(time.Duration(value)) * count
	}
	return total
}

func (r BenchReport) budgetOverruns(foregroundTime time.Duration) uint64 {
	foregroundNanos := uint64(foregroundTime.Nanoseconds())
	if foregroundNanos <= r.frameBudgetNanos {
		return 0
	}
	overBudgetNanos := foregroundNanos - r.frameBudgetNanos
	return (overBudgetNanos + r.frameBudgetNanos - 1) / r.frameBudgetNanos
}

// ForegroundWork returns aggregate statistics for total foreground executor work observed
// during the measured interval: every task poll, action handler and input dispatch on the
// foreground thread, whether or not it produced a window draw. It reports false when no
// foreground work was recorded.
func (r BenchReport) ForegroundWork() (ForegroundWorkSummary, bool) {
	r.frameSnapshot.mu.Lock()
	defer r.frameSnapshot.mu.Unlock()
	foregroundWork := r.frameSnapshot.foregroundWork
	if foregroundWork.histogram.IsEmpty() {
		return ForegroundWorkSummary{}, false
	}
	maxDuration := time.Duration(foregroundWork.histogram.Max())
	return ForegroundWorkSummary{
		Count:                    foregroundWork.histogram.Len(),
		Total:                    time.Duration(foregroundWork.totalNanos),
		Max:                      maxDuration,
		P50:                      time.Duration(foregroundWork.histogram.ValueAtQuantile(0.50)),
		P90:                      time.Duration(foregroundWork.histogram.ValueAtQuantile(0.90)),
		P95:                      time.Duration(foregroundWork.histogram.ValueAtQuantile(0.95)),
		P99:                      time.Duration(foregroundWork.histogram.ValueAtQuantile(0.99)),
		FrameBudgetOverrunsTotal: r.totalBudgetOverruns(foregroundWork.histogram),
		FrameBudgetOverrunsMax:   r.budgetOverruns(maxDuration),
	}, true
}

// Print prints this report to stderr.
func (r BenchReport) Print(benchmarkName string) {
	frameSnapshot := r.frameSnapshot
	frameSnapshot.mu.Lock()
	defer frameSnapshot.mu.Unlock()
	if frameSnapshot.isEmpty() && r.metrics.IsEmpty() {
		return
	}

	fmt.Fprintf(os.Stderr, "GPUI bench report (all observed iterations): %s\n", benchmarkName)
	fmt.Fprintln(os.Stderr, "  note: includes Criterion warmup/calibration")
	r.metrics.Print("  ")
	r.printHistogram("window dirty-to-draw", frameSnapshot.dirtyToDraw)
	r.printHistogram("window draw", frameSnapshot.draw)
	r.printHistogram("window present interval", frameSnapshot.presentInterval)
	if !frameSnapshot.invalidationsPerFrame.IsEmpty() {
		fmt.Fprintf(os.Stderr, "  invalidations per frame: mean %.2f, max %d\n",
			frameSnapshot.invalidationsPerFrame.Mean(),
			frameSnapshot.invalidationsPerFrame.Max())
	}
	r.printForegroundWork(frameSnapshot.foregroundWork)
}

func (r BenchReport) printHistogram(name string, histogram *Histogram) {
	if histogram.IsEmpty() {
		return
	}
	fmt.Fprintf(os.Stderr, "  %s:\n", name)
	r.printHistogramBody(histogram)
}

func (r BenchReport) printForegroundWork(foregroundWork *durationHistogram) {
	if foregroundWork.histogram.IsEmpty() {
		return
	}
	fmt.Fprintln(os.Stderr, "  foreground executor work (task polls, actions, input dispatch):")
	fmt.Fprintln(os.Stderr, "    note: excludes window draw/present, reported separately above")
	fmt.Fprintf(os.Stderr, "    total: %s\n", formatBenchDuration(time.Duration(foregroundWork.totalNanos)))
	r.printHistogramBody(foregroundWork.histogram)
}

func (r BenchReport) printHistogramBody(histogram *Histogram) {
	maxForegroundTime := time.Duration(histogram.Max())
	fmt.Fprintf(os.Stderr, "    samples: %d\n", histogram.Len())
	fmt.Fprintf(os.Stderr, "    mean: %s\n", formatBenchDuration(time.Duration(uint64(histogram.Mean()))))
	fmt.Fprintf(os.Stderr, "    p50: %s\n", formatBenchDuration(time.Duration(histogram.ValueAtQuantile(0.50))))
	fmt.Fprintf(os.Stderr, "    p90: %s\n", formatBenchDuration(time.Duration(histogram.ValueAtQuantile(0.90))))
	fmt.Fprintf(os.Stderr, "    p95: %s\n", formatBenchDuration(time.Duration(histogram.ValueAtQuantile(0.95))))
	fmt.Fprintf(os.Stderr, "    p99: %s\n", formatBenchDuration(time.Duration(histogram.ValueAtQuantile(0.99))))
	fmt.Fprintf(os.Stderr, "    max: %s\n", formatBenchDuration(maxForegroundTime))
	fmt.Fprintf(os.Stderr, "    frame budget overruns total: %d\n", r.totalBudgetOverruns(histogram))
	fmt.Fprintf(os.Stderr, "    frame budget overruns max: %d\n", r.budgetOverruns(maxForegroundTime))
}

type windowFrameSnapshot struct {
	mu                    sync.Mutex
	dirtyToDraw           *Histogram
	draw                  *Histogram
	presentInterval       *Histogram
	invalidationsPerFrame *Histogram
	foregroundWork        *durationHistogram
}

func newBenchHistogram() *Histogram {
	h, err := NewHistogram(3)
	if err != nil {
		panic("3 significant digits is valid")
	}
	return h
}

func newWindowFrameSnapshot() *windowFrameSnapshot {
	return &windowFrameSnapshot{
		dirtyToDraw:           newBenchHistogram(),
		draw:                  newBenchHistogram(),
		presentInterval:       newBenchHistogram(),
		invalidationsPerFrame: newBenchHistogram(),
		foregroundWork:        newDurationHistogram(),
	}
}

func (s *windowFrameSnapshot) isEmpty() bool {
	return s.dirtyToDraw.IsEmpty() &&
		s.draw.IsEmpty() &&
		s.presentInterval.IsEmpty() &&
		s.foregroundWork.histogram.IsEmpty()
}

type durationHistogram struct {
	histogram  *Histogram
	totalNanos uint64
}

func newDurationHistogram() *durationHistogram {
	return &durationHistogram{histogram: newBenchHistogram()}
}

func (h *durationHistogram) record(duration time.Duration) {
	nanos := uint64(duration.Nanoseconds())
	_ = h.histogram.Record(nanos)
	h.totalNanos += nanos
}

func formatBenchDuration(duration time.Duration) string {
	return fmt.Sprintf("%.3fms", duration.Seconds()*1000.)
}

// BenchCounter reads a monotonically increasing counter, such as allocations so far.
type BenchCounter func() uint64

type benchSecondary struct {
	name    string
	counter BenchCounter
}

// BenchMeasurement is the measurement a GPUI benchmark runs with: wall time from testing.B,
// plus secondary counters recorded per iteration into the report's MetricReport.
type BenchMeasurement struct {
	secondaries []benchSecondary
	started     []uint64
}

// NewBenchMeasurement returns a wall-time measurement with no secondary counters.
func NewBenchMeasurement() *BenchMeasurement {
	return &BenchMeasurement{}
}

// WithSecondary adds a secondary counter reported per iteration under name.
func (m *BenchMeasurement) WithSecondary(name string, counter BenchCounter) *BenchMeasurement {
	m.secondaries = append(m.secondaries, benchSecondary{name: name, counter: counter})
	return m
}

func runtimeMetricCounter(name string) BenchCounter {
	return func() uint64 {
		sample := []metrics.Sample{{Name: name}}
		metrics.Read(sample)
		if sample[0].Value.Kind() != metrics.KindUint64 {
			return 0
		}
		return sample[0].Value.Uint64()
	}
}

// BenchMeasurementFromEnv returns the measurement selected by $BENCH_MEASUREMENT: unset or
// "default" records heap allocations and allocated bytes per iteration, "wall-time" disables
// every counter.
func BenchMeasurementFromEnv() (*BenchMeasurement, error) {
	switch value := os.Getenv("BENCH_MEASUREMENT"); value {
	case "", "default":
		return NewBenchMeasurement().
			WithSecondary("heap allocations", runtimeMetricCounter("/gc/heap/allocs:objects")).
			WithSecondary("heap bytes allocated", runtimeMetricCounter("/gc/heap/allocs:bytes")), nil
	case "wall-time":
		return NewBenchMeasurement(), nil
	default:
		return nil, fmt.Errorf("unsupported BENCH_MEASUREMENT %q: expected \"default\" or \"wall-time\"", value)
	}
}

// BenchMeasurementFromEnvOrExit is BenchMeasurementFromEnv, exiting the process on an invalid
// $BENCH_MEASUREMENT.
func BenchMeasurementFromEnvOrExit() *BenchMeasurement {
	measurement, err := BenchMeasurementFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	return measurement
}

func (m *BenchMeasurement) start() {
	m.started = m.started[:0]
	for _, secondary := range m.secondaries {
		m.started = append(m.started, secondary.counter())
	}
}

func (m *BenchMeasurement) stop(report *MetricReport) {
	for i, secondary := range m.secondaries {
		end := secondary.counter()
		var delta uint64
		if end > m.started[i] {
			delta = end - m.started[i]
		}
		report.addPending(secondary.name, delta)
	}
}

// MetricReport collects per-iteration secondary metrics. It is safe for concurrent use.
type MetricReport struct {
	mu      sync.Mutex
	order   []string
	pending map[string]uint64
	values  map[string][]float64
}

// NewMetricReport returns an empty report.
func NewMetricReport() *MetricReport {
	return &MetricReport{pending: map[string]uint64{}, values: map[string][]float64{}}
}

func (r *MetricReport) addPending(name string, delta uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[name]; !ok {
		r.values[name] = nil
		r.order = append(r.order, name)
	}
	r.pending[name] += delta
}

func (r *MetricReport) discardPending() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.pending)
}

func (r *MetricReport) recordSample(iterations uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if iterations > 0 {
		for _, name := range r.order {
			total, ok := r.pending[name]
			if !ok {
				continue
			}
			r.values[name] = append(r.values[name], float64(total)/float64(iterations))
		}
	}
	clear(r.pending)
}

// Values returns the per-iteration values recorded for the named metric, one per sample.
func (r *MetricReport) Values(name string) ([]float64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values, ok := r.values[name]
	if !ok {
		return nil, false
	}
	return append([]float64(nil), values...), true
}

// IsEmpty reports whether no sample was recorded.
func (r *MetricReport) IsEmpty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, values := range r.values {
		if len(values) > 0 {
			return false
		}
	}
	return true
}

// Print prints the mean per-iteration value of every metric to stderr, each line prefixed.
func (r *MetricReport) Print(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, name := range r.order {
		values := r.values[name]
		if len(values) == 0 {
			continue
		}
		var sum float64
		for _, value := range values {
			sum += value
		}
		fmt.Fprintf(os.Stderr, "%s%s per iteration: mean %.2f (%d samples)\n", prefix, name, sum/float64(len(values)), len(values))
	}
}

// Bencher is the measured loop a BenchAppContext drives: Go's testing.B with the
// measurement whose secondary counters are recorded into the report.
type Bencher struct {
	// B is the benchmark whose iteration count and timer are used.
	B *testing.B
	// Measurement supplies the secondary counters; nil records none.
	Measurement *BenchMeasurement
}

func (bn *Bencher) measurement() *BenchMeasurement {
	if bn.Measurement == nil {
		bn.Measurement = NewBenchMeasurement()
	}
	return bn.Measurement
}

func (bn *Bencher) iter(report *MetricReport, routine func()) {
	m := bn.measurement()
	bn.B.StartTimer()
	m.start()
	for range bn.B.N {
		routine()
	}
	m.stop(report)
	bn.B.StopTimer()
}

func benchIterBatchedRef[I, O any](bn *Bencher, report *MetricReport, setup func() I, routine func(input *I) O, dropOutput func(output O), dropInput func(input I)) {
	m := bn.measurement()
	for range bn.B.N {
		input := setup()
		bn.B.StartTimer()
		m.start()
		output := routine(&input)
		m.stop(report)
		bn.B.StopTimer()
		dropOutput(output)
		dropInput(input)
	}
}

func releaseBenchValue(value any) {
	if releaser, ok := value.(interface{ Release() }); ok {
		releaser.Release()
	}
}

type benchTraceScope struct {
	collector        *FrameTimingCollector
	journalCollector *ForegroundJournalCollector
	traceGuard       traceGuard
}

func startBenchTraceScope(journalCollector *ForegroundJournalCollector) *benchTraceScope {
	guard := traceScope()
	return &benchTraceScope{
		collector:        NewFrameTimingCollector(),
		journalCollector: journalCollector,
		traceGuard:       guard,
	}
}

func (s *benchTraceScope) finish() tracedEvents {
	events := tracedEvents{
		frameEvents:    s.collector.CollectUnseen(),
		journalEntries: s.journalCollector.CollectUnseen().Entries,
	}
	s.traceGuard.End()
	return events
}

func (s *benchTraceScope) end() {
	s.traceGuard.End()
}

type tracedEvents struct {
	frameEvents    []FrameEvent
	journalEntries []ForegroundJournalEntry
}

func (e tracedEvents) foregroundEvents() []ForegroundEvent {
	events := make([]ForegroundEvent, 0, len(e.journalEntries))
	for _, entry := range e.journalEntries {
		if event, ok := entry.(ForegroundJournalEntryEvent); ok {
			events = append(events, event.Event)
		}
	}
	return events
}

type measuredTaskInput[I any] struct {
	input      I
	traceScope *benchTraceScope
}

type measuredTaskOutput[O any] struct {
	traceScope *benchTraceScope
	report     BenchReport
	output     O
}

func (o *measuredTaskOutput[O]) Close() {
	traceScope := o.traceScope
	if traceScope == nil {
		panic("measured task output should retain its trace scope")
	}
	o.traceScope = nil
	events := traceScope.finish()
	o.report.recordFrameTimings(events.frameEvents)
	o.report.recordForegroundEvents(events.foregroundEvents())
	releaseBenchValue(o.output)
}

func runTaskToCompletion[O any](foregroundExecutor ForegroundExecutor, dispatcher *ThreadedDispatcher, task *Task[O]) O {
	var output Option[O]
	foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		value, err := task.Await(s)
		if err != nil {
			return struct{}{}, err
		}
		output = Some(value)
		return struct{}{}, nil
	}).Detach()
	if dispatcher == nil {
		panic("BenchAppContext requires a ThreadedDispatcher")
	}
	dispatcher.runUntil(func() bool { return output.IsSome() })
	value, _ := output.Get()
	return value
}

type benchBencherSlot struct {
	bencher *Bencher
}

// BenchAppContext is a GPUI app context for benchmarks. It owns a benchmark app instance and
// exposes only the app and window operations benchmark setup needs; testing.B remains
// responsible for the measured loop. Copies made by the context share the app and bencher.
type BenchAppContext struct {
	app                *AppCell
	bgExecutor         BackgroundExecutor
	foregroundExecutor ForegroundExecutor
	benchmarkName      Option[string]
	bencher            *benchBencherSlot
	report             BenchReport
}

var _ AppContext = (*BenchAppContext)(nil)

// NewBenchAppContext creates a benchmark app context backed by platform, whose executors must
// be backed by a ThreadedDispatcher (see BenchPlatform); it panics otherwise.
func NewBenchAppContext(platform Platform, benchmarkName Option[string], bencher *Bencher) *BenchAppContext {
	return buildBenchAppContext(platform, benchmarkName, bencher, DefaultBenchReport())
}

// NewBenchAppContextWithPlatformAndReport creates a benchmark app context backed by platform
// that records into report.
func NewBenchAppContextWithPlatformAndReport(platform Platform, benchmarkName Option[string], bencher *Bencher, report BenchReport) *BenchAppContext {
	return buildBenchAppContext(platform, benchmarkName, bencher, report)
}

func buildBenchAppContext(platform Platform, benchmarkName Option[string], bencher *Bencher, report BenchReport) *BenchAppContext {
	backgroundExecutor := platform.BackgroundExecutor()
	if backgroundExecutor.Dispatcher().AsThreaded() == nil {
		panic("BenchAppContext requires a platform whose executors are backed by a ThreadedDispatcher; construct one with gpui::bench_platform")
	}
	foregroundExecutor := platform.ForegroundExecutor()
	testWindowFrameWakerSchedulesFrame = true
	app := newApp(platform, NoAssets{}, NullHTTPClient{})
	if app.app.foregroundJournal.ring == nil {
		app.app.foregroundJournal = installForegroundJournal()
	}
	if bencher.B != nil {
		bencher.B.StopTimer()
		bencher.B.ResetTimer()
	}
	return &BenchAppContext{
		app:                app,
		bgExecutor:         backgroundExecutor,
		foregroundExecutor: foregroundExecutor,
		benchmarkName:      benchmarkName,
		bencher:            &benchBencherSlot{bencher: bencher},
		report:             report,
	}
}

func (cx *BenchAppContext) clone() *BenchAppContext {
	c := *cx
	return &c
}

func (cx *BenchAppContext) dispatcher() *ThreadedDispatcher {
	dispatcher := cx.bgExecutor.Dispatcher().AsThreaded()
	if dispatcher == nil {
		panic("validated in BenchAppContext::build")
	}
	return dispatcher
}

// BenchmarkName returns the benchmark function name that created this context.
func (cx *BenchAppContext) BenchmarkName() (string, bool) {
	return cx.benchmarkName.Get()
}

// BackgroundExecutor returns the background executor used by this benchmark app.
func (cx *BenchAppContext) BackgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

// ForegroundExecutor returns the foreground executor used by this benchmark app.
func (cx *BenchAppContext) ForegroundExecutor() ForegroundExecutor {
	return cx.foregroundExecutor
}

// Update updates the app and flushes synchronous GPUI effects afterward.
func (cx *BenchAppContext) Update(update func(cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.update(update)
}

// UpdateR is Update for a callback that returns a value.
func (cx *BenchAppContext) UpdateR[R any](update func(cx *App) R) R {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.updateR(update)
}

// Read reads app state.
func (cx *BenchAppContext) Read(read func(cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	read(app.Get())
}

// ReadR is Read for a callback that returns a value.
func (cx *BenchAppContext) ReadR[R any](read func(cx *App) R) R {
	app := cx.app.Borrow()
	defer app.Release()
	return read(app.Get())
}

// RunUntilIdle runs queued foreground tasks on this goroutine and waits for in-flight
// background work to finish. Timers that are not due yet are not waited for. Scheduled frames
// are delivered after each task poll and when there are no ready tasks.
func (cx *BenchAppContext) RunUntilIdle() {
	dispatcher := cx.dispatcher()
	dispatcher.runUntilIdleWith(func() bool {
		ranTasks := dispatcher.runReadyMainTasksWith(func() bool { return true }, func() {
			cx.dispatchPendingFrames(func() bool { return true })
		})
		if !ranTasks {
			cx.dispatchPendingFrames(func() bool { return true })
		}
		return ranTasks
	})
}

// Settle alternates draining queued work with GPUI update cycles until neither makes progress,
// so state released by benchmark code is fully released.
func (cx *BenchAppContext) Settle() {
	dispatcher := cx.dispatcher()
	for {
		cx.RunUntilIdle()
		cx.Update(func(*App) {})
		cx.dispatchPendingFrames(func() bool { return true })
		if dispatcher.isIdle() {
			return
		}
	}
}

// RunUntil runs main-thread tasks until ready returns a value. While incomplete, scheduled
// frames run after each task poll and while waiting for work.
func (cx *BenchAppContext) RunUntil[R any](ready func() (R, bool)) R {
	var result R
	cx.dispatcher().runUntilWithFrames(func() bool {
		value, ok := ready()
		if ok {
			result = value
		}
		return ok
	}, func() bool { return cx.dispatchPendingFrames(func() bool { return true }) })
	return result
}

func (cx *BenchAppContext) foregroundJournalCollector() *ForegroundJournalCollector {
	return cx.ReadR(func(app *App) *ForegroundJournalCollector {
		return app.ForegroundJournal().Collector()
	})
}

// BenchIter measures a generic workload with the benchmark's iteration loop. benchmark runs
// once per iteration, then the scheduled frame batch is delivered without draining foreground
// tasks. Window draws are recorded into the benchmark's frame report.
func (cx *BenchAppContext) BenchIter(benchmark func(cx *BenchAppContext)) {
	bencher := cx.takeBencher("bench_iter")
	cx.dispatchPendingFrames(func() bool { return true })
	collector := startBenchTraceScope(cx.foregroundJournalCollector())
	defer collector.end()
	cx.report.metrics.discardPending()
	var iterations uint64
	bencher.iter(cx.report.metrics, func() {
		iterations++
		benchmark(cx)
		cx.dispatchPendingFrames(func() bool { return true })
	})
	events := collector.finish()
	cx.report.metrics.recordSample(iterations)
	cx.report.recordFrameTimings(events.frameEvents)
	cx.report.recordForegroundEvents(events.foregroundEvents())
	cx.replaceBencher(bencher)
}

// BenchTask measures a GPUI task to completion with the benchmark's iteration loop. benchmark
// runs once per iteration; the task's output is released after the timed interval.
func (cx *BenchAppContext) BenchTask[O any](benchmark func(cx *BenchAppContext) *Task[O]) {
	benchBatchedTaskInternal(cx, "bench_task", func(*BenchAppContext) struct{} { return struct{}{} }, func(_ *struct{}, cx *BenchAppContext) *Task[O] {
		return benchmark(cx)
	})
}

// BenchBatchedTask measures a GPUI task with per-iteration setup outside the timed interval.
// setup's input is passed to benchmark, which returns the task whose completion is measured.
// The task output and then the input are released (Release, when they have one) after timing
// stops. Each iteration is its own batch, so tracing and destruction cannot overlap adjacent
// measurements.
func (cx *BenchAppContext) BenchBatchedTask[I, O any](setup func(cx *BenchAppContext) I, benchmark func(input *I, cx *BenchAppContext) *Task[O]) {
	benchBatchedTaskInternal(cx, "bench_batched_task", setup, benchmark)
}

func benchBatchedTaskInternal[I, O any](cx *BenchAppContext, benchmarkKind string, setup func(cx *BenchAppContext) I, benchmark func(input *I, cx *BenchAppContext) *Task[O]) {
	bencher := cx.takeBencher(benchmarkKind)
	setupContext := cx.clone()
	benchmarkContext := cx.clone()
	foregroundExecutor := cx.foregroundExecutor
	report := cx.report
	report.metrics.discardPending()
	var iterations uint64

	benchIterBatchedRef(bencher, report.metrics,
		func() measuredTaskInput[I] {
			setupContext.Settle()
			input := setup(setupContext)
			setupContext.dispatchPendingFrames(func() bool { return true })
			return measuredTaskInput[I]{
				input:      input,
				traceScope: startBenchTraceScope(setupContext.foregroundJournalCollector()),
			}
		},
		func(measuredInput *measuredTaskInput[I]) *measuredTaskOutput[O] {
			iterations++
			task := benchmark(&measuredInput.input, benchmarkContext)
			benchmarkContext.dispatchPendingFrames(func() bool { return true })
			var output Option[O]
			completion := foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
				value, err := task.Await(s)
				if err != nil {
					return struct{}{}, err
				}
				output = Some(value)
				return struct{}{}, nil
			})
			value := benchmarkContext.RunUntil(func() (O, bool) { return output.Get() })
			completion.Cancel()
			traceScope := measuredInput.traceScope
			measuredInput.traceScope = nil
			return &measuredTaskOutput[O]{
				traceScope: traceScope,
				report:     report,
				output:     value,
			}
		},
		func(output *measuredTaskOutput[O]) { output.Close() },
		func(input measuredTaskInput[I]) {
			if input.traceScope != nil {
				input.traceScope.end()
			}
			releaseBenchValue(input.input)
		},
	)
	report.metrics.recordSample(iterations)
	cx.replaceBencher(bencher)
}

// BenchRenderer measures unpaced update and rendering work for an entity in its current
// window. Each iteration pumps at most the initial ready task count, delivering pending frames
// after every poll, then runs update and delivers pending frames again.
func (cx *BenchAppContext) BenchRenderer[V any](view Entity[V], update func(view *V, window *Window, cx *Context[V])) {
	bencher := cx.takeBencher("bench_renderer")
	dispatcher := cx.dispatcher()
	cx.dispatchPendingFrames(func() bool { return true })
	collector := startBenchTraceScope(cx.foregroundJournalCollector())
	defer collector.end()
	cx.report.metrics.discardPending()
	var iterations uint64
	bencher.iter(cx.report.metrics, func() {
		iterations++
		dispatcher.runReadyMainTasksWith(func() bool { return true }, func() {
			cx.dispatchPendingFrames(func() bool { return true })
		})
		ok := cx.withWindow(view.EntityID(), func(window *Window, app *App) {
			AppContextUpdateEntity(app, view, func(state *V, ecx *Context[V]) struct{} {
				update(state, window, ecx)
				return struct{}{}
			})
		})
		if !ok {
			panic("cannot benchmark renderer for entity without a current window")
		}
		cx.dispatchPendingFrames(func() bool { return true })
	})

	events := collector.finish()
	cx.report.metrics.recordSample(iterations)
	cx.report.recordFrameTimings(events.frameEvents)
	cx.report.recordForegroundEvents(events.foregroundEvents())
	cx.replaceBencher(bencher)
}

// BenchRendererSession measures finite rendering sessions with fresh, untimed setup for each
// iteration. setup returns session state, its window and a shared stop flag. Loop turns pump
// at most the initial ready foreground task count, then call input with a zero-based turn
// number unless stopped; pending frames are delivered after each task poll and input
// callback. Once stopped, only scheduled frame callbacks run until the window is clean and its
// final changes have been submitted for presentation. Store true to stop. The session state
// is released (Release, when it has one) after timing stops.
//
// It panics if the window is removed or stopping and final presentation exceed timeout.
func (cx *BenchAppContext) BenchRendererSession[I any](timeout time.Duration, setup func(cx *BenchAppContext) (I, AnyWindowHandle, *atomic.Bool), input func(state *I, frame uint64, window *Window, cx *App)) {
	type session struct {
		state   I
		window  AnyWindowHandle
		stopped *atomic.Bool
	}
	bencher := cx.takeBencher("bench_renderer_session")
	setupContext := cx.clone()
	benchmarkContext := cx.clone()
	dispatcher := cx.dispatcher()
	report := cx.report
	report.metrics.discardPending()
	var iterations uint64

	benchIterBatchedRef(bencher, report.metrics,
		func() measuredTaskInput[session] {
			setupContext.Settle()
			state, window, stopped := setup(setupContext)
			setupContext.dispatchPendingFrames(func() bool { return true })
			return measuredTaskInput[session]{
				input:      session{state: state, window: window, stopped: stopped},
				traceScope: startBenchTraceScope(setupContext.foregroundJournalCollector()),
			}
		},
		func(measuredInput *measuredTaskInput[session]) *measuredTaskOutput[struct{}] {
			iterations++
			s := &measuredInput.input
			started := time.Now()
			checkDeadline := func() {
				if time.Since(started) >= timeout {
					panic(fmt.Sprintf("renderer session did not stop within %v with final changes presented", timeout))
				}
			}
			canPoll := func() bool {
				checkDeadline()
				return !s.stopped.Load()
			}
			isFinished := func(cx *BenchAppContext) bool {
				if !s.stopped.Load() {
					return false
				}
				app := cx.app.Borrow()
				defer app.Release()
				window, ok := app.Get().windows.get(s.window.WindowID())
				if !ok || window == nil {
					panic("renderer session window must remain open")
				}
				return !window.invalidator.isDirty() && !*window.needsPresent
			}
			dispatchFrames := func(cx *BenchAppContext) {
				checkDeadline()
				cx.dispatchPendingFrames(func() bool {
					checkDeadline()
					return !isFinished(cx)
				})
				checkDeadline()
			}
			var frame uint64
			for {
				checkDeadline()
				if isFinished(benchmarkContext) {
					break
				}
				if canPoll() {
					dispatcher.runReadyMainTasksWith(canPoll, func() {
						dispatchFrames(benchmarkContext)
					})
					if !canPoll() {
						continue
					}
					err := benchmarkContext.updateWindow(s.window, func(_ AnyView, window *Window, app *App) {
						if canPoll() {
							input(&s.state, frame, window, app)
						}
					})
					if err != nil {
						panic("renderer session window must remain open")
					}
				}
				dispatchFrames(benchmarkContext)
				frame++
			}
			traceScope := measuredInput.traceScope
			measuredInput.traceScope = nil
			return &measuredTaskOutput[struct{}]{
				traceScope: traceScope,
				report:     report,
			}
		},
		func(output *measuredTaskOutput[struct{}]) { output.Close() },
		func(input measuredTaskInput[session]) {
			if input.traceScope != nil {
				input.traceScope.end()
			}
			releaseBenchValue(input.input.state)
		},
	)
	report.metrics.recordSample(iterations)
	cx.replaceBencher(bencher)
}

type pendingBenchFrame struct {
	handle AnyWindowHandle
	window *TestWindow
}

func (cx *BenchAppContext) dispatchPendingFrames(shouldContinue func() bool) bool {
	var pending []pendingBenchFrame
	{
		app := cx.app.BorrowMut()
		for _, window := range app.windows.all() {
			if window == nil {
				continue
			}
			handle := window.handle
			platformWindow, ok := window.platformWindow.(*TestWindow)
			if !ok {
				app.Release()
				panic("benchmark platform window")
			}
			if platformWindow.FrameScheduled() {
				pending = append(pending, pendingBenchFrame{handle: handle, window: platformWindow.Clone()})
			}
		}
		app.Release()
	}

	dispatched := false
	for _, p := range pending {
		if !shouldContinue() {
			break
		}
		app := cx.app.Borrow()
		window, ok := app.Get().windows.get(p.handle.WindowID())
		isOpen := ok && window != nil
		app.Release()
		if isOpen {
			if p.window.SimulateScheduledFrame() {
				dispatched = true
			}
		}
	}
	return dispatched
}

// AddEmptyWindow adds an active window with an empty root view for benchmark setup.
// Activation is settled before returning so renderer measurements exercise foreground
// animation rather than the inactive-window frame throttle.
func (cx *BenchAppContext) AddEmptyWindow() *BenchWindowContext {
	bounds := cx.ReadR(func(app *App) Bounds[Pixels] {
		return BoundsMaximized(None[DisplayID](), app)
	})
	var window AnyWindowHandle
	func() {
		app := cx.app.BorrowMut()
		defer app.Release()
		options := DefaultWindowOptions()
		options.WindowBounds = Some(WindowBoundsWindowed(bounds))
		handle, err := app.OpenWindow(options, func(_ *Window, cx *App) Entity[Empty] {
			return cx.New(func(*Context[Empty]) *Empty { return &Empty{} })
		})
		if err != nil {
			panic(fmt.Sprintf("failed to open benchmark window: %v", err))
		}
		window = handle.AnyWindowHandle
		if err := app.updateWindow(window, func(_ AnyView, window *Window, _ *App) {
			window.ActivateWindow()
		}); err != nil {
			panic(fmt.Sprintf("failed to activate newly created benchmark window: %v", err))
		}
	}()

	cx.RunUntilIdle()
	return &BenchWindowContext{cx: cx.clone(), window: window}
}

func (cx *BenchAppContext) takeBencher(benchmarkKind string) *Bencher {
	bencher := cx.bencher.bencher
	if bencher == nil {
		panic(fmt.Sprintf("cannot start %s: benchmark measurement is already running", benchmarkKind))
	}
	cx.bencher.bencher = nil
	return bencher
}

func (cx *BenchAppContext) replaceBencher(bencher *Bencher) {
	previous := cx.bencher.bencher
	cx.bencher.bencher = bencher
	if previous != nil {
		panic("benchmark bencher was unexpectedly present after measurement")
	}
}

// Teardown runs GPUI benchmark teardown: it quits the app, cancels any timers still armed on
// the shared dispatcher and drains the work that cancellation unblocks so they cannot fire
// during a later benchmark. It assumes no other BenchAppContext is live on this goroutine.
func (cx *BenchAppContext) Teardown() {
	cx.RunUntilIdle()
	cx.Update(func(cx *App) {
		cx.Quit()
	})
	cx.RunUntilIdle()

	dispatcher := cx.dispatcher()
	cx.app = nil
	testWindowFrameWakerSchedulesFrame = false

	for range 100 {
		if dispatcher.CancelPendingTimers() == 0 {
			return
		}
		dispatcher.RunUntilIdle()
	}
	panic(fmt.Sprintf("benchmark teardown kept scheduling timers: %s", dispatcher.DebugState()))
}

func (cx *BenchAppContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.newEntity(entityType, build)
}

func (cx *BenchAppContext) reserveEntity(entityType reflect.Type) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.reserveEntity(entityType)
}

func (cx *BenchAppContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.insertEntity(slot, build)
}

func (cx *BenchAppContext) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.updateEntity(handle, update)
}

func (cx *BenchAppContext) asMut(AnyEntity) *App {
	panic("Cannot use as_mut with BenchAppContext. Call update() instead.")
}

func (cx *BenchAppContext) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readEntity(handle, read)
}

func (cx *BenchAppContext) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.updateWindow(window, update)
}

func (cx *BenchAppContext) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.withWindow(entityID, f)
}

func (cx *BenchAppContext) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	app := cx.app.Borrow()
	defer app.Release()
	return app.Get().readWindow(window, read)
}

func (cx *BenchAppContext) backgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

func (cx *BenchAppContext) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readGlobal(globalType, read)
}

// New creates a new entity.
func (cx *BenchAppContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later.
func (cx *BenchAppContext) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot reserved with ReserveEntity.
func (cx *BenchAppContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// ReadWindow reads the root view of a window.
func (cx *BenchAppContext) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on the background executor.
func (cx *BenchAppContext) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.bgExecutor.Spawn(future)
}

// ReadGlobal reads the global of type G.
func (cx *BenchAppContext) ReadGlobal[G, R any](callback func(global *G, cx *App) R) R {
	return AppContextReadGlobal(cx, callback)
}

// WithWindow runs f against the entity's current window, reporting false when it has none.
func (cx *BenchAppContext) WithWindow[R any](entityID EntityID, f func(window *Window, cx *App) R) (R, bool) {
	return AppContextWithWindow(cx, entityID, f)
}

// BenchWindowContext is a window-specific context for GPUI benchmarks. It provides access to a
// benchmark window without test-only helpers such as input simulation.
type BenchWindowContext struct {
	cx     *BenchAppContext
	window AnyWindowHandle
}

var _ VisualContext = (*BenchWindowContext)(nil)

// AppContext returns the underlying benchmark app context.
func (cx *BenchWindowContext) AppContext() *BenchAppContext {
	return cx.cx
}

// WindowHandle returns the window associated with this context.
func (cx *BenchWindowContext) WindowHandle() AnyWindowHandle {
	return cx.window
}

// RunUntilIdle runs queued foreground tasks and waits for in-flight background work to
// finish. Pending timers are not waited for.
func (cx *BenchWindowContext) RunUntilIdle() {
	cx.cx.RunUntilIdle()
}

// Update updates the benchmark window.
func (cx *BenchWindowContext) Update(update func(window *Window, cx *App)) {
	err := cx.cx.updateWindow(cx.window, func(_ AnyView, window *Window, app *App) {
		update(window, app)
	})
	if err != nil {
		panic("benchmark window was unexpectedly closed")
	}
}

// UpdateR is Update for a callback that returns a value.
func (cx *BenchWindowContext) UpdateR[R any](update func(window *Window, cx *App) R) R {
	var result R
	cx.Update(func(window *Window, app *App) { result = update(window, app) })
	return result
}

// SimulateResize resizes the window's content area as the platform does when the user drags
// its edge, through the window's resize callback. It must not be called from inside Update.
func (cx *BenchWindowContext) SimulateResize(size Size[Pixels]) {
	var platformWindow *TestWindow
	func() {
		app := cx.cx.app.BorrowMut()
		defer app.Release()
		window, ok := app.windows.get(cx.window.WindowID())
		if !ok || window == nil {
			panic("benchmark window was unexpectedly closed")
		}
		tw, ok := window.platformWindow.(*TestWindow)
		if !ok {
			panic("benchmark platform window")
		}
		platformWindow = tw.Clone()
	}()
	platformWindow.SimulateResize(size)
}

func (cx *BenchWindowContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	var entity AnyEntity
	if err := cx.cx.updateWindow(cx.window, func(_ AnyView, _ *Window, app *App) {
		entity = app.newEntity(entityType, build)
	}); err != nil {
		panic("benchmark window was unexpectedly closed")
	}
	return entity
}

func (cx *BenchWindowContext) reserveEntity(entityType reflect.Type) AnyEntity {
	return cx.cx.reserveEntity(entityType)
}

func (cx *BenchWindowContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	var entity AnyEntity
	if err := cx.cx.updateWindow(cx.window, func(_ AnyView, _ *Window, app *App) {
		entity = app.insertEntity(slot, build)
	}); err != nil {
		panic("benchmark window was unexpectedly closed")
	}
	return entity
}

func (cx *BenchWindowContext) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	cx.cx.updateEntity(handle, update)
}

func (cx *BenchWindowContext) asMut(handle AnyEntity) *App {
	return cx.cx.asMut(handle)
}

func (cx *BenchWindowContext) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	cx.cx.readEntity(handle, read)
}

func (cx *BenchWindowContext) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	return cx.cx.updateWindow(window, update)
}

func (cx *BenchWindowContext) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	return cx.cx.withWindow(entityID, f)
}

func (cx *BenchWindowContext) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	return cx.cx.readWindow(window, read)
}

func (cx *BenchWindowContext) backgroundExecutor() BackgroundExecutor {
	return cx.cx.backgroundExecutor()
}

func (cx *BenchWindowContext) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	cx.cx.readGlobal(globalType, read)
}

func (cx *BenchWindowContext) updateWindowContext(update func(window *Window, cx *App)) error {
	return cx.cx.updateWindow(cx.window, func(_ AnyView, window *Window, app *App) {
		update(window, app)
	})
}

// New creates a new entity in this context's window.
func (cx *BenchWindowContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later.
func (cx *BenchWindowContext) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a reserved slot in this context's window.
func (cx *BenchWindowContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// ReadWindow reads the root view of a window.
func (cx *BenchWindowContext) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on the background executor.
func (cx *BenchWindowContext) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.cx.BackgroundSpawn(future)
}

// ReadGlobal reads the global of type G.
func (cx *BenchWindowContext) ReadGlobal[G, R any](callback func(global *G, cx *App) R) R {
	return AppContextReadGlobal(cx, callback)
}

// UpdateWindowEntity updates an entity in its current window, failing when it has none.
func (cx *BenchWindowContext) UpdateWindowEntity[T, R any](entity Entity[T], update func(entity *T, window *Window, cx *Context[T]) R) (R, error) {
	var result R
	ok := cx.cx.withWindow(entity.EntityID(), func(window *Window, app *App) {
		result = AppContextUpdateEntity(app, entity, func(state *T, ecx *Context[T]) R {
			return update(state, window, ecx)
		})
	})
	if !ok {
		return result, fmt.Errorf("entity has no current window; use `update` instead of `update_in`")
	}
	return result, nil
}

// NewWindowEntity creates a new entity with access to the window.
func (cx *BenchWindowContext) NewWindowEntity[T any](buildEntity func(window *Window, cx *Context[T]) *T) (Entity[T], error) {
	return VisualContextNewWindowEntity(cx, buildEntity)
}

// ReplaceRootView replaces the root view of the window. *V must implement Render[V].
func (cx *BenchWindowContext) ReplaceRootView[V any](buildView func(window *Window, cx *Context[V]) *V) (Entity[V], error) {
	return VisualContextReplaceRootView(cx, buildView)
}

// Focus focuses the entity in the window. *V must implement Focusable.
func (cx *BenchWindowContext) Focus[V any](entity Entity[V]) error {
	return VisualContextFocus(cx, entity)
}
