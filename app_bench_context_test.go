package gpui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func secondaryValuesPerIteration(t *testing.T, counter *atomic.Uint64, benchmark func(cx *BenchAppContext, counter *atomic.Uint64)) []uint64 {
	t.Helper()
	report := DefaultBenchReport()
	var platform Platform
	onBenchThread(nil, func() {
		platform = BenchPlatform(nil, NewNoopTextSystem())
	})
	measurement := NewBenchMeasurement().WithSecondary("fake counter", counter.Load)

	testing.Benchmark(func(b *testing.B) {
		onBenchThread(b, func() {
			cx := NewBenchAppContextWithPlatformAndReport(
				platform,
				Some("secondary_values_per_iteration"),
				&Bencher{B: b, Measurement: measurement},
				report,
			)
			counter.Add(1_000_000)
			benchmark(cx, counter)
			cx.Teardown()
		})
	})

	values, ok := report.Metrics().Values("fake counter")
	if !ok {
		t.Fatal("the fake counter was recorded")
	}
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		result = append(result, uint64(value))
	}
	return result
}

func TestBenchContext_SecondaryMetricIsNormalizedPerIterationForBenchIter(t *testing.T) {
	values := secondaryValuesPerIteration(t, new(atomic.Uint64), func(cx *BenchAppContext, counter *atomic.Uint64) {
		cx.BenchIter(func(*BenchAppContext) {
			counter.Add(11)
		})
	})
	if len(values) == 0 {
		t.Fatal("expected samples")
	}
	for _, value := range values {
		if value != 11 {
			t.Fatalf("each sample's total should be divided by its iteration count: %v", values)
		}
	}
}

func TestBenchContext_SecondaryMetricIsAccumulatedAcrossPerIterationBatches(t *testing.T) {
	values := secondaryValuesPerIteration(t, new(atomic.Uint64), func(cx *BenchAppContext, counter *atomic.Uint64) {
		cx.BenchBatchedTask(
			func(*BenchAppContext) struct{} {
				counter.Add(1_000)
				return struct{}{}
			},
			func(_ *struct{}, cx *BenchAppContext) *Task[struct{}] {
				counter.Add(11)
				return cx.BackgroundSpawn(func(*scheduler.Scope) (struct{}, error) { return struct{}{}, nil })
			},
		)
	})
	if len(values) == 0 {
		t.Fatal("expected samples")
	}
	for _, value := range values {
		if value != 11 {
			t.Fatalf("PerIteration batches should sum to one per-iteration value: %v", values)
		}
	}
}

func skipWithoutProfiler(t *testing.T) {
	t.Helper()
	if !profilerFeatureEnabled {
		t.Skip("foreground task polls are journaled only with -tags gpui_profiler, which bench-support implies")
	}
}

func sleepTask(executor ForegroundExecutor, d time.Duration) *Task[struct{}] {
	return executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		time.Sleep(d)
		return struct{}{}, nil
	})
}

func TestBenchContext_ForegroundWorkReportsLongTaskWithoutWindowDraw(t *testing.T) {
	skipWithoutProfiler(t)
	onBenchThread(nil, func() {
		journal, journalGuard := installTestForegroundJournal(1024, 64)
		defer journalGuard.End()
		dispatcher := NewThreadedDispatcher()
		foregroundExecutor := NewForegroundExecutor(dispatcher)

		traceScope := startBenchTraceScope(journal.Collector())

		task := sleepTask(foregroundExecutor, 60*time.Millisecond)
		runTaskToCompletion(foregroundExecutor, dispatcher, task)

		events := traceScope.finish()
		if len(events.frameEvents) != 0 {
			t.Error("no window was involved, so no frame events should be recorded")
		}

		report := DefaultBenchReport()
		report.recordForegroundEvents(events.foregroundEvents())

		summary, ok := report.ForegroundWork()
		if !ok {
			t.Fatal("a long task poll should be reported even without a window draw")
		}
		if summary.Count < 1 {
			t.Error("expected at least one recorded item")
		}
		if summary.Max < 55*time.Millisecond {
			t.Errorf("expected the long poll's duration to be recorded, got %v", summary.Max)
		}
		if summary.Total < 55*time.Millisecond {
			t.Errorf("expected the long poll's duration to be included in the total, got %v", summary.Total)
		}
	})
}

func TestBenchContext_ForegroundWorkExcludesSetupBeforeTraceScopeStarts(t *testing.T) {
	skipWithoutProfiler(t)
	onBenchThread(nil, func() {
		journal, journalGuard := installTestForegroundJournal(1024, 64)
		defer journalGuard.End()
		dispatcher := NewThreadedDispatcher()
		foregroundExecutor := NewForegroundExecutor(dispatcher)

		setupTask := sleepTask(foregroundExecutor, 80*time.Millisecond)
		runTaskToCompletion(foregroundExecutor, dispatcher, setupTask)

		traceScope := startBenchTraceScope(journal.Collector())

		measuredTask := sleepTask(foregroundExecutor, 10*time.Millisecond)
		runTaskToCompletion(foregroundExecutor, dispatcher, measuredTask)

		events := traceScope.finish()
		report := DefaultBenchReport()
		report.recordForegroundEvents(events.foregroundEvents())

		summary, ok := report.ForegroundWork()
		if !ok {
			t.Fatal("the measured task's poll should be reported")
		}
		if summary.Max >= 40*time.Millisecond {
			t.Errorf("setup work's 80ms poll must not leak into the measured summary, got %v", summary.Max)
		}
		if summary.Total >= 40*time.Millisecond {
			t.Errorf("setup work's 80ms poll must not leak into the measured total, got %v", summary.Total)
		}
	})
}

func TestBenchContext_BenchTaskReportsLongTaskWithoutWindow(t *testing.T) {
	skipWithoutProfiler(t)
	var platform Platform
	onBenchThread(nil, func() {
		platform = BenchPlatform(nil, NewNoopTextSystem())
	})
	report := DefaultBenchReport()
	name := "bench_task_reports_long_task_without_window"

	testing.Benchmark(func(b *testing.B) {
		onBenchThread(b, func() {
			cx := NewBenchAppContextWithPlatformAndReport(platform, Some(name), &Bencher{B: b}, report)
			cx.BenchTask(func(cx *BenchAppContext) *Task[struct{}] {
				return sleepTask(cx.ForegroundExecutor(), 20*time.Millisecond)
			})
			cx.Teardown()
		})
	})

	summary, ok := report.ForegroundWork()
	if !ok {
		t.Fatal("bench_task should report foreground work with no window involved")
	}
	if summary.Max < 15*time.Millisecond {
		t.Errorf("expected a ~20ms task poll to be recorded, got %v", summary.Max)
	}
}

func TestBenchContext_TaskCompletionSupportsNonSendForegroundOutput(t *testing.T) {
	onBenchThread(nil, func() {
		dispatcher := NewThreadedDispatcher()
		backgroundExecutor := NewBackgroundExecutor(dispatcher)
		foregroundExecutor := NewForegroundExecutor(dispatcher)
		sender, receiver := scheduler.Oneshot[struct{}]()

		backgroundExecutor.Spawn(func(*scheduler.Scope) (struct{}, error) {
			if err := sender.Send(struct{}{}); err != nil {
				panic("foreground receiver should remain alive")
			}
			return struct{}{}, nil
		}).Detach()
		expectedOutput := new(int)
		*expectedOutput = 42
		taskOutput := expectedOutput
		task := foregroundExecutor.Spawn(func(s *scheduler.Scope) (*int, error) {
			if _, err := receiver.Recv(s); err != nil {
				panic("background task should send a value")
			}
			return taskOutput, nil
		})

		output := runTaskToCompletion(foregroundExecutor, dispatcher, task)
		if output != expectedOutput {
			t.Error("task runner should preserve non-Send foreground output")
		}
	})
}
