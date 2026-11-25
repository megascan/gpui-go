package gpui

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

// TestOption configures RunTest and PropertyTest, as the arguments of #[gpui::test] and
// #[gpui::property_test].
type TestOption struct {
	apply func(*testConfig)
}

type testConfig struct {
	seeds      []uint64
	iterations uint64
	retries    int
	onFailure  func()
	cases      uint32
}

func newTestConfig(opts []TestOption) testConfig {
	cfg := testConfig{iterations: 1, cases: defaultPropertyTestCases()}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return cfg
}

// Seed runs the test with one explicit seed (`seed = n`).
func Seed(seed uint64) TestOption {
	return TestOption{apply: func(c *testConfig) { c.seeds = []uint64{seed} }}
}

// Seeds runs the test once per explicit seed (`seeds(a, b, ...)`).
func Seeds(seeds ...uint64) TestOption {
	return TestOption{apply: func(c *testConfig) { c.seeds = append([]uint64(nil), seeds...) }}
}

// Iterations runs the test with seeds 0..n-1 (`iterations = n`).
func Iterations(n uint64) TestOption {
	return TestOption{apply: func(c *testConfig) { c.iterations = n }}
}

// Retries reruns a seed whose run panicked up to n times (`retries = n`). A failure reported
// through t.Error or t.Fatal cannot be undone, so it is not retried.
func Retries(n int) TestOption {
	return TestOption{apply: func(c *testConfig) { c.retries = n }}
}

// OnFailure calls f when the test fails after its retries (`on_failure = "path"`).
func OnFailure(f func()) TestOption {
	return TestOption{apply: func(c *testConfig) { c.onFailure = f }}
}

// Cases sets the number of cases PropertyTest generates (proptest's `config.cases`).
func Cases(n uint32) TestOption {
	return TestOption{apply: func(c *testConfig) { c.cases = n }}
}

// TestRun is one seeded run of a gpui test: the arguments #[gpui::test] would pass to the test
// function.
type TestRun struct {
	t          *testing.T
	seed       uint64
	dispatcher *TestDispatcher
	rng        *rand.Rand
	contexts   []*TestAppContext
}

func newTestRun(t *testing.T, dispatcher *TestDispatcher, seed uint64) *TestRun {
	return &TestRun{t: t, seed: seed, dispatcher: dispatcher}
}

// T returns the test.
func (tc *TestRun) T() *testing.T {
	return tc.t
}

// Seed returns the run's scheduler seed.
func (tc *TestRun) Seed() uint64 {
	return tc.seed
}

// Dispatcher returns the run's test dispatcher.
func (tc *TestRun) Dispatcher() *TestDispatcher {
	return tc.dispatcher
}

// Cx returns the run's first TestAppContext, building it on first use.
func (tc *TestRun) Cx() *TestAppContext {
	if len(tc.contexts) == 0 {
		return tc.NewContext()
	}
	return tc.contexts[0]
}

// NewContext builds another TestAppContext on the run's scheduler; a test may ask for as many
// as it needs. The run tears each one down at its end.
func (tc *TestRun) NewContext() *TestAppContext {
	name := ""
	if tc.t != nil {
		name = tc.t.Name()
	}
	cx := TestAppContextBuild(tc.dispatcher.Clone(), Some(name))
	tc.contexts = append(tc.contexts, cx)
	return cx
}

// Rand returns the run's random number generator, seeded with the run's seed (the StdRng
// argument; D-RNG).
func (tc *TestRun) Rand() *rand.Rand {
	if tc.rng == nil {
		tc.rng = rand.New(rand.NewPCG(tc.seed, 0))
	}
	return tc.rng
}

// BackgroundExecutor returns a background executor on the run's dispatcher.
func (tc *TestRun) BackgroundExecutor() BackgroundExecutor {
	return NewBackgroundExecutor(tc.dispatcher.Clone())
}

// BlockTest runs body as the async test function of an async #[gpui::test]: a coroutine on a
// foreground executor, driven until it completes.
func (tc *TestRun) BlockTest(body func(s *scheduler.Scope)) {
	NewForegroundExecutor(tc.dispatcher.Clone()).BlockTest(scheduler.Async(func(s *scheduler.Scope) struct{} {
		body(s)
		return struct{}{}
	}))
}

func (tc *TestRun) teardown() {
	for _, cx := range tc.contexts {
		cx.RunUntilParked()
		cx.Update(func(cx *App) {
			cx.BackgroundExecutor().ForbidParking()
			cx.Quit()
		})
		cx.RunUntilParked()
		cx.release()
	}
	tc.dispatcher.DrainTasks()
	for _, cx := range tc.contexts {
		reportAppLeaks(cx.app)
	}
	tc.contexts = nil
}

// RunTestOnce runs testFn once with a dispatcher seeded with seed and ends the scheduler's test.
// It is what PropertyTest runs per case.
func RunTestOnce[R any](seed uint64, testFn func(dispatcher *TestDispatcher) R) R {
	dispatcher := NewTestDispatcher(seed)
	s := dispatcher.Scheduler()
	result := testFn(dispatcher)
	s.EndTest()
	return result
}

// RunTest runs fn as a #[gpui::test]: once per seed from CalculateSeeds, with a fresh
// TestDispatcher per seed, retrying a panicking seed up to Retries times. After fn returns, each
// TestAppContext it asked for is run until parked, quit and dropped, and the dispatcher's
// remaining tasks are drained.
func RunTest(t *testing.T, fn func(tc *TestRun), opts ...TestOption) {
	t.Helper()
	cfg := newTestConfig(opts)
	seeds, isMultipleRuns := CalculateSeeds(cfg.iterations, cfg.seeds)
	for seed := range seeds {
		for attempt := 0; ; attempt++ {
			if isMultipleRuns {
				fmt.Fprintf(os.Stderr, "seed = %d\n", seed)
			}
			failure, panicked := cfg.attempt(t, seed, isMultipleRuns, fn)
			if !panicked {
				if t.Failed() {
					cfg.fail(seed, isMultipleRuns)
					return
				}
				break
			}
			if attempt < cfg.retries {
				fmt.Printf("attempt %d failed, retrying\n", attempt)
				continue
			}
			cfg.fail(seed, isMultipleRuns)
			panic(failure)
		}
	}
}

func (cfg testConfig) attempt(t *testing.T, seed uint64, isMultipleRuns bool, fn func(tc *TestRun)) (failure any, panicked bool) {
	exited := true
	defer func() {
		if r := recover(); r != nil {
			failure, panicked = r, true
			return
		}
		if exited {
			cfg.fail(seed, isMultipleRuns)
		}
	}()
	dispatcher := NewTestDispatcher(seed)
	s := dispatcher.Scheduler()
	tc := newTestRun(t, dispatcher, seed)
	fn(tc)
	tc.teardown()
	dispatcher.DrainTasks()
	s.EndTest()
	exited = false
	return nil, false
}

func (cfg testConfig) fail(seed uint64, isMultipleRuns bool) {
	if isMultipleRuns {
		fmt.Fprintf(os.Stderr, "failing seed: %d\n", seed)
		fmt.Fprintf(os.Stderr, "You can rerun from this seed by setting the environmental variable SEED to %d\n", seed)
	}
	if cfg.onFailure != nil {
		cfg.onFailure()
	}
}

func defaultPropertyTestCases() uint32 {
	if v, ok := os.LookupEnv("PROPTEST_CASES"); ok {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			return uint32(n)
		}
	}
	return 256
}

// SeedStrategy returns the scheduler seed of one property-test case: $SEED when set, otherwise
// a value drawn from the case generator, which does not shrink.
func SeedStrategy(rng *rand.Rand) uint64 {
	if v, ok := os.LookupEnv("SEED"); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		return n
	}
	return rng.Uint64()
}

// PropertyTestRNGSeed returns the fixed seed of the property-test case generator: $SEED when
// set and valid, otherwise 0. It is upstream's apply_seed_to_proptest_config.
func PropertyTestRNGSeed() uint64 {
	if v, ok := os.LookupEnv("SEED"); ok {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

// PropertyTest runs fn as a #[gpui::property_test]: Cases cases (proptest's default 256, or
// $PROPTEST_CASES) from a generator seeded with PropertyTestRNGSeed, each on a fresh dispatcher
// seeded with SeedStrategy. fn draws its own arguments from rng. Failing cases are not shrunk
// (D-PROPTEST).
func PropertyTest(t *testing.T, fn func(tc *TestRun, rng *rand.Rand), opts ...TestOption) {
	t.Helper()
	cfg := newTestConfig(opts)
	rng := rand.New(rand.NewPCG(PropertyTestRNGSeed(), 0))
	for range cfg.cases {
		seed := SeedStrategy(rng)
		done := false
		func() {
			defer func() {
				if !done {
					t.Logf("property test failed with scheduler seed %d", seed)
					if cfg.onFailure != nil {
						cfg.onFailure()
					}
				}
			}()
			RunTestOnce(seed, func(dispatcher *TestDispatcher) struct{} {
				tc := newTestRun(t, dispatcher, seed)
				fn(tc, rng)
				tc.teardown()
				return struct{}{}
			})
			done = true
		}()
		if t.Failed() {
			return
		}
	}
}

// Observation is a stream of the change notifications of an entity, returned by Observe.
type Observation[T any] struct {
	rx           *scheduler.ChanReceiver[T]
	subscription Subscription
}

// Next waits for the next notification. It reports false when the stream has ended.
func (o *Observation[T]) Next(cx scheduler.Awaiter) (T, bool) {
	return o.rx.Recv(cx)
}

// Poll polls the stream for its next notification.
func (o *Observation[T]) Poll(cx *scheduler.Context) (scheduler.Result[T], bool) {
	return o.rx.Poll(cx)
}

// Close drops the observation: it unsubscribes and closes the stream.
func (o *Observation[T]) Close() {
	o.subscription.Unsubscribe()
	o.rx.Close()
}

// Observe returns a stream of the change events of the given entity.
func Observe[T any](entity Entity[T], cx *TestAppContext) *Observation[struct{}] {
	tx, rx := scheduler.Chan[struct{}](0)
	var subscription Subscription
	cx.Update(func(cx *App) {
		subscription = cx.Observe(entity, func(Entity[T], *App) {
			_ = tx.TrySend(struct{}{})
		})
	})
	return &Observation[struct{}]{rx: rx, subscription: subscription}
}
