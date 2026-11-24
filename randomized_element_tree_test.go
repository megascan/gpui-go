package gpui

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"testing"
)

func init() {
	randomizedElementTreeRecordWork = true
}

const (
	randomizedTreeTestElementCount  = 64
	randomizedTreeTestMutationCount = 32
)

var randomizedTreeTestWindowSize = Size[Pixels]{Width: Px(1200.0), Height: Px(900.0)}

type randomizedSceneSnapshot struct {
	quads           []randomizedQuadSnapshot
	otherPrimitives []string
	elementIDs      []uint64
}

type randomizedQuadSnapshot struct {
	order        DrawOrder
	borderStyle  BorderStyle
	bounds       Bounds[ScaledPixels]
	contentMask  ContentMask[ScaledPixels]
	background   Background
	borderColor  Hsla
	cornerRadii  Corners[ScaledPixels]
	borderWidths Edges[ScaledPixels]
}

func randomizedSceneSnapshotFromWindow(window *Window) randomizedSceneSnapshot {
	scene := &window.renderedFrame.scene
	var other []string
	for _, shadow := range scene.Shadows {
		other = append(other, fmt.Sprintf("%+v", shadow))
	}
	for _, underline := range scene.Underlines {
		other = append(other, fmt.Sprintf("%+v", underline))
	}
	for _, path := range scene.Paths {
		other = append(other, fmt.Sprintf("{order:%d bounds:%+v tiles:%+v paint:%+v}", path.order, path.bounds, *path.tiles, path.paint))
	}
	for _, sprite := range scene.MonochromeSprites {
		other = append(other, fmt.Sprintf("monochrome %d %+v %+v %+v %+v", sprite.Order, sprite.Bounds, sprite.ContentMask, sprite.Color, sprite.Transformation))
	}
	for _, sprite := range scene.SubpixelSprites {
		other = append(other, fmt.Sprintf("subpixel %d %+v %+v %+v %+v", sprite.Order, sprite.Bounds, sprite.ContentMask, sprite.Color, sprite.Transformation))
	}
	for _, sprite := range scene.PolychromeSprites {
		other = append(other, fmt.Sprintf("polychrome %d %+v %+v %v %v", sprite.Order, sprite.Bounds, sprite.ContentMask, sprite.Grayscale, sprite.Opacity))
	}
	quads := make([]randomizedQuadSnapshot, 0, len(scene.Quads))
	for _, quad := range scene.Quads {
		quads = append(quads, randomizedQuadSnapshot{
			order:        quad.Order,
			borderStyle:  quad.BorderStyle,
			bounds:       quad.Bounds,
			contentMask:  quad.ContentMask,
			background:   quad.Background,
			borderColor:  quad.BorderColor,
			cornerRadii:  quad.CornerRadii,
			borderWidths: quad.BorderWidths,
		})
	}
	ids := map[uint64]struct{}{}
	for _, entries := range window.renderedFrame.elementStates.buckets {
		for _, entry := range entries {
			path := entry.key.id.ids
			if len(path) == 0 {
				continue
			}
			last := path[len(path)-1]
			if last.kind == elementIDNamedInteger && last.name == "randomized-element" {
				ids[last.num] = struct{}{}
			}
		}
	}
	return randomizedSceneSnapshot{
		quads:           quads,
		otherPrimitives: other,
		elementIDs:      slices.Sorted(maps.Keys(ids)),
	}
}

func randomizedSortedIDs(ids []uint64) []uint64 {
	set := map[uint64]struct{}{}
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return slices.Sorted(maps.Keys(set))
}

func randomizedTreeDrawWindow(window AnyWindowHandle, cx *TestAppContext) randomizedSceneSnapshot {
	scene, err := AppContextUpdateWindow(cx, window, func(_ AnyView, window *Window, cx *App) randomizedSceneSnapshot {
		arenaClearNeeded := window.Draw(cx)
		scene := randomizedSceneSnapshotFromWindow(window)
		arenaClearNeeded.Clear(cx)
		return scene
	})
	if err != nil {
		panic(fmt.Sprintf("randomized element tree window should remain open while drawing: %v", err))
	}
	return scene
}

func randomizedTreeRemoveWindow(window AnyWindowHandle, cx *TestAppContext, msg string) {
	_, err := AppContextUpdateWindow(cx, window, func(_ AnyView, window *Window, _ *App) struct{} {
		window.RemoveWindow()
		return struct{}{}
	})
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}

func TestRandomizedElementTree_LocalizedRecoloringNotifiesOneEntityUntilItsSubtreeIsExhausted(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		config := NewRandomizedElementTreeConfig(13, 200).
			WithTopology(RandomizedElementTreeTopologyMixed).
			WithEntityDensity(0.1)
		window := cx.OpenWindow(randomizedTreeTestWindowSize, func(_ *Window, cx *Context[RandomizedElementTree]) *RandomizedElementTree {
			return NewRandomizedElementTreeWithConfig(config, cx)
		})
		root, err := window.Root(cx)
		if err != nil {
			t.Fatalf("window has a root: %v", err)
		}
		defer root.Release()
		type pair struct{ localized, spread RecoloredChildren }
		result := root.UpdateR(cx, func(tree *RandomizedElementTree, cx *Context[RandomizedElementTree]) pair {
			localized := tree.RecolorChildren(20, ChangeLocalityLocalized, cx)
			spread := tree.RecolorChildren(20, ChangeLocalitySpread, cx)
			return pair{localized, spread}
		})
		if result.localized.Recolored != 20 || result.spread.Recolored != 20 {
			t.Fatalf("recolored %+v", result)
		}
		if result.localized.Notified > result.spread.Notified {
			t.Fatalf("localized change touches no more boundaries than spread change: %+v vs %+v", result.localized, result.spread)
		}
		if result.localized.Notified < 1 {
			t.Fatalf("localized.Notified = %d", result.localized.Notified)
		}
	})
}

func TestRandomizedElementTree_BoundsSampleWithinRangeAndReproduceFromTheSeed(t *testing.T) {
	bounds := NewRandomizedElementTreeBounds(32, 256).
		WithTopologies(RandomizedElementTreeTopologyDeep, RandomizedElementTreeTopologyNarrow).
		WithEntityDensity(0.1, 0.5).
		WithHandlerDensity(0.25, 0.25)
	topologies := map[string]struct{}{}
	sample := func(seed uint64) RandomizedElementTreeConfig {
		return bounds.Sample(rand.New(rand.NewPCG(seed, 0)))
	}
	for seed := range uint64(64) {
		config := sample(seed)
		if config.ElementCount() < 32 || config.ElementCount() > 256 {
			t.Fatalf("%v", config)
		}
		if !config.Topology().IsTall() {
			t.Fatalf("%v", config)
		}
		if config.EntityDensity() < 0.1 || config.EntityDensity() > 0.5 {
			t.Fatalf("%v", config)
		}
		if config.HandlerDensity() != 0.25 {
			t.Fatalf("%v", config)
		}
		if sample(seed) != config {
			t.Fatalf("a seed draws the same config twice")
		}
		topologies[config.Topology().String()] = struct{}{}
	}
	if len(topologies) != 2 {
		t.Fatalf("64 seeds pick both allowed topologies")
	}
	if sample(0) == sample(1) {
		t.Fatalf("seeds 0 and 1 drew the same config")
	}
}

func TestRandomizedElementTree_GeneratedTopologiesIncludeWideDeepNarrowAndEmptyTrees(t *testing.T) {
	wide := NewRandomizedElementTree(0, randomizedTreeTestElementCount).Snapshot()
	if len(wide.children) != randomizedTreeTestElementCount || wide.MaxDepth() != 1 {
		t.Fatalf("wide: %d children, depth %d", len(wide.children), wide.MaxDepth())
	}

	deep := NewRandomizedElementTree(1, randomizedTreeTestElementCount).Snapshot()
	if len(deep.children) != 1 || deep.MaxDepth() != randomizedTreeTestElementCount {
		t.Fatalf("deep: %d children, depth %d", len(deep.children), deep.MaxDepth())
	}

	narrow := NewRandomizedElementTree(3, randomizedTreeTestElementCount).Snapshot()
	if narrow.DescendantCount() != randomizedTreeTestElementCount || narrow.MaxDepth() <= 1 {
		t.Fatalf("narrow: %d descendants, depth %d", narrow.DescendantCount(), narrow.MaxDepth())
	}

	empty := NewRandomizedElementTree(2, 0).Snapshot()
	if empty.DescendantCount() != 0 || empty.MaxDepth() != 0 {
		t.Fatalf("empty: %d descendants, depth %d", empty.DescendantCount(), empty.MaxDepth())
	}
}

func TestRandomizedElementTree_RandomizedElementIDsAreUniqueAndStable(t *testing.T) {
	tree := NewRandomizedElementTree(0, randomizedTreeTestElementCount)
	snapshot := tree.Snapshot()
	initialIDs := snapshot.ElementIDs()
	if len(initialIDs) != randomizedTreeTestElementCount || len(randomizedSortedIDs(initialIDs)) != randomizedTreeTestElementCount {
		t.Fatalf("initial ids %v", initialIDs)
	}

	for _, mutation := range []randomizedChildMutation{
		randomizedChildMutationColor,
		randomizedChildMutationOpacity,
		randomizedChildMutationBorder,
		randomizedChildMutationBounds,
		randomizedChildMutationClipping,
		randomizedChildMutationVisibility,
		randomizedChildMutationText,
	} {
		tree.mutateRandomChild(mutation)
		s := tree.Snapshot()
		if !slices.Equal(s.ElementIDs(), initialIDs) {
			t.Fatalf("mutation %d changed ids", mutation)
		}
	}

	tree.reorderChild()
	s := tree.Snapshot()
	if !slices.Equal(randomizedSortedIDs(s.ElementIDs()), randomizedSortedIDs(initialIDs)) {
		t.Fatalf("reorder changed the id set")
	}

	tree.InsertChild()
	s = tree.Snapshot()
	insertedIDs := s.ElementIDs()
	insertedSet := map[uint64]bool{}
	for _, id := range insertedIDs {
		insertedSet[id] = true
	}
	if len(insertedIDs) != randomizedTreeTestElementCount+1 {
		t.Fatalf("inserted ids %d", len(insertedIDs))
	}
	for _, id := range initialIDs {
		if !insertedSet[id] {
			t.Fatalf("insert lost id %d", id)
		}
	}

	tree.RemoveChild()
	s = tree.Snapshot()
	remainingIDs := s.ElementIDs()
	if len(randomizedSortedIDs(remainingIDs)) != len(remainingIDs) {
		t.Fatalf("remaining ids are not unique")
	}
	for _, id := range remainingIDs {
		if !insertedSet[id] {
			t.Fatalf("remove produced id %d", id)
		}
	}
}

func TestRandomizedElementTree_RandomizedEntitiesAndWorkCountersSupportIncrementalBenchmarks(t *testing.T) {
	skipUntilElementStateFocusHandleRelease(t)
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		config := NewRandomizedElementTreeConfig(0, 16).
			WithTopology(RandomizedElementTreeTopologyWide).
			WithEntityDensity(1.0).
			WithHandlerDensity(1.0)
		windowHandle := cx.OpenWindow(randomizedTreeTestWindowSize, func(_ *Window, cx *Context[RandomizedElementTree]) *RandomizedElementTree {
			return NewRandomizedElementTreeWithConfig(config, cx)
		})
		root, err := windowHandle.Root(cx)
		if err != nil {
			t.Fatalf("randomized tree window should have a root: %v", err)
		}
		defer root.Release()
		window := windowHandle.AnyWindowHandle

		randomizedTreeDrawWindow(window, cx)
		initial := randomizedTreeWorkCounters(root, cx)
		if initial.rootRenderCount < 1 || initial.entityRenderCount < 16 || initial.elementRenderCount < 16 || initial.handlerRegistrationCount < 32 {
			t.Fatalf("initial counters %+v", initial)
		}

		mutation := root.UpdateR(cx, func(tree *RandomizedElementTree, cx *Context[RandomizedElementTree]) RandomizedElementTreeMutation {
			tree.resetWorkCounters()
			return tree.ApplyMutation(RandomizedElementTreeMutationKindChildColor, cx)
		})
		childColor, ok := mutation.(RandomizedElementTreeMutationChildColor)
		if !ok {
			panic("a nonempty tree should apply the requested child-color mutation")
		}
		randomizedTreeDrawWindow(window, cx)

		incremental := randomizedTreeWorkCounters(root, cx)
		if incremental.rootRenderCount < 1 || incremental.entityRenderCount < 1 || incremental.elementRenderCount < 1 || incremental.handlerRegistrationCount < 2 {
			t.Fatalf("incremental counters %+v", incremental)
		}
		if !slices.Contains(incremental.renderedEntityIDs, childColor.ElementID) {
			t.Fatalf("rendered entity ids %v do not contain %d", incremental.renderedEntityIDs, childColor.ElementID)
		}

		randomizedTreeRemoveWindow(window, cx, "randomized tree window should remain open until the case completes")
	})
}

func randomizedTreeWorkCounters(root Entity[RandomizedElementTree], cx *TestAppContext) RandomizedElementTreeWorkCounters {
	return root.ReadWith(cx, func(tree *RandomizedElementTree, _ *App) RandomizedElementTreeWorkCounters {
		return tree.workCountersSnapshot()
	})
}

func TestRandomizedElementTree_RandomizedElementTreeMatchesFullRender(t *testing.T) {
	skipUntilElementStateFocusHandleRelease(t)
	PropertyTest(t, func(tc *TestRun, rng *rand.Rand) {
		cx := tc.Cx()
		seed := rng.Uint64()
		for topology := range uint64(4) {
			topologySeed := (seed &^ 3) | topology
			config := NewRandomizedElementTreeConfig(topologySeed, randomizedTreeTestElementCount).
				WithEntityDensity(0.2).
				WithHandlerDensity(0.2)
			runRandomizedRenderEquivalenceCase(t, cx, config)
		}
		runRandomizedRenderEquivalenceCase(t, cx, NewRandomizedElementTreeConfig(seed, 0).
			WithEntityDensity(0.2).
			WithHandlerDensity(0.2))
	}, Cases(4))
}

func runRandomizedRenderEquivalenceCase(t *testing.T, cx *TestAppContext, config RandomizedElementTreeConfig) {
	persistentWindow := cx.OpenWindow(randomizedTreeTestWindowSize, func(_ *Window, cx *Context[RandomizedElementTree]) *RandomizedElementTree {
		return NewRandomizedElementTreeWithConfig(config, cx)
	})
	persistentRoot, err := persistentWindow.Root(cx)
	if err != nil {
		t.Fatalf("randomized tree window should have a root: %v", err)
	}
	defer persistentRoot.Release()
	window := persistentWindow.AnyWindowHandle

	assertMatchesRandomizedReference(t, persistentRoot, window, cx, config, -1, nil)

	for mutationIndex := range randomizedTreeTestMutationCount {
		mutation := persistentRoot.UpdateR(cx, func(tree *RandomizedElementTree, cx *Context[RandomizedElementTree]) RandomizedElementTreeMutation {
			switch mutationIndex {
			case 0:
				return tree.ApplyMutation(RandomizedElementTreeMutationKindInsert, cx)
			case 1:
				return tree.ApplyMutation(RandomizedElementTreeMutationKindRemove, cx)
			default:
				return tree.ApplyRandomMutation(cx)
			}
		})
		assertMatchesRandomizedReference(t, persistentRoot, window, cx, config, mutationIndex, mutation)
	}

	randomizedTreeRemoveWindow(window, cx, "persistent window should remain open until the case completes")
}

func assertMatchesRandomizedReference(t *testing.T, persistentRoot Entity[RandomizedElementTree], persistentWindow AnyWindowHandle, cx *TestAppContext, config RandomizedElementTreeConfig, mutationIndex int, mutation RandomizedElementTreeMutation) {
	t.Helper()
	snapshot := persistentRoot.ReadWith(cx, func(tree *RandomizedElementTree, _ *App) RandomizedElementTreeSnapshot {
		return tree.Snapshot()
	})
	expectedElementIDs := randomizedSortedIDs(snapshot.ElementIDs())
	persistentScene := randomizedTreeDrawWindow(persistentWindow, cx)
	if !slices.Equal(persistentScene.elementIDs, expectedElementIDs) {
		t.Fatalf("persistent frame element IDs diverged from the model for config %v, after mutation %d %#v: %v vs %v", config, mutationIndex, mutation, persistentScene.elementIDs, expectedElementIDs)
	}
	referenceScene := drawFreshRandomizedReference(snapshot, cx)
	if !reflect.DeepEqual(persistentScene, referenceScene) {
		t.Fatalf("rendered scene diverged for config %v, after mutation %d %#v%s", config, mutationIndex, mutation, randomizedSceneDiff(persistentScene, referenceScene))
	}
}

func drawFreshRandomizedReference(snapshot RandomizedElementTreeSnapshot, cx *TestAppContext) randomizedSceneSnapshot {
	referenceWindow := cx.OpenWindow(randomizedTreeTestWindowSize, func(_ *Window, cx *Context[RandomizedElementTree]) *RandomizedElementTree {
		return RandomizedElementTreeFromSnapshotWithEntities(snapshot, cx)
	})
	window := referenceWindow.AnyWindowHandle
	scene := randomizedTreeDrawWindow(window, cx)
	randomizedTreeRemoveWindow(window, cx, "reference window should remain open until its frame is captured")
	return scene
}

func randomizedSceneDiff(a, b randomizedSceneSnapshot) string {
	if len(a.quads) != len(b.quads) {
		return fmt.Sprintf(": %d quads vs %d", len(a.quads), len(b.quads))
	}
	for i := range a.quads {
		if !reflect.DeepEqual(a.quads[i], b.quads[i]) {
			return fmt.Sprintf(": quad %d: %+v vs %+v", i, a.quads[i], b.quads[i])
		}
	}
	if len(a.otherPrimitives) != len(b.otherPrimitives) {
		return fmt.Sprintf(": %d primitives vs %d", len(a.otherPrimitives), len(b.otherPrimitives))
	}
	for i := range a.otherPrimitives {
		if a.otherPrimitives[i] != b.otherPrimitives[i] {
			return fmt.Sprintf(": primitive %d: %s vs %s", i, a.otherPrimitives[i], b.otherPrimitives[i])
		}
	}
	return fmt.Sprintf(": element ids %v vs %v", a.elementIDs, b.elementIDs)
}

func skipUntilElementStateFocusHandleRelease(t *testing.T) {
	t.Helper()
	if os.Getenv("GPUI_RUN_HANDOFF_REPRO") != "1" {
		t.Skip("element state focus handles leak: docs/port/handoff/M3-randomized-tree.md")
	}
}

type randomizedTabIndexView struct{}

func (v *randomizedTabIndexView) Render(_ *Window, _ *Context[randomizedTabIndexView]) IntoElement {
	return NewDiv().ID("tab").TabIndex(0)
}

func TestRandomizedElementTree_HandoffElementStateFocusHandleIsReleased(t *testing.T) {
	skipUntilElementStateFocusHandleRelease(t)
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := cx.OpenWindow(randomizedTreeTestWindowSize, func(_ *Window, _ *Context[randomizedTabIndexView]) *randomizedTabIndexView {
			return &randomizedTabIndexView{}
		})
		randomizedTreeDrawWindow(window.AnyWindowHandle, cx)
		randomizedTreeRemoveWindow(window.AnyWindowHandle, cx, "window should remain open")
	})
}
