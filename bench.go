package gpui

import (
	"fmt"
	"math/rand/v2"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
)

// BenchOptions configures RunBench and its variants, as #[gpui::bench]'s arguments do.
type BenchOptions struct {
	// Fps is the frame rate whose frame budget the report counts overruns against; zero
	// selects the default of 120.
	Fps uint64
	// InputName names the inputs in sub-benchmark and report names; it requires inputs and
	// defaults to the benchmark's name.
	InputName string
	// Group names the report group; it requires inputs and defaults to the benchmark's name.
	Group string
	// Seeds are explicit seeds; they require an rng.
	Seeds []uint64
	// Iterations is how many consecutive seeds run; it requires an rng and defaults to 1.
	Iterations uint64
	// TextSystem shapes text; nil selects NoopTextSystem. Platform packages supply their
	// production text system here.
	TextSystem PlatformTextSystem
	// HeadlessRendererFactory supplies a renderer for benchmark windows; nil discards scenes.
	HeadlessRendererFactory func() (PlatformHeadlessRenderer, error)
}

type benchFunc[I any] func(input I, rng *rand.Rand, cx *BenchAppContext)

// RunBench runs fn as a #[gpui::bench] benchmark without inputs.
func RunBench(b *testing.B, opts BenchOptions, fn func(cx *BenchAppContext)) {
	b.Helper()
	runBench(b, opts, false, nil, false, func(_ struct{}, _ *rand.Rand, cx *BenchAppContext) { fn(cx) })
}

// RunBenchRng runs fn as a #[gpui::bench] benchmark taking a seeded rng, once per seed from
// CalculateSeeds, each seed its own sub-benchmark.
func RunBenchRng(b *testing.B, opts BenchOptions, fn func(rng *rand.Rand, cx *BenchAppContext)) {
	b.Helper()
	runBench(b, opts, true, nil, false, func(_ struct{}, rng *rand.Rand, cx *BenchAppContext) { fn(rng, cx) })
}

// RunBenchInputs runs fn as a #[gpui::bench(inputs = ...)] benchmark, each input its own
// sub-benchmark named InputName/input.
func RunBenchInputs[I any](b *testing.B, opts BenchOptions, inputs []I, fn func(input I, cx *BenchAppContext)) {
	b.Helper()
	runBench(b, opts, false, inputs, true, func(input I, _ *rand.Rand, cx *BenchAppContext) { fn(input, cx) })
}

// RunBenchInputsRng is RunBenchInputs for a benchmark taking a seeded rng.
func RunBenchInputsRng[I any](b *testing.B, opts BenchOptions, inputs []I, fn func(input I, rng *rand.Rand, cx *BenchAppContext)) {
	b.Helper()
	runBench(b, opts, true, inputs, true, fn)
}

func runBench[I any](b *testing.B, opts BenchOptions, takesRng bool, inputs []I, hasInputs bool, fn benchFunc[I]) {
	b.Helper()
	if opts.Iterations == 0 && takesRng {
		opts.Iterations = 1
	}
	if !takesRng && (opts.Seeds != nil || opts.Iterations != 0) {
		panic("#[gpui::bench] `seed`, `seeds`, and `iterations` require a `StdRng` parameter")
	}
	if !hasInputs {
		if opts.InputName != "" {
			panic("#[gpui::bench] `input_name` requires `inputs`")
		}
		if opts.Group != "" {
			panic("#[gpui::bench] `group` requires `inputs`")
		}
	}
	newReport := func() BenchReport {
		if opts.Fps != 0 {
			return BenchReportWithFps(opts.Fps)
		}
		return DefaultBenchReport()
	}
	textSystem := opts.TextSystem
	if textSystem == nil {
		textSystem = NewNoopTextSystem()
	}
	measurement := BenchMeasurementFromEnvOrExit()
	outerName := b.Name()

	routine := func(b *testing.B, report BenchReport, input I, seed uint64) {
		onBenchThread(b, func() {
			var rng *rand.Rand
			if takesRng {
				rng = rand.New(rand.NewPCG(seed, 0))
			}
			cx := NewBenchAppContextWithPlatformAndReport(
				BenchPlatform(opts.HeadlessRendererFactory, textSystem),
				Some(outerName),
				&Bencher{B: b, Measurement: measurement},
				report,
			)
			fn(input, rng, cx)
			cx.Teardown()
		})
	}
	forEachSeed := func(run func(seed uint64)) {
		if !takesRng {
			run(0)
			return
		}
		seeds, _ := CalculateSeeds(opts.Iterations, opts.Seeds)
		for seed := range seeds {
			run(seed)
		}
	}

	if hasInputs {
		inputName := opts.InputName
		if inputName == "" {
			inputName = outerName
		}
		groupName := opts.Group
		if groupName == "" {
			groupName = outerName
		}
		for _, input := range inputs {
			forEachSeed(func(seed uint64) {
				report := newReport()
				parameter := fmt.Sprint(input)
				if takesRng {
					parameter = fmt.Sprintf("%v/seed-%d", input, seed)
				}
				reportName := fmt.Sprintf("%s/%s/%s", groupName, inputName, parameter)
				b.Run(inputName+"/"+parameter, func(b *testing.B) {
					routine(b, report, input, seed)
				})
				report.Print(reportName)
			})
		}
		return
	}

	var zero I
	forEachSeed(func(seed uint64) {
		report := newReport()
		if takesRng {
			name := fmt.Sprintf("seed-%d", seed)
			b.Run(name, func(b *testing.B) {
				routine(b, report, zero, seed)
			})
			report.Print(outerName + "/" + name)
			return
		}
		routine(b, report, zero, seed)
		report.Print(outerName)
	})
}

type benchThreadJob struct {
	f    func()
	done chan benchThreadResult
}

type benchThreadResult struct {
	panicked  bool
	recovered any
	stack     []byte
	exited    bool
}

var benchThread struct {
	mu   sync.Mutex
	jobs chan benchThreadJob
}

func onBenchThread(b *testing.B, f func()) {
	benchThread.mu.Lock()
	if benchThread.jobs == nil {
		jobs := make(chan benchThreadJob)
		benchThread.jobs = jobs
		go runBenchThread(jobs)
	}
	jobs := benchThread.jobs
	benchThread.mu.Unlock()

	done := make(chan benchThreadResult, 1)
	jobs <- benchThreadJob{f: f, done: done}
	result := <-done
	if result.panicked {
		fmt.Fprintf(os.Stderr, "panic on the benchmark goroutine: %v\n%s\n", result.recovered, result.stack)
		panic(result.recovered)
	}
	if result.exited {
		benchThread.mu.Lock()
		if benchThread.jobs == jobs {
			benchThread.jobs = nil
		}
		benchThread.mu.Unlock()
		if b != nil && !b.Failed() && !b.Skipped() {
			b.FailNow()
		}
		runtime.Goexit()
	}
}

func runBenchThread(jobs chan benchThreadJob) {
	for job := range jobs {
		runBenchThreadJob(job)
	}
}

func runBenchThreadJob(job benchThreadJob) {
	returned := false
	defer func() {
		if returned {
			return
		}
		if r := recover(); r != nil {
			job.done <- benchThreadResult{panicked: true, recovered: r, stack: debug.Stack()}
			return
		}
		job.done <- benchThreadResult{exited: true}
	}()
	job.f()
	returned = true
	job.done <- benchThreadResult{}
}
