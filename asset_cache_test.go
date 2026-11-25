package gpui

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

type hashBytes []byte

func (b hashBytes) Hash(h *FxHasher) {
	h.Write(b)
}

func TestAssetCache_HashMatchesRustcHash(t *testing.T) {
	cases := []struct {
		name string
		got  uint64
		want uint64
	}{
		{"0u8", Hash(uint8(0)), 0},
		{"1u8", Hash(uint8(1)), 12157901119326311915},
		{"100u8", Hash(uint8(100)), 16751747135202103309},
		{"u8max", Hash(uint8(math.MaxUint8)), 1211781028898739645},
		{"u16max", Hash(uint16(math.MaxUint16)), 16279819243059860173},
		{"u32max", Hash(uint32(math.MaxUint32)), 7729994835221066939},
		{"u64max", Hash(uint64(math.MaxUint64)), 6288842954450348564},
		{"i8min", Hash(int8(math.MinInt8)), 6684841074112525780},
		{"i16max", Hash(int16(math.MaxInt16)), 2060959061933882993},
		{"i32min", Hash(int32(math.MinInt32)), 9943947977240134995},
		{"i64min", Hash(int64(math.MinInt64)), 33554432},
		{"i64max", Hash(int64(math.MaxInt64)), 6288842954483902996},
		{"empty", Hash(hashBytes{}), 17606491139363777937},
		{"zero", Hash(hashBytes{0}), 5448590020104574886},
		{"six zeros", Hash(hashBytes{0, 0, 0, 0, 0, 0}), 16766921560080789783},
		{"one", Hash(hashBytes{1}), 5922447956811044110},
		{"two", Hash(hashBytes{2}), 5229781508510959783},
		{"uwu", Hash(hashBytes("uwu")), 7168164714682931527},
		{"long", Hash(hashBytes("These are some bytes for testing rustc_hash.")), 2349210501944688211},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestAssetCache_HashSliceAndStringPrefixes(t *testing.T) {
	var slice FxHasher
	slice.WriteUsize(3)
	slice.Write([]byte("uwu"))
	if got := Hash([]byte("uwu")); got != slice.Finish() {
		t.Errorf("[]byte: got %d, want %d", got, slice.Finish())
	}
	var str FxHasher
	str.Write([]byte("uwu"))
	str.WriteU8(0xff)
	if got := Hash("uwu"); got != str.Finish() {
		t.Errorf("string: got %d, want %d", got, str.Finish())
	}
	if Hash(SharedString("uwu")) != Hash("uwu") {
		t.Error("SharedString hashes differently from its str")
	}
}

type inferenceAsset struct{}

type inferenceSource struct{ id int }

func (inferenceAsset) Load(source inferenceSource, cx *App) func(s *scheduler.Scope) scheduler.Result[int] {
	return func(*scheduler.Scope) scheduler.Result[int] { return scheduler.Result[int]{Value: source.id} }
}

var _ = func(cx *App) {
	var _, _ = cx.FetchAsset[inferenceAsset](inferenceSource{})
	var _ bool = cx.HasAsset[inferenceAsset](inferenceSource{})
	cx.RemoveAsset[inferenceAsset](inferenceSource{})
	var _, _ = cx.FetchAsset[AssetLogger[inferenceAsset, inferenceSource, int]](inferenceSource{})
}

type testLoadResult struct {
	value int
	err   string
}

func okLoad(value int) testLoadResult {
	return testLoadResult{value: value}
}

func errLoad(err string) testLoadResult {
	return testLoadResult{err: err}
}

func TestAssetCache_CachedLoadCachesSuccessAndError(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		successfulLoad := cx.UpdateR(func(cx *App) *cachedLoad[testLoadResult] {
			return newCachedLoad(func(*scheduler.Scope) testLoadResult { return okLoad(42) }, cx)
		})
		failedLoad := cx.UpdateR(func(cx *App) *cachedLoad[testLoadResult] {
			return newCachedLoad(func(*scheduler.Scope) testLoadResult { return errLoad("load failed") }, cx)
		})

		if _, ok := successfulLoad.get(); ok {
			t.Fatal("successful load is ready before running")
		}
		if _, ok := failedLoad.get(); ok {
			t.Fatal("failed load is ready before running")
		}

		cx.RunUntilParked()

		for range 2 {
			if got, ok := successfulLoad.get(); !ok || got != okLoad(42) {
				t.Fatalf("successfulLoad.get() = (%v, %v)", got, ok)
			}
		}
		for range 2 {
			if got, ok := failedLoad.get(); !ok || got != errLoad("load failed") {
				t.Fatalf("failedLoad.get() = (%v, %v)", got, ok)
			}
		}
	})
}

func TestAssetCache_CompletedLoadWithoutSubscribersIsCachedAndDeduplicated(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		sender, receiver := scheduler.Oneshot[testLoadResult]()
		source := newTestAssetSource(1, receiver)

		for range 2 {
			if got, ok := fetchTestAsset(cx, source); ok {
				t.Fatalf("fetch before completion = %v", got)
			}
		}
		if n := source.loadCount.Load(); n != 1 {
			t.Fatalf("load count = %d, want 1", n)
		}

		if err := sender.Send(okLoad(42)); err != nil {
			t.Fatalf("send: %v", err)
		}
		cx.RunUntilParked()

		for range 2 {
			if got, ok := fetchTestAsset(cx, source); !ok || got != okLoad(42) {
				t.Fatalf("fetch after completion = (%v, %v)", got, ok)
			}
		}
		if n := source.loadCount.Load(); n != 1 {
			t.Fatalf("load count = %d, want 1", n)
		}
	})
}

func TestAssetCache_LoadCompletionNotifiesEachObservingEntityOnce(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		sender, receiver := scheduler.Oneshot[int]()
		load := cx.UpdateR(func(cx *App) *cachedLoad[int] {
			return newCachedLoad(func(s *scheduler.Scope) int {
				v, err := receiver.Recv(s)
				if err != nil {
					panic("test sends a result")
				}
				return v
			}, cx)
		})
		firstNotificationCount := 0
		secondNotificationCount := 0
		var firstEntity, secondEntity Entity[struct{}]
		cx.Update(func(cx *App) {
			firstEntity = cx.New(func(*Context[struct{}]) *struct{} { return &struct{}{} })
			secondEntity = cx.New(func(*Context[struct{}]) *struct{} { return &struct{}{} })
			cx.Observe(firstEntity, func(Entity[struct{}], *App) { firstNotificationCount++ }).Detach()
			cx.Observe(secondEntity, func(Entity[struct{}], *App) { secondNotificationCount++ }).Detach()
		})
		defer firstEntity.Release()
		defer secondEntity.Release()

		if _, ok := load.useBy(firstEntity.EntityID()); ok {
			t.Fatal("load ready before completion")
		}
		if _, ok := load.useBy(firstEntity.EntityID()); ok {
			t.Fatal("load ready before completion")
		}
		if _, ok := load.useBy(secondEntity.EntityID()); ok {
			t.Fatal("load ready before completion")
		}

		if err := sender.Send(42); err != nil {
			t.Fatalf("send: %v", err)
		}
		cx.RunUntilParked()

		if firstNotificationCount != 1 {
			t.Fatalf("first notification count = %d, want 1", firstNotificationCount)
		}
		if secondNotificationCount != 1 {
			t.Fatalf("second notification count = %d, want 1", secondNotificationCount)
		}
		if got, ok := load.useBy(firstEntity.EntityID()); !ok || got != 42 {
			t.Fatalf("useBy(first) = (%v, %v)", got, ok)
		}
		if got, ok := load.useBy(secondEntity.EntityID()); !ok || got != 42 {
			t.Fatalf("useBy(second) = (%v, %v)", got, ok)
		}
	})
}

func TestAssetCache_RemovingPendingAssetCancelsItAndReplacementIgnoresStaleCompletion(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		staleSender, staleReceiver := scheduler.Oneshot[testLoadResult]()
		replacementSender, replacementReceiver := scheduler.Oneshot[testLoadResult]()
		source := newTestAssetSource(2, staleReceiver, replacementReceiver)

		if got, ok := fetchTestAsset(cx, source); ok {
			t.Fatalf("fetch before completion = %v", got)
		}
		if n := source.loadCount.Load(); n != 1 {
			t.Fatalf("load count = %d, want 1", n)
		}
		cx.RunUntilParked()

		cx.Update(func(cx *App) { cx.RemoveAsset[testAsset](source) })
		if cx.UpdateR(func(cx *App) bool { return cx.HasAsset[testAsset](source) }) {
			t.Fatal("asset still cached after RemoveAsset")
		}
		cx.RunUntilParked()
		if err := staleSender.Send(okLoad(1)); err == nil {
			t.Fatal("stale load was not cancelled")
		}

		if got, ok := fetchTestAsset(cx, source); ok {
			t.Fatalf("fetch of replacement before completion = %v", got)
		}
		if n := source.loadCount.Load(); n != 2 {
			t.Fatalf("load count = %d, want 2", n)
		}

		if err := replacementSender.Send(errLoad("replacement failed")); err != nil {
			t.Fatalf("send: %v", err)
		}
		cx.RunUntilParked()

		if got, ok := fetchTestAsset(cx, source); !ok || got != errLoad("replacement failed") {
			t.Fatalf("fetch after replacement = (%v, %v)", got, ok)
		}
		if n := source.loadCount.Load(); n != 2 {
			t.Fatalf("load count = %d, want 2", n)
		}
	})
}

func TestAssetCache_EvictionAfterWorkerCompletionDoesNotPublishIntoReplacement(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		sender, receiver := scheduler.Oneshot[testLoadResult]()
		replacementSender, replacementReceiver := scheduler.Oneshot[testLoadResult]()
		source := newTestAssetSource(3, receiver, replacementReceiver)

		if got, ok := fetchTestAsset(cx, source); ok {
			t.Fatalf("fetch before completion = %v", got)
		}
		if err := sender.Send(okLoad(1)); err != nil {
			t.Fatalf("send: %v", err)
		}
		for source.completionCount.Load() == 0 {
			if !cx.bgExecutor.Tick() {
				t.Fatal("tick found nothing to run before the worker completed")
			}
		}
		if got, ok := fetchTestAsset(cx, source); ok {
			t.Fatalf("fetch before publication = %v", got)
		}

		cx.Update(func(cx *App) {
			cx.RemoveAsset[testAsset](source)
			if got, ok := cx.FetchAsset[testAsset](source); ok {
				t.Fatalf("fetch of replacement inside the eviction update = %v", got)
			}
		})
		cx.RunUntilParked()
		if got, ok := fetchTestAsset(cx, source); ok {
			t.Fatalf("stale completion was published into the replacement: %v", got)
		}

		if err := replacementSender.Send(okLoad(2)); err != nil {
			t.Fatalf("send: %v", err)
		}
		cx.RunUntilParked()
		if got, ok := fetchTestAsset(cx, source); !ok || got != okLoad(2) {
			t.Fatalf("fetch after replacement = (%v, %v)", got, ok)
		}
		if n := source.loadCount.Load(); n != 2 {
			t.Fatalf("load count = %d, want 2", n)
		}
	})
}

func fetchTestAsset(cx *TestAppContext, source testAssetSource) (testLoadResult, bool) {
	type fetched struct {
		result testLoadResult
		ok     bool
	}
	f := cx.UpdateR(func(cx *App) fetched {
		result, ok := cx.FetchAsset[testAsset](source)
		return fetched{result, ok}
	})
	return f.result, f.ok
}

type testAsset struct{}

type testAssetSource struct {
	id              int
	loadCount       *atomic.Int64
	completionCount *atomic.Int64
	mu              *sync.Mutex
	receivers       *[]*scheduler.OneshotReceiver[testLoadResult]
}

func newTestAssetSource(id int, receivers ...*scheduler.OneshotReceiver[testLoadResult]) testAssetSource {
	queue := append([]*scheduler.OneshotReceiver[testLoadResult](nil), receivers...)
	return testAssetSource{
		id:              id,
		loadCount:       new(atomic.Int64),
		completionCount: new(atomic.Int64),
		mu:              new(sync.Mutex),
		receivers:       &queue,
	}
}

func (s testAssetSource) Hash(h *FxHasher) {
	h.WriteUsize(uint64(s.id))
}

func (testAsset) Load(source testAssetSource, _ *App) func(s *scheduler.Scope) testLoadResult {
	source.loadCount.Add(1)
	source.mu.Lock()
	if len(*source.receivers) == 0 {
		source.mu.Unlock()
		panic("each test load should have a receiver")
	}
	receiver := (*source.receivers)[0]
	*source.receivers = (*source.receivers)[1:]
	source.mu.Unlock()
	return func(s *scheduler.Scope) testLoadResult {
		defer receiver.Close()
		result, err := receiver.Recv(s)
		if err != nil {
			result = errLoad("cancelled")
		}
		source.completionCount.Add(1)
		return result
	}
}
