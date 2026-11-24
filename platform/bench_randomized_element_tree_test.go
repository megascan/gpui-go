package platform_test

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"runtime/metrics"
	"slices"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

type treeFamily struct {
	name         string
	bounds       gpui.RandomizedElementTreeBounds
	mixedChanges int
}

func (f treeFamily) sample(rng *rand.Rand) treeInput {
	nameHash := uint64(0xcbf2_9ce4_8422_2325)
	for _, b := range []byte(f.name) {
		nameHash = (nameHash ^ uint64(b)) * 0x0000_0100_0000_01b3
	}
	familyRng := rand.New(rand.NewPCG(rng.Uint64()^nameHash, 0))
	return treeInput{family: f.name, config: f.bounds.Sample(familyRng)}
}

func (f treeFamily) String() string {
	return f.name
}

type treeInput struct {
	family string
	config gpui.RandomizedElementTreeConfig
}

func (i treeInput) String() string {
	var topology string
	switch i.config.Topology() {
	case gpui.RandomizedElementTreeTopologyWide:
		topology = "wide"
	case gpui.RandomizedElementTreeTopologyDeep:
		topology = "deep"
	case gpui.RandomizedElementTreeTopologyMixed:
		topology = "mixed"
	case gpui.RandomizedElementTreeTopologyNarrow:
		topology = "narrow"
	}
	return fmt.Sprintf("%s tree: %s, %d elements, %d%% entities, %d%% handlers",
		i.family,
		topology,
		i.config.ElementCount(),
		int(math.Round(i.config.EntityDensity()*100.0)),
		int(math.Round(i.config.HandlerDensity()*100.0)),
	)
}

const tallMaxElements = 256

const minEntityDensity = 0.05

func treeFamilies() []treeFamily {
	var tall []gpui.RandomizedElementTreeTopology
	for _, topology := range gpui.RandomizedElementTreeTopologyAll {
		if topology.IsTall() {
			tall = append(tall, topology)
		}
	}
	return []treeFamily{
		{
			name:         "any",
			mixedChanges: 192,
			bounds: gpui.NewRandomizedElementTreeBounds(32, tallMaxElements).
				WithEntityDensity(minEntityDensity, 0.5).
				WithHandlerDensity(0.1, 0.4),
		},
		{
			name:         "wide",
			mixedChanges: 16,
			bounds: gpui.NewRandomizedElementTreeBounds(128, 2048).
				WithTopologies(gpui.RandomizedElementTreeTopologyWide).
				WithEntityDensity(minEntityDensity, 0.25).
				WithHandlerDensity(0.1, 0.4),
		},
		{
			name:         "tall",
			mixedChanges: 512,
			bounds: gpui.NewRandomizedElementTreeBounds(32, tallMaxElements).
				WithTopologies(tall...).
				WithEntityDensity(minEntityDensity, 0.25).
				WithHandlerDensity(0.1, 0.4),
		},
		{
			name:         "mixed",
			mixedChanges: 192,
			bounds: gpui.NewRandomizedElementTreeBounds(128, 1024).
				WithTopologies(gpui.RandomizedElementTreeTopologyMixed).
				WithEntityDensity(minEntityDensity, 0.25).
				WithHandlerDensity(0.1, 0.4),
		},
		{
			name:         "dense-entities",
			mixedChanges: 64,
			bounds: gpui.NewRandomizedElementTreeBounds(128, 1024).
				WithTopologies(gpui.RandomizedElementTreeTopologyWide, gpui.RandomizedElementTreeTopologyMixed).
				WithEntityDensity(0.75, 1.0).
				WithHandlerDensity(0.1, 0.4),
		},
	}
}

func treeBenchOptions(b *testing.B, group string) gpui.BenchOptions {
	return platformBenchOptions(b, gpui.BenchOptions{InputName: "tree", Iterations: 2, Group: group, Fps: 120})
}

type allocationStats struct {
	allocations uint64
	liveBytes   uint64
}

func readAllocationStats() allocationStats {
	samples := []metrics.Sample{
		{Name: "/gc/heap/allocs:objects"},
		{Name: "/memory/classes/heap/objects:bytes"},
	}
	metrics.Read(samples)
	var stats allocationStats
	if samples[0].Value.Kind() == metrics.KindUint64 {
		stats.allocations = samples[0].Value.Uint64()
	}
	if samples[1].Value.Kind() == metrics.KindUint64 {
		stats.liveBytes = samples[1].Value.Uint64()
	}
	return stats
}

type treeMutateFunc func(tree *gpui.RandomizedElementTree, cx *gpui.Context[gpui.RandomizedElementTree])

type treeWindowMutateFunc func(tree *gpui.RandomizedElementTree, window *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree])

func measureTree(group string, input treeInput, cx *gpui.BenchAppContext, mutate treeMutateFunc) {
	measureTreeInWindow(group, input, cx, func(tree *gpui.RandomizedElementTree, _ *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) {
		mutate(tree, cx)
	})
}

func measureTreeInWindow(group string, input treeInput, cx *gpui.BenchAppContext, mutate treeWindowMutateFunc) {
	measureTreePrepared(group, input, cx,
		func(*gpui.RandomizedElementTree, *gpui.Window) struct{} { return struct{}{} },
		func(_ *struct{}, tree *gpui.RandomizedElementTree, window *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) {
			mutate(tree, window, cx)
		},
	)
}

func measureTreePrepared[State any](
	group string,
	input treeInput,
	cx *gpui.BenchAppContext,
	prepare func(tree *gpui.RandomizedElementTree, window *gpui.Window) State,
	mutate func(state *State, tree *gpui.RandomizedElementTree, window *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]),
) {
	config := input.config
	beforeWindow := readAllocationStats()
	window := cx.AddEmptyWindow()
	tree := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[gpui.RandomizedElementTree] {
		return w.ReplaceRoot(app, func(_ *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) *gpui.RandomizedElementTree {
			return gpui.NewRandomizedElementTreeWithConfig(config, cx)
		})
	})
	defer tree.Release()
	cx.RunUntilIdle()
	window.Update(func(w *gpui.Window, _ *gpui.App) { w.Refresh() })
	cx.RunUntilIdle()
	describeOnce(input, beforeWindow, readAllocationStats())
	state := window.UpdateR(func(w *gpui.Window, app *gpui.App) State {
		var state State
		tree.Update(app, func(t *gpui.RandomizedElementTree, _ *gpui.Context[gpui.RandomizedElementTree]) {
			state = prepare(t, w)
		})
		return state
	})

	var summary seedTotals
	var previousStarted time.Time
	var previousStats allocationStats
	hasPrevious := false
	cx.BenchRenderer(tree, func(t *gpui.RandomizedElementTree, w *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) {
		now := time.Now()
		stats := readAllocationStats()
		if hasPrevious {
			summary.elapsed += now.Sub(previousStarted)
			summary.iterations++
			summary.allocations += stats.allocations - previousStats.allocations
		}
		previousStarted, previousStats, hasPrevious = now, stats, true

		mutate(&state, t, w, cx)
	})
	window.Update(func(w *gpui.Window, _ *gpui.App) { reportSceneStats(group, input.family, w) })
	recordForSummary(group, input, summary)
}

func reportSceneStats(group, input string, window *gpui.Window) {
	if os.Getenv("GPUI_BENCH_SCENE_STATS") == "1" {
		scene := window.RenderedScene()
		fmt.Fprintf(os.Stderr, "scene stats %s/%s: batches=%d, paths=%d, surfaces=%d\n",
			group, input, sceneBatchCount(scene), len(scene.Paths), len(scene.Surfaces))
	}
}

var lastDescribed gpui.Option[gpui.RandomizedElementTreeConfig]

func describeOnce(input treeInput, beforeWindow, afterFirstFrame allocationStats) {
	isNew := lastDescribed != gpui.Some(input.config)
	lastDescribed = gpui.Some(input.config)
	if !isNew {
		return
	}
	fmt.Fprintf(os.Stderr, "%v; window holds %.1f KB after the first frame\n",
		input, float64(int64(afterFirstFrame.liveBytes)-int64(beforeWindow.liveBytes))/1024.)
}

type seedTotals struct {
	elapsed     time.Duration
	iterations  uint64
	allocations uint64
}

type summarySeed struct {
	config gpui.RandomizedElementTreeConfig
	totals seedTotals
}

type summaryEntry struct {
	group  string
	family string
	seeds  []summarySeed
}

var treeSummary []summaryEntry

func recordForSummary(group string, input treeInput, totals seedTotals) {
	index := slices.IndexFunc(treeSummary, func(entry summaryEntry) bool {
		return entry.group == group && entry.family == input.family
	})
	if index < 0 {
		treeSummary = append(treeSummary, summaryEntry{group: group, family: input.family})
		index = len(treeSummary) - 1
	}
	seeds := &treeSummary[index].seeds
	for i := range *seeds {
		seed := &(*seeds)[i]
		if seed.config == input.config {
			seed.totals.elapsed += totals.elapsed
			seed.totals.iterations += totals.iterations
			seed.totals.allocations += totals.allocations
			return
		}
	}
	*seeds = append(*seeds, summarySeed{config: input.config, totals: totals})
}

func geometricMean(values []float64) float64 {
	logSum := 0.0
	for _, value := range values {
		logSum += math.Log(max(value, math.SmallestNonzeroFloat64))
	}
	return math.Exp(logSum / float64(max(len(values), 1)))
}

func BenchmarkRandomizedElementTree_FullRefresh(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/full refresh"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureTreeInWindow("full refresh", input, cx, func(_ *gpui.RandomizedElementTree, w *gpui.Window, _ *gpui.Context[gpui.RandomizedElementTree]) {
			w.Refresh()
		})
	})
}

func sampleWithoutPaths(family treeFamily, rng *rand.Rand) treeInput {
	input := family.sample(rng)
	input.config = input.config.WithPaths(false)
	return input
}

func BenchmarkRandomizedElementTree_PathlessFullRefresh(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "PathlessTree/full refresh"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := sampleWithoutPaths(family, rng)
		measureTreeInWindow("pathless full refresh", input, cx, func(_ *gpui.RandomizedElementTree, w *gpui.Window, _ *gpui.Context[gpui.RandomizedElementTree]) {
			w.Refresh()
		})
	})
}

func BenchmarkRandomizedElementTree_PathlessElementStyle(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "PathlessTree/element style"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := sampleWithoutPaths(family, rng)
		measureElementStyle("pathless element style", input, cx)
	})
}

func BenchmarkRandomizedElementTree_Unchanged(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/unchanged"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureTree("unchanged", input, cx, func(_ *gpui.RandomizedElementTree, cx *gpui.Context[gpui.RandomizedElementTree]) {
			cx.Notify()
		})
	})
}

func measureElementStyle(group string, input treeInput, cx *gpui.BenchAppContext) {
	measureTreePrepared(group, input, cx,
		func(tree *gpui.RandomizedElementTree, _ *gpui.Window) gpui.Option[uint64] {
			if id, ok := tree.RandomElementID(); ok {
				return gpui.Some(id)
			}
			return gpui.None[uint64]()
		},
		func(elementID *gpui.Option[uint64], tree *gpui.RandomizedElementTree, _ *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) {
			if id, ok := elementID.Get(); ok {
				tree.AlternateElementColor(id, cx)
			}
		},
	)
}

func BenchmarkRandomizedElementTree_ElementStyle(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/element style"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureElementStyle("element style", input, cx)
	})
}

func applyKind(kind gpui.RandomizedElementTreeMutationKind) treeMutateFunc {
	return func(tree *gpui.RandomizedElementTree, cx *gpui.Context[gpui.RandomizedElementTree]) {
		tree.ApplyMutation(kind, cx)
	}
}

func alternateInsertRemove() treeMutateFunc {
	insert := true
	return func(tree *gpui.RandomizedElementTree, cx *gpui.Context[gpui.RandomizedElementTree]) {
		kind := gpui.RandomizedElementTreeMutationKindRemoveLeaf
		if insert {
			kind = gpui.RandomizedElementTreeMutationKindInsert
		}
		tree.ApplyMutation(kind, cx)
		insert = !insert
	}
}

func BenchmarkRandomizedElementTree_LeafBounds(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/leaf bounds"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureTree("leaf bounds", input, cx, applyKind(gpui.RandomizedElementTreeMutationKindChildBounds))
	})
}

func BenchmarkRandomizedElementTree_RootLayout(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/root layout"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureTree("root layout", input, cx, applyKind(gpui.RandomizedElementTreeMutationKindRootBounds))
	})
}

func BenchmarkRandomizedElementTree_InsertRemove(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/insert-remove"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureTree("insert-remove", input, cx, alternateInsertRemove())
	})
}

func BenchmarkRandomizedElementTree_Reorder(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/reorder"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureTree("reorder", input, cx, applyKind(gpui.RandomizedElementTreeMutationKindReorder))
	})
}

func hoverTargets(window *gpui.Window) []gpui.Point[gpui.Pixels] {
	scaleFactor := window.ScaleFactor()
	var targets []gpui.Point[gpui.Pixels]
	for _, quad := range window.RenderedScene().Quads {
		visible := quad.Bounds.Intersect(quad.ContentMask.Bounds)
		minimum := gpui.ScaledPixels(4. * scaleFactor)
		if visible.Size.Width < minimum || visible.Size.Height < minimum {
			continue
		}
		center := visible.Center()
		targets = append(targets, gpui.NewPoint(
			gpui.Px(center.X.AsF32()/scaleFactor),
			gpui.Px(center.Y.AsF32()/scaleFactor),
		))
	}
	slices.SortFunc(targets, func(a, b gpui.Point[gpui.Pixels]) int {
		return cmp.Or(cmp.Compare(a.Y.AsF32(), b.Y.AsF32()), cmp.Compare(a.X.AsF32(), b.X.AsF32()))
	})
	return slices.Compact(targets)
}

func measureHover(group string, input treeInput, cx *gpui.BenchAppContext) {
	frame := 0
	measureTreePrepared(group, input, cx,
		func(_ *gpui.RandomizedElementTree, window *gpui.Window) []gpui.Point[gpui.Pixels] {
			return hoverTargets(window)
		},
		func(targets *[]gpui.Point[gpui.Pixels], _ *gpui.RandomizedElementTree, window *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) {
			if index := frame % max(len(*targets), 1); index < len(*targets) {
				window.DispatchEvent(gpui.MouseMoveEvent{
					Position:      (*targets)[index],
					Modifiers:     gpui.Modifiers{},
					PressedButton: gpui.None[gpui.MouseButton](),
				}, cx.App)
			}
			frame++
		},
	)
}

func BenchmarkRandomizedElementTree_Hover(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/hover"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		input.config = input.config.WithHoverStyles(true)
		measureHover("hover", input, cx)
	})
}

type mixedWorkload struct {
	rng         *rand.Rand
	inputRng    *rand.Rand
	initialSize int
	size        int
	rootKinds   []gpui.RandomizedElementTreeMutationKind
}

func newMixedWorkload(seed uint64, initialSize int, cachedViews bool) *mixedWorkload {
	rootKinds := []gpui.RandomizedElementTreeMutationKind{
		gpui.RandomizedElementTreeMutationKindRootColor,
		gpui.RandomizedElementTreeMutationKindRootOpacity,
		gpui.RandomizedElementTreeMutationKindRootBorder,
		gpui.RandomizedElementTreeMutationKindRootBounds,
		gpui.RandomizedElementTreeMutationKindRootClipping,
	}
	if !cachedViews {
		rootKinds = append(rootKinds, gpui.RandomizedElementTreeMutationKindRootVisibility)
	}
	return &mixedWorkload{
		rng:         rand.New(rand.NewPCG(seed^0x6d69_7865_6421, 0)),
		inputRng:    rand.New(rand.NewPCG(seed^0x696e_7075_7473, 0)),
		initialSize: initialSize,
		size:        initialSize,
		rootKinds:   rootKinds,
	}
}

func (w *mixedWorkload) pointerPosition() gpui.Point[gpui.Pixels] {
	return gpui.NewPoint(
		gpui.Px(randomF32Range(w.inputRng, 0.0, 1000.0)),
		gpui.Px(randomF32Range(w.inputRng, 0.0, 800.0)),
	)
}

func (w *mixedWorkload) apply(tree *gpui.RandomizedElementTree, kind gpui.RandomizedElementTreeMutationKind, cx *gpui.Context[gpui.RandomizedElementTree]) {
	switch mutation := tree.ApplyMutation(kind, cx).(type) {
	case gpui.RandomizedElementTreeMutationInserted:
		w.size++
	case gpui.RandomizedElementTreeMutationRemoved:
		w.size = max(w.size-mutation.RemovedElementCount, 0)
	}
}

func (w *mixedWorkload) change(tree *gpui.RandomizedElementTree, window *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) gpui.Option[gpui.Size[gpui.Pixels]] {
	roll := randomIntRange(w.rng, 0, 100)
	switch {
	case roll < 30:
		kinds := [3]gpui.RandomizedElementTreeMutationKind{
			gpui.RandomizedElementTreeMutationKindChildColor,
			gpui.RandomizedElementTreeMutationKindChildOpacity,
			gpui.RandomizedElementTreeMutationKindChildBorder,
		}
		w.apply(tree, kinds[randomIntRange(w.rng, 0, 3)], cx)
	case roll < 40:
		w.apply(tree, gpui.RandomizedElementTreeMutationKindChildBounds, cx)
	case roll < 45:
		w.apply(tree, gpui.RandomizedElementTreeMutationKindChildClipping, cx)
	case roll < 48:
		w.apply(tree, gpui.RandomizedElementTreeMutationKindChildVisibility, cx)
	case roll < 52:
		w.apply(tree, w.rootKinds[randomIntRange(w.rng, 0, len(w.rootKinds))], cx)
	case roll < 64:
		var kind gpui.RandomizedElementTreeMutationKind
		switch {
		case w.size < w.initialSize:
			kind = gpui.RandomizedElementTreeMutationKindInsert
		case w.rng.Float64() < 0.5:
			kind = gpui.RandomizedElementTreeMutationKindRemove
		default:
			kind = gpui.RandomizedElementTreeMutationKindRemoveLeaf
		}
		w.apply(tree, kind, cx)
	case roll < 70:
		w.apply(tree, gpui.RandomizedElementTreeMutationKindReorder, cx)
	case roll < 74:
		count := max(w.size/20, 1)
		tree.RecolorChildren(count, gpui.ChangeLocalitySpread, cx)
	case roll < 84:
		position := w.pointerPosition()
		window.DispatchEvent(gpui.MouseMoveEvent{
			Position:      position,
			Modifiers:     gpui.Modifiers{},
			PressedButton: gpui.None[gpui.MouseButton](),
		}, cx.App)
	case roll < 92:
		position := w.pointerPosition()
		distance := gpui.Px(randomF32Range(w.inputRng, -160.0, 160.0))
		var delta gpui.Point[gpui.Pixels]
		if w.inputRng.Float64() < 0.8 {
			delta = gpui.NewPoint(gpui.Px(0.), distance)
		} else {
			delta = gpui.NewPoint(distance, gpui.Px(0.))
		}
		window.DispatchEvent(gpui.ScrollEvent{
			Position: position,
			Delta:    gpui.ScrollDeltaPixels(delta),
		}, cx.App)
	case roll < 95:
		window.FocusNext(cx.App)
	case roll < 98:
		return gpui.Some(gpui.NewSize(
			gpui.Px(randomF32Range(w.inputRng, 480.0, 1600.0)),
			gpui.Px(randomF32Range(w.inputRng, 360.0, 1000.0)),
		))
	default:
		cx.Notify()
	}
	return gpui.None[gpui.Size[gpui.Pixels]]()
}

type mixedRun struct {
	window   *gpui.BenchWindowContext
	tree     gpui.Entity[gpui.RandomizedElementTree]
	workload *mixedWorkload
}

func (r *mixedRun) Release() {
	r.window.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() })
	r.tree.Release()
}

func measureMixed(family treeFamily, input treeInput, cachedViews bool, cx *gpui.BenchAppContext) {
	config := input.config.
		WithCachedViews(cachedViews).
		WithHoverStyles(true).
		WithAnimationDensity(0.15)
	cx.BenchBatchedTask(
		func(cx *gpui.BenchAppContext) *mixedRun {
			window := cx.AddEmptyWindow()
			tree := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[gpui.RandomizedElementTree] {
				return w.ReplaceRoot(app, func(_ *gpui.Window, cx *gpui.Context[gpui.RandomizedElementTree]) *gpui.RandomizedElementTree {
					return gpui.NewRandomizedElementTreeWithConfig(config, cx)
				})
			})
			cx.RunUntilIdle()
			window.Update(func(w *gpui.Window, _ *gpui.App) { w.Refresh() })
			cx.RunUntilIdle()
			initialSize := cx.ReadR(func(app *gpui.App) int {
				snapshot := tree.Read(app).Snapshot()
				return snapshot.DescendantCount()
			})
			return &mixedRun{
				window:   window,
				tree:     tree,
				workload: newMixedWorkload(config.Seed(), initialSize, cachedViews),
			}
		},
		func(run **mixedRun, cx *gpui.BenchAppContext) *gpui.Task[struct{}] {
			r := *run
			for range family.mixedChanges {
				resize := r.window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Option[gpui.Size[gpui.Pixels]] {
					var resize gpui.Option[gpui.Size[gpui.Pixels]]
					r.tree.Update(app, func(tree *gpui.RandomizedElementTree, cx *gpui.Context[gpui.RandomizedElementTree]) {
						resize = r.workload.change(tree, w, cx)
					})
					return resize
				})
				if size, ok := resize.Get(); ok {
					r.window.SimulateResize(size)
				}
				cx.RunUntilIdle()
			}
			return scheduler.ReadyTask(struct{}{})
		},
	)
}

func BenchmarkRandomizedElementTree_Mixed(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "RandomizedTree/mixed"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureMixed(family, input, false, cx)
	})
}

func cachedInput(family treeFamily, rng *rand.Rand) treeInput {
	input := family.sample(rng)
	input.config = input.config.WithCachedViews(true)
	return input
}

func BenchmarkRandomizedElementTree_CachedFullRefresh(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/full refresh"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := cachedInput(family, rng)
		measureTreeInWindow("cached full refresh", input, cx, func(_ *gpui.RandomizedElementTree, w *gpui.Window, _ *gpui.Context[gpui.RandomizedElementTree]) {
			w.Refresh()
		})
	})
}

func BenchmarkRandomizedElementTree_CachedElementStyle(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/element style"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := cachedInput(family, rng)
		measureElementStyle("cached element style", input, cx)
	})
}

func BenchmarkRandomizedElementTree_CachedLeafBounds(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/leaf bounds"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := cachedInput(family, rng)
		measureTree("cached leaf bounds", input, cx, applyKind(gpui.RandomizedElementTreeMutationKindChildBounds))
	})
}

func BenchmarkRandomizedElementTree_CachedInsertRemove(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/insert-remove"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := cachedInput(family, rng)
		measureTree("cached insert-remove", input, cx, alternateInsertRemove())
	})
}

func BenchmarkRandomizedElementTree_CachedHover(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/hover"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := cachedInput(family, rng)
		input.config = input.config.WithHoverStyles(true)
		measureHover("cached hover", input, cx)
	})
}

func BenchmarkRandomizedElementTree_CachedMixed(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/mixed"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := family.sample(rng)
		measureMixed(family, input, true, cx)
	})
}

func BenchmarkRandomizedElementTree_CachedReorder(b *testing.B) {
	gpui.RunBenchInputsRng(b, treeBenchOptions(b, "CachedTree/reorder"), treeFamilies(), func(family treeFamily, rng *rand.Rand, cx *gpui.BenchAppContext) {
		input := cachedInput(family, rng)
		measureTree("cached reorder", input, cx, applyKind(gpui.RandomizedElementTreeMutationKindReorder))
	})
}

type changingShareInput struct {
	family       treeFamily
	sharePercent int
	locality     gpui.ChangeLocality
}

func (i changingShareInput) localityName() string {
	if i.locality == gpui.ChangeLocalitySpread {
		return "spread"
	}
	return "local"
}

func (i changingShareInput) String() string {
	return fmt.Sprintf("%v-f%d-%s", i.family, i.sharePercent, i.localityName())
}

func changingShareInputs() []changingShareInput {
	var inputs []changingShareInput
	for _, family := range treeFamilies() {
		for _, share := range []struct {
			percent  int
			locality gpui.ChangeLocality
		}{
			{0, gpui.ChangeLocalityLocalized},
			{1, gpui.ChangeLocalityLocalized},
			{5, gpui.ChangeLocalityLocalized},
			{25, gpui.ChangeLocalityLocalized},
			{100, gpui.ChangeLocalityLocalized},
			{5, gpui.ChangeLocalitySpread},
			{25, gpui.ChangeLocalitySpread},
		} {
			inputs = append(inputs, changingShareInput{family: family, sharePercent: share.percent, locality: share.locality})
		}
	}
	return inputs
}

func BenchmarkRandomizedElementTree_ChangingShare(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "tree", Iterations: 1, Group: "RandomizedTree/changing share", Fps: 120})
	gpui.RunBenchInputsRng(b, opts, changingShareInputs(), func(input changingShareInput, rng *rand.Rand, cx *gpui.BenchAppContext) {
		tree := input.family.sample(rng)
		elementCount := tree.config.ElementCount()
		perFrame := 0
		if input.sharePercent != 0 {
			perFrame = max((input.sharePercent*elementCount+99)/100, 1)
		}
		group := fmt.Sprintf("changing share f%d-%s", input.sharePercent, input.localityName())
		measureTree(group, tree, cx, func(tree *gpui.RandomizedElementTree, cx *gpui.Context[gpui.RandomizedElementTree]) {
			if perFrame == 0 {
				cx.Notify()
			} else {
				tree.RecolorChildren(perFrame, input.locality, cx)
			}
		})
	})
}

func metricLabels(count int, value int) []gpui.IntoElement {
	labels := make([]gpui.IntoElement, 0, count)
	for label := range count {
		labels = append(labels, gpui.NewDiv().
			FlexShrink0().
			H(gpui.Px(16.)).
			ChildText(fmt.Sprintf("Metric %02d: %06d", label, value)))
	}
	return labels
}

type textPanel struct {
	index int
	value uint32
}

func (p *textPanel) Render(_ *gpui.Window, _ *gpui.Context[textPanel]) gpui.IntoElement {
	return gpui.NewDiv().
		ID(gpui.ElementIDNamedInteger("text-panel", uint64(p.index))).
		Flex().
		FlexCol().
		W(gpui.Px(220.)).
		H(gpui.Px(192.)).
		Bg(gpui.Rgb(0x181818)).
		OverflowHidden().
		Children(metricLabels(64, int(p.value))...)
}

type textPanelRoot struct {
	panels     []gpui.Entity[textPanel]
	fontFamily gpui.SharedString
}

func newTextPanelRoot(panelCount int, fontFamily string, cx *gpui.Context[textPanelRoot]) *textPanelRoot {
	root := &textPanelRoot{fontFamily: gpui.SharedString(fontFamily)}
	for index := range panelCount {
		root.panels = append(root.panels, cx.New(func(*gpui.Context[textPanel]) *textPanel {
			return &textPanel{index: index, value: 0}
		}))
	}
	return root
}

func (r *textPanelRoot) Drop(_ *gpui.App) {
	for _, panel := range r.panels {
		panel.Release()
	}
	r.panels = nil
}

func entityChildren[T any](entities []gpui.Entity[T]) []gpui.IntoElement {
	children := make([]gpui.IntoElement, 0, len(entities))
	for _, entity := range entities {
		children = append(children, entity.Clone())
	}
	return children
}

func (r *textPanelRoot) Render(_ *gpui.Window, _ *gpui.Context[textPanelRoot]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("text-panels").
		Flex().
		FlexWrap().
		FontFamily(string(r.fontFamily)).
		TextSize(gpui.Px(12.)).
		TextColor(gpui.Rgb(0xffffff)).
		ChildText("0123456789").
		Children(entityChildren(r.panels)...)
}

func textShapingAvailable() bool {
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "text benchmarks are unavailable on Windows: headless platforms shape no text")
		return false
	}
	return true
}

func textPanelCounts() []int {
	if textShapingAvailable() {
		return []int{8, 32, 96}
	}
	return nil
}

func BenchmarkRandomizedElementTree_TextPanelUpdates(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "panels", Group: "TextPanels/two updates", Fps: 120})
	gpui.RunBenchInputs(b, opts, textPanelCounts(), func(panelCount int, cx *gpui.BenchAppContext) {
		measureTextPanelUpdates(panelCount, "monospace", cx)
	})
}

func installedFont() string {
	if runtime.GOOS == "darwin" {
		return "Menlo"
	}
	return "Noto Sans Mono"
}

func requireInstalledFont(cx *gpui.BenchAppContext) {
	font := gpui.NewFont(gpui.SharedString(installedFont()))
	cx.Read(func(app *gpui.App) {
		textSystem := app.TextSystem()
		fontID := textSystem.ResolveFont(font)
		resolved, ok := textSystem.GetFontForID(fontID)
		if !ok || resolved.Family != font.Family {
			panic(fmt.Sprintf("assertion `left == right` failed: the text benchmarks require %s to be installed", installedFont()))
		}
	})
}

func BenchmarkRandomizedElementTree_InstalledFontPanelUpdates(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "panels", Group: "TextPanels/installed font two updates", Fps: 120})
	gpui.RunBenchInputs(b, opts, textPanelCounts(), func(panelCount int, cx *gpui.BenchAppContext) {
		requireInstalledFont(cx)
		measureTextPanelUpdates(panelCount, installedFont(), cx)
	})
}

func measureTextPanelUpdates(panelCount int, fontFamily string, cx *gpui.BenchAppContext) {
	window := cx.AddEmptyWindow()
	root := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[textPanelRoot] {
		return w.ReplaceRoot(app, func(_ *gpui.Window, cx *gpui.Context[textPanelRoot]) *textPanelRoot {
			return newTextPanelRoot(panelCount, fontFamily, cx)
		})
	})
	defer root.Release()
	cx.RunUntilIdle()
	window.Update(func(w *gpui.Window, _ *gpui.App) { w.Refresh() })
	cx.RunUntilIdle()
	cx.BenchRenderer(root, func(root *textPanelRoot, _ *gpui.Window, cx *gpui.Context[textPanelRoot]) {
		for _, index := range []int{0, panelCount / 2} {
			if index < len(root.panels) {
				root.panels[index].Update(cx.App, func(panel *textPanel, cx *gpui.Context[textPanel]) {
					panel.value = (panel.value + 1) % 1_000_000
					cx.Notify()
				})
			}
		}
	})
	window.Update(func(w *gpui.Window, _ *gpui.App) { reportSceneStats("TextPanels", fmt.Sprint(panelCount), w) })
	window.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() })
}

type populatedWindow struct {
	cx *gpui.AsyncWindowContext
}

func (p *populatedWindow) Release() {
	if err := p.cx.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() }); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove populated window: %v\n", err)
	}
}

func populate(window *gpui.Window, cx *gpui.App, panelCount int) *populatedWindow {
	root := window.ReplaceRoot(cx, func(_ *gpui.Window, cx *gpui.Context[textPanelRoot]) *textPanelRoot {
		return newTextPanelRoot(panelCount, installedFont(), cx)
	})
	root.Release()
	return &populatedWindow{cx: window.ToAsync(cx)}
}

var describedFirstPopulated []int

func BenchmarkRandomizedElementTree_FirstPopulatedTextPanels(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "panels", Group: "TextPanels/first populated installed font", Fps: 120})
	gpui.RunBenchInputs(b, opts, textPanelCounts(), func(panelCount int, cx *gpui.BenchAppContext) {
		requireInstalledFont(cx)

		describe := !slices.Contains(describedFirstPopulated, panelCount)
		if describe {
			describedFirstPopulated = append(describedFirstPopulated, panelCount)
			before := readAllocationStats()
			window := cx.AddEmptyWindow()
			empty := readAllocationStats()
			populated := window.UpdateR(func(w *gpui.Window, app *gpui.App) *populatedWindow { return populate(w, app, panelCount) })
			cx.RunUntilIdle()
			after := readAllocationStats()
			fmt.Fprintf(os.Stderr, "first populated panels: %d, family %s; live heap before window %d B, empty window %d B, populated %d B; population delta %d B (process-wide estimates, not exclusive window ownership)\n",
				panelCount, installedFont(), before.liveBytes, empty.liveBytes, after.liveBytes, int64(after.liveBytes)-int64(empty.liveBytes))
			window.Update(func(w *gpui.Window, _ *gpui.App) {
				reportSceneStats("TextPanels first populated", fmt.Sprint(panelCount), w)
			})
			populated.Release()
			cx.RunUntilIdle()
		}

		cx.BenchBatchedTask(
			func(cx *gpui.BenchAppContext) *gpui.BenchWindowContext { return cx.AddEmptyWindow() },
			func(window **gpui.BenchWindowContext, _ *gpui.BenchAppContext) *gpui.Task[*populatedWindow] {
				populated := (*window).UpdateR(func(w *gpui.Window, app *gpui.App) *populatedWindow { return populate(w, app, panelCount) })
				return scheduler.ReadyTask(populated)
			},
		)
	})
}

type churn uint8

const (
	churnReplace churn = iota
	churnRemount
	churnID
	churnText
)

func (c churn) String() string {
	switch c {
	case churnReplace:
		return "replace-8-of-96"
	case churnRemount:
		return "remount-48-of-96"
	case churnID:
		return "id-and-anonymous-count"
	default:
		return "text-96"
	}
}

type churnPanel struct {
	index    int
	frame    int
	workload churn
}

func (p *churnPanel) Render(_ *gpui.Window, _ *gpui.Context[churnPanel]) gpui.IntoElement {
	id := p.index
	if p.workload == churnID {
		id = p.frame*96 + p.index
	}
	labels := 64
	if p.workload == churnID {
		labels = 56 + p.frame%10
	}
	value := p.frame % 1_000_000
	if p.workload == churnID {
		value = 0
	}
	return gpui.NewDiv().
		ID(gpui.ElementIDNamedInteger("churn-panel", uint64(id))).
		Flex().
		FlexCol().
		W(gpui.Px(220.)).
		H(gpui.Px(192.)).
		Bg(gpui.Rgb(0x181818)).
		OverflowHidden().
		Children(metricLabels(labels, value)...)
}

type churnRoot struct {
	panels   []gpui.Entity[churnPanel]
	workload churn
	frame    int
}

func newChurnRoot(workload churn, cx *gpui.Context[churnRoot]) *churnRoot {
	root := &churnRoot{workload: workload}
	for index := range 96 {
		root.panels = append(root.panels, cx.New(func(*gpui.Context[churnPanel]) *churnPanel {
			return &churnPanel{index: index, frame: 0, workload: workload}
		}))
	}
	return root
}

func (r *churnRoot) Drop(_ *gpui.App) {
	for _, panel := range r.panels {
		panel.Release()
	}
	r.panels = nil
}

func (r *churnRoot) visiblePanels() int {
	if r.workload == churnRemount && r.frame%2 == 1 {
		return 48
	}
	return 96
}

func (r *churnRoot) advance(cx *gpui.Context[churnRoot]) {
	r.frame++
	switch r.workload {
	case churnReplace:
		for offset := range 8 {
			index := ((r.frame-1)*8 + offset) % 96
			frame, workload := r.frame, r.workload
			replacement := cx.New(func(*gpui.Context[churnPanel]) *churnPanel {
				return &churnPanel{index: index, frame: frame, workload: workload}
			})
			if index < len(r.panels) {
				r.panels[index].Release()
				r.panels[index] = replacement
			} else {
				replacement.Release()
			}
		}
	case churnID, churnText:
		for _, panel := range r.panels {
			frame := r.frame
			panel.Update(cx.App, func(panel *churnPanel, cx *gpui.Context[churnPanel]) {
				panel.frame = frame
				cx.Notify()
			})
		}
	case churnRemount:
	}
	cx.Notify()
}

func (r *churnRoot) Render(_ *gpui.Window, _ *gpui.Context[churnRoot]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("churn-root").
		Flex().
		FlexWrap().
		FontFamily(installedFont()).
		TextSize(gpui.Px(12.)).
		TextColor(gpui.Rgb(0xffffff)).
		ChildText("0123456789").
		Children(entityChildren(r.panels[:r.visiblePanels()])...)
}

func churnWorkloads() []churn {
	if textShapingAvailable() {
		return []churn{churnReplace, churnRemount, churnID, churnText}
	}
	return nil
}

var heapCheckedChurn []churn

func reportChurnHeap(workload churn, cx *gpui.BenchAppContext) {
	if slices.Contains(heapCheckedChurn, workload) {
		return
	}
	heapCheckedChurn = append(heapCheckedChurn, workload)
	beforeWindow := readAllocationStats()
	window := cx.AddEmptyWindow()
	root := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[churnRoot] {
		return w.ReplaceRoot(app, func(_ *gpui.Window, cx *gpui.Context[churnRoot]) *churnRoot { return newChurnRoot(workload, cx) })
	})
	cx.RunUntilIdle()
	type checkpoint struct {
		frame int
		stats allocationStats
	}
	checkpoints := make([]checkpoint, 0, 4)
	checkpoints = append(checkpoints, checkpoint{0, readAllocationStats()})
	for frame := 1; frame <= 500; frame++ {
		window.Update(func(_ *gpui.Window, app *gpui.App) {
			root.Update(app, func(root *churnRoot, cx *gpui.Context[churnRoot]) { root.advance(cx) })
		})
		cx.RunUntilIdle()
		if frame == 50 || frame == 200 || frame == 500 {
			checkpoints = append(checkpoints, checkpoint{frame, readAllocationStats()})
		}
	}
	window.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() })
	root.Release()
	cx.RunUntilIdle()
	for _, c := range checkpoints {
		fmt.Fprintf(os.Stderr, "layout churn %v: frame %d, live heap %d B, window delta %d B (process-wide requested bytes)\n",
			workload, c.frame, c.stats.liveBytes, int64(c.stats.liveBytes)-int64(beforeWindow.liveBytes))
	}
}

func BenchmarkRandomizedElementTree_LayoutChurn(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "workload", Group: "LayoutChurn", Fps: 120})
	gpui.RunBenchInputs(b, opts, churnWorkloads(), func(workload churn, cx *gpui.BenchAppContext) {
		requireInstalledFont(cx)
		reportChurnHeap(workload, cx)
		window := cx.AddEmptyWindow()
		root := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[churnRoot] {
			return w.ReplaceRoot(app, func(_ *gpui.Window, cx *gpui.Context[churnRoot]) *churnRoot { return newChurnRoot(workload, cx) })
		})
		defer root.Release()
		cx.RunUntilIdle()
		cx.BenchRenderer(root, func(root *churnRoot, _ *gpui.Window, cx *gpui.Context[churnRoot]) { root.advance(cx) })
		window.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() })
	})
}

type scrollWorkload uint8

const (
	scrollWorkloadText scrollWorkload = iota
	scrollWorkloadVirtual
	scrollWorkloadNested
)

func (w scrollWorkload) String() string {
	switch w {
	case scrollWorkloadText:
		return "editor-like-text"
	case scrollWorkloadVirtual:
		return "uniform-list"
	default:
		return "pane-with-static-siblings"
	}
}

type scrollPane struct {
	workload   scrollWorkload
	handle     gpui.ScrollHandle
	listHandle gpui.UniformListScrollHandle
	distance   float32
	direction  float32
}

func (p *scrollPane) advance(cx *gpui.Context[scrollPane]) {
	if p.distance >= 6000.0 {
		p.direction = -1.0
	} else if p.distance <= 600.0 {
		p.direction = 1.0
	}
	p.distance += p.direction * 12.0
	p.handle.SetOffset(gpui.NewPoint(gpui.Px(0.0), gpui.Px(-p.distance)))
	cx.Notify()
}

func scrollRow(index int) *gpui.Div {
	return gpui.NewDiv().
		Flex().
		H(gpui.Px(22.0)).
		FlexShrink0().
		Child(gpui.NewDiv().W(gpui.Px(64.0)).TextColor(gpui.Rgb(0x858585)).ChildText(fmt.Sprintf("%5d", index))).
		Child(gpui.NewDiv().TextColor(gpui.Rgb(0xc586c0)).ChildText("let ")).
		Child(gpui.NewDiv().TextColor(gpui.Rgb(0x9cdcfe)).ChildText(fmt.Sprintf("line_%d", index))).
		Child(gpui.NewDiv().TextColor(gpui.Rgb(0xd4d4d4)).ChildText(" = document.visible_lines(viewport);"))
}

func (p *scrollPane) Render(_ *gpui.Window, cx *gpui.Context[scrollPane]) gpui.IntoElement {
	container := gpui.NewDiv().
		SizeFull().
		FontFamily(installedFont()).
		TextSize(gpui.Px(14.0)).
		Bg(gpui.Rgb(0x202020))
	if p.workload == scrollWorkloadVirtual {
		return container.Child(
			gpui.NewUniformList(
				"scroll-rows",
				5000,
				cx.Processor(func(_ *scrollPane, visibleRange gpui.Range, _ *gpui.Window, _ *gpui.Context[scrollPane]) []*gpui.Div {
					rows := make([]*gpui.Div, 0, visibleRange.End-visibleRange.Start)
					for index := visibleRange.Start; index < visibleRange.End; index++ {
						rows = append(rows, scrollRow(index))
					}
					return rows
				}),
			).
				SizeFull().
				TrackUniformListScroll(p.listHandle),
		).IntoAnyElement()
	}
	rows := make([]gpui.IntoElement, 0, 1000)
	for index := range 1000 {
		rows = append(rows, scrollRow(index))
	}
	return container.
		ID("scroll-text").
		OverflowYScroll().
		TrackScroll(p.handle).
		Children(rows...).
		IntoAnyElement()
}

type scrollStaticPanel struct {
	status bool
}

func (p *scrollStaticPanel) Render(_ *gpui.Window, _ *gpui.Context[scrollStaticPanel]) gpui.IntoElement {
	count := 40
	label := "File"
	if p.status {
		count = 1
		label = "Status"
	}
	rows := make([]gpui.IntoElement, 0, count)
	for index := range count {
		rows = append(rows, gpui.NewDiv().H(gpui.Px(22.0)).ChildText(fmt.Sprintf("%s %d", label, index)))
	}
	return gpui.NewDiv().
		SizeFull().
		Bg(gpui.Rgb(0x303030)).
		TextColor(gpui.Rgb(0xcccccc)).
		Children(rows...)
}

type scrollRoot struct {
	pane    gpui.Entity[scrollPane]
	sidebar gpui.Entity[scrollStaticPanel]
	status  gpui.Entity[scrollStaticPanel]
	nested  bool
}

func (r *scrollRoot) Drop(_ *gpui.App) {
	r.pane.Release()
	r.sidebar.Release()
	r.status.Release()
}

func (r *scrollRoot) Render(_ *gpui.Window, _ *gpui.Context[scrollRoot]) gpui.IntoElement {
	return gpui.NewDiv().
		SizeFull().
		Flex().
		FlexCol().
		Child(gpui.NewDiv().
			Flex().
			Flex1().
			MinH0().
			When(r.nested, func(element *gpui.Div) *gpui.Div {
				return element.Child(gpui.NewDiv().W(gpui.Px(220.0)).HFull().Child(r.sidebar.Clone()))
			}).
			Child(gpui.NewDiv().
				Flex1().
				MinW0().
				HFull().
				OverflowHidden().
				Child(r.pane.Clone()))).
		When(r.nested, func(element *gpui.Div) *gpui.Div {
			return element.Child(gpui.NewDiv().H(gpui.Px(28.0)).Child(r.status.Clone()))
		})
}

func scrollWorkloads() []scrollWorkload {
	if textShapingAvailable() {
		return []scrollWorkload{scrollWorkloadText, scrollWorkloadVirtual, scrollWorkloadNested}
	}
	return nil
}

func BenchmarkRandomizedElementTree_Scroll(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "workload", Group: "Scroll", Fps: 360})
	gpui.RunBenchInputs(b, opts, scrollWorkloads(), func(workload scrollWorkload, cx *gpui.BenchAppContext) {
		window := cx.AddEmptyWindow()
		pane := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[scrollPane] {
			pane := app.New(func(*gpui.Context[scrollPane]) *scrollPane {
				listHandle := gpui.NewUniformListScrollHandle()
				handle := gpui.NewScrollHandle()
				if workload == scrollWorkloadVirtual {
					handle = listHandle.State.BaseHandle
				}
				return &scrollPane{
					workload:   workload,
					handle:     handle,
					listHandle: listHandle,
					distance:   600.0,
					direction:  1.0,
				}
			})
			root := w.ReplaceRoot(app, func(_ *gpui.Window, cx *gpui.Context[scrollRoot]) *scrollRoot {
				return &scrollRoot{
					pane:    pane.Clone(),
					sidebar: cx.New(func(*gpui.Context[scrollStaticPanel]) *scrollStaticPanel { return &scrollStaticPanel{status: false} }),
					status:  cx.New(func(*gpui.Context[scrollStaticPanel]) *scrollStaticPanel { return &scrollStaticPanel{status: true} }),
					nested:  workload == scrollWorkloadNested,
				}
			})
			root.Release()
			return pane
		})
		defer pane.Release()
		cx.RunUntilIdle()
		for range 8 {
			window.Update(func(_ *gpui.Window, app *gpui.App) {
				pane.Update(app, func(pane *scrollPane, cx *gpui.Context[scrollPane]) { pane.advance(cx) })
			})
			cx.RunUntilIdle()
		}
		cx.BenchRenderer(pane, func(pane *scrollPane, _ *gpui.Window, cx *gpui.Context[scrollPane]) { pane.advance(cx) })
		window.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() })
	})
}

func BenchmarkRandomizedElementTree_FamilySummary(b *testing.B) {
	summary := treeSummary
	treeSummary = nil
	if len(summary) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "\nRandomizedTree family summary (geometric mean across seeds; includes Criterion warm-up)")
	fmt.Fprintf(os.Stderr, "  %-34s %-15s %5s %14s %14s\n", "group", "family", "seeds", "time/iter", "allocs/iter")
	for _, entry := range summary {
		measured := slices.ContainsFunc(entry.seeds, func(seed summarySeed) bool { return seed.totals.iterations > 0 })
		if !measured {
			fmt.Fprintf(os.Stderr, "  %-34s %-15s %5d %14s\n", entry.group, entry.family, len(entry.seeds), "unavailable")
			continue
		}
		perIteration := func(value func(totals seedTotals) float64) float64 {
			var values []float64
			for _, seed := range entry.seeds {
				if seed.totals.iterations > 0 {
					values = append(values, value(seed.totals)/float64(seed.totals.iterations))
				}
			}
			return geometricMean(values)
		}
		elapsed := perIteration(func(totals seedTotals) float64 { return totals.elapsed.Seconds() })
		allocations := perIteration(func(totals seedTotals) float64 { return float64(totals.allocations) })
		fmt.Fprintf(os.Stderr, "  %-34s %-15s %5d %11.1f µs %14.0f\n", entry.group, entry.family, len(entry.seeds), elapsed*1e6, allocations)
	}
}
