package gpui

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"
)

// RandomizedElementTreeAction is dispatched to no one: the action generated elements register
// handlers for. It is not registered, so it adds no action namespace.
type RandomizedElementTreeAction struct{}

// Name returns the action's name.
func (RandomizedElementTreeAction) Name() string {
	return "randomized_element_tree::RandomizedElementTreeAction"
}

// BoxedClone clones the action.
func (a RandomizedElementTreeAction) BoxedClone() Action {
	return a
}

// PartialEq reports whether other is an equal RandomizedElementTreeAction.
func (a RandomizedElementTreeAction) PartialEq(other Action) bool {
	o, ok := other.(RandomizedElementTreeAction)
	return ok && a == o
}

// RandomizedElementTreeTopology is the initial shape generated for a randomized element tree.
type RandomizedElementTreeTopology uint8

const (
	// RandomizedElementTreeTopologyWide makes every generated element a direct child of the root.
	RandomizedElementTreeTopologyWide RandomizedElementTreeTopology = iota
	// RandomizedElementTreeTopologyDeep makes every generated element the child of the previous one.
	RandomizedElementTreeTopologyDeep
	// RandomizedElementTreeTopologyMixed makes generated elements choose parents throughout the tree.
	RandomizedElementTreeTopologyMixed
	// RandomizedElementTreeTopologyNarrow makes generated elements usually extend one deep branch,
	// with occasional root children.
	RandomizedElementTreeTopologyNarrow
)

// RandomizedElementTreeTopologyAll lists every topology.
var RandomizedElementTreeTopologyAll = [4]RandomizedElementTreeTopology{
	RandomizedElementTreeTopologyWide,
	RandomizedElementTreeTopologyDeep,
	RandomizedElementTreeTopologyMixed,
	RandomizedElementTreeTopologyNarrow,
}

// String formats the topology like upstream's Debug.
func (t RandomizedElementTreeTopology) String() string {
	switch t {
	case RandomizedElementTreeTopologyWide:
		return "Wide"
	case RandomizedElementTreeTopologyDeep:
		return "Deep"
	case RandomizedElementTreeTopologyMixed:
		return "Mixed"
	default:
		return "Narrow"
	}
}

// IsTall reports whether the tree's depth grows with its element count.
func (t RandomizedElementTreeTopology) IsTall() bool {
	return t == RandomizedElementTreeTopologyDeep || t == RandomizedElementTreeTopologyNarrow
}

func randomizedElementTreeTopologyFromSeed(seed uint64) RandomizedElementTreeTopology {
	switch seed % 4 {
	case 0:
		return RandomizedElementTreeTopologyWide
	case 1:
		return RandomizedElementTreeTopologyDeep
	case 2:
		return RandomizedElementTreeTopologyMixed
	default:
		return RandomizedElementTreeTopologyNarrow
	}
}

// RandomizedElementTreeConfig configures a reproducible randomized element-tree workload.
type RandomizedElementTreeConfig struct {
	seed             uint64
	elementCount     int
	topology         RandomizedElementTreeTopology
	entityDensity    float64
	handlerDensity   float64
	animationDensity float64
	cachedViews      bool
	hoverStyles      bool
	paths            bool
}

// NewRandomizedElementTreeConfig creates an element-only workload whose topology is selected by seed.
func NewRandomizedElementTreeConfig(seed uint64, elementCount int) RandomizedElementTreeConfig {
	return RandomizedElementTreeConfig{
		seed:         seed,
		elementCount: elementCount,
		topology:     randomizedElementTreeTopologyFromSeed(seed),
		paths:        true,
	}
}

// WithCachedViews uses definite-size cached entity boundaries.
func (c RandomizedElementTreeConfig) WithCachedViews(cached bool) RandomizedElementTreeConfig {
	c.cachedViews = cached
	return c
}

// WithAnimationDensity sets the probability that each entity-backed element shows a looping
// animation.
func (c RandomizedElementTreeConfig) WithAnimationDensity(density float64) RandomizedElementTreeConfig {
	c.animationDensity = normalizedDensity(density)
	return c
}

// WithHoverStyles gives every element a hover style and a group-hover style keyed to the root.
func (c RandomizedElementTreeConfig) WithHoverStyles(hoverStyles bool) RandomizedElementTreeConfig {
	c.hoverStyles = hoverStyles
	return c
}

// WithPaths sets whether elements paint vector paths.
func (c RandomizedElementTreeConfig) WithPaths(paths bool) RandomizedElementTreeConfig {
	c.paths = paths
	return c
}

// WithTopology selects the initial tree topology independently of the seed.
func (c RandomizedElementTreeConfig) WithTopology(topology RandomizedElementTreeTopology) RandomizedElementTreeConfig {
	c.topology = topology
	return c
}

// WithEntityDensity sets the probability that each generated element is backed by an entity.
func (c RandomizedElementTreeConfig) WithEntityDensity(density float64) RandomizedElementTreeConfig {
	c.entityDensity = normalizedDensity(density)
	return c
}

// WithHandlerDensity sets the probability of each generated interaction feature on an element.
func (c RandomizedElementTreeConfig) WithHandlerDensity(density float64) RandomizedElementTreeConfig {
	c.handlerDensity = normalizedDensity(density)
	return c
}

// Seed returns the seed used for generation and subsequent random mutations.
func (c RandomizedElementTreeConfig) Seed() uint64 {
	return c.seed
}

// ElementCount returns the requested number of initial descendants.
func (c RandomizedElementTreeConfig) ElementCount() int {
	return c.elementCount
}

// Topology returns the selected initial topology.
func (c RandomizedElementTreeConfig) Topology() RandomizedElementTreeTopology {
	return c.topology
}

// EntityDensity returns the probability that a generated element is backed by an entity.
func (c RandomizedElementTreeConfig) EntityDensity() float64 {
	return c.entityDensity
}

// HandlerDensity returns the probability of each generated interaction feature.
func (c RandomizedElementTreeConfig) HandlerDensity() float64 {
	return c.handlerDensity
}

// String formats the config like upstream's Debug.
func (c RandomizedElementTreeConfig) String() string {
	return fmt.Sprintf("RandomizedElementTreeConfig { seed: %d, element_count: %d, topology: %v, entity_density: %v, handler_density: %v, animation_density: %v, cached_views: %v, hover_styles: %v, paths: %v }",
		c.seed, c.elementCount, c.topology, c.entityDensity, c.handlerDensity, c.animationDensity, c.cachedViews, c.hoverStyles, c.paths)
}

type randomizedFloatRange struct {
	start, end float64
}

type randomizedIntRange struct {
	start, end int
}

// RandomizedElementTreeBounds are the bounds an RNG is drawn against to produce a
// RandomizedElementTreeConfig. Ranges are inclusive.
type RandomizedElementTreeBounds struct {
	topologies     []RandomizedElementTreeTopology
	elementCount   randomizedIntRange
	entityDensity  randomizedFloatRange
	handlerDensity randomizedFloatRange
}

// NewRandomizedElementTreeBounds returns trees of any topology with elementCountStart to
// elementCountEnd (inclusive) elements, no entities and no handlers.
func NewRandomizedElementTreeBounds(elementCountStart, elementCountEnd int) RandomizedElementTreeBounds {
	return RandomizedElementTreeBounds{
		topologies:   RandomizedElementTreeTopologyAll[:],
		elementCount: randomizedIntRange{elementCountStart, elementCountEnd},
	}
}

// WithTopologies restricts the topologies a seed may pick. An empty list keeps them all.
func (b RandomizedElementTreeBounds) WithTopologies(topologies ...RandomizedElementTreeTopology) RandomizedElementTreeBounds {
	if len(topologies) != 0 {
		b.topologies = slices.Clone(topologies)
	}
	return b
}

// WithEntityDensity sets the inclusive range a seed draws the share of entity-backed elements from.
func (b RandomizedElementTreeBounds) WithEntityDensity(start, end float64) RandomizedElementTreeBounds {
	b.entityDensity = randomizedFloatRange{normalizedDensity(start), normalizedDensity(end)}
	return b
}

// WithHandlerDensity sets the inclusive range a seed draws the per-feature handler probability from.
func (b RandomizedElementTreeBounds) WithHandlerDensity(start, end float64) RandomizedElementTreeBounds {
	b.handlerDensity = randomizedFloatRange{normalizedDensity(start), normalizedDensity(end)}
	return b
}

// Sample draws one config from these bounds; the config's own seed is drawn from rng too.
func (b RandomizedElementTreeBounds) Sample(rng *rand.Rand) RandomizedElementTreeConfig {
	topology := b.topologies[rng.IntN(len(b.topologies))]
	elementCount := randomIntInclusive(rng, b.elementCount.start, b.elementCount.end)
	return NewRandomizedElementTreeConfig(rng.Uint64(), elementCount).
		WithTopology(topology).
		WithEntityDensity(sampleRange(b.entityDensity, rng)).
		WithHandlerDensity(sampleRange(b.handlerDensity, rng))
}

func sampleRange(r randomizedFloatRange, rng *rand.Rand) float64 {
	if r.start >= r.end {
		return r.start
	}
	return r.start + rng.Float64()*(r.end-r.start)
}

var randomizedElementTreeRecordWork bool

// RandomizedElementTreeWorkCounters is the work performed while converting randomized entities
// and elements into an element tree. It is recorded only in this package's tests.
type RandomizedElementTreeWorkCounters struct {
	rootRenderCount          int
	entityRenderCount        int
	elementRenderCount       int
	handlerRegistrationCount int
	renderedEntityIDs        []uint64
}

func (w *RandomizedElementTreeWorkCounters) recordRootRender() {
	if randomizedElementTreeRecordWork {
		w.rootRenderCount++
	}
}

func (w *RandomizedElementTreeWorkCounters) recordEntityRender(elementID uint64) {
	if randomizedElementTreeRecordWork {
		w.entityRenderCount++
		w.renderedEntityIDs = append(w.renderedEntityIDs, elementID)
	}
}

func (w *RandomizedElementTreeWorkCounters) recordElementRender(handlerCount int) {
	if randomizedElementTreeRecordWork {
		w.elementRenderCount++
		w.handlerRegistrationCount += handlerCount
	}
}

func (w *RandomizedElementTreeWorkCounters) reset() {
	w.rootRenderCount = 0
	w.entityRenderCount = 0
	w.elementRenderCount = 0
	w.handlerRegistrationCount = 0
	w.renderedEntityIDs = w.renderedEntityIDs[:0]
}

func (w *RandomizedElementTreeWorkCounters) clone() RandomizedElementTreeWorkCounters {
	c := *w
	c.renderedEntityIDs = slices.Clone(w.renderedEntityIDs)
	return c
}

// RandomizedElementTreeSnapshot is a cloneable description of a randomized element tree's
// renderable state.
type RandomizedElementTreeSnapshot struct {
	root             randomizedElementStyle
	children         []randomizedElementNode
	nextElementID    uint64
	renderOptions    randomizedRenderOptions
	pathsByElementID map[uint64][]int
}

// Clone deep-copies the snapshot.
func (s *RandomizedElementTreeSnapshot) Clone() RandomizedElementTreeSnapshot {
	c := *s
	c.children = cloneRandomizedNodes(s.children)
	c.pathsByElementID = make(map[uint64][]int, len(s.pathsByElementID))
	for id, path := range s.pathsByElementID {
		c.pathsByElementID[id] = slices.Clone(path)
	}
	return c
}

// DescendantCount returns the number of randomized elements below the persistent root.
func (s *RandomizedElementTreeSnapshot) DescendantCount() int {
	count := 0
	stack := randomizedNodeRefs(s.children)
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		stack = append(stack, randomizedNodeRefs(node.children)...)
	}
	return count
}

// MaxDepth returns the deepest randomized element below the persistent root; a snapshot with no
// descendants has depth zero.
func (s *RandomizedElementTreeSnapshot) MaxDepth() int {
	type entry struct {
		node  *randomizedElementNode
		depth int
	}
	maxDepth := 0
	stack := make([]entry, 0, len(s.children))
	for i := range s.children {
		stack = append(stack, entry{&s.children[i], 1})
	}
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		maxDepth = max(maxDepth, e.depth)
		for i := range e.node.children {
			stack = append(stack, entry{&e.node.children[i], e.depth + 1})
		}
	}
	return maxDepth
}

// ElementIDs returns the stable element IDs in depth-first order.
func (s *RandomizedElementTreeSnapshot) ElementIDs() []uint64 {
	paths := randomizedNodePaths(s.children)
	ids := make([]uint64, 0, len(paths))
	for _, path := range paths {
		if node := randomizedNodeAtPath(s.children, path); node != nil {
			ids = append(ids, node.id)
		}
	}
	return ids
}

// EntityCount returns the number of descendants backed by persistent child entities.
func (s *RandomizedElementTreeSnapshot) EntityCount() int {
	count := 0
	stack := randomizedNodeRefs(s.children)
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node.isEntity {
			count++
		}
		stack = append(stack, randomizedNodeRefs(node.children)...)
	}
	return count
}

// HandlerCount returns the number of interaction handlers in the generated tree.
func (s *RandomizedElementTreeSnapshot) HandlerCount() int {
	count := 0
	stack := randomizedNodeRefs(s.children)
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count += node.features.handlerCount()
		stack = append(stack, randomizedNodeRefs(node.children)...)
	}
	return count
}

func (s *RandomizedElementTreeSnapshot) rebuildElementPaths() {
	s.pathsByElementID = map[uint64][]int{}
	for _, path := range randomizedNodePaths(s.children) {
		if node := randomizedNodeAtPath(s.children, path); node != nil {
			s.pathsByElementID[node.id] = path
		}
	}
}

func (s *RandomizedElementTreeSnapshot) node(elementID uint64) *randomizedElementNode {
	path, ok := s.pathsByElementID[elementID]
	if !ok {
		return nil
	}
	return randomizedNodeAtPath(s.children, path)
}

// RandomizedElementTreeMutationKind is a requested class of deterministic randomized-tree mutation.
type RandomizedElementTreeMutationKind uint8

const (
	// RandomizedElementTreeMutationKindRootColor changes the root background color.
	RandomizedElementTreeMutationKindRootColor RandomizedElementTreeMutationKind = iota
	// RandomizedElementTreeMutationKindRootOpacity changes the root opacity.
	RandomizedElementTreeMutationKindRootOpacity
	// RandomizedElementTreeMutationKindRootBorder changes the root border.
	RandomizedElementTreeMutationKindRootBorder
	// RandomizedElementTreeMutationKindRootBounds changes the root width and height.
	RandomizedElementTreeMutationKindRootBounds
	// RandomizedElementTreeMutationKindRootClipping toggles clipping on the root.
	RandomizedElementTreeMutationKindRootClipping
	// RandomizedElementTreeMutationKindRootVisibility toggles root visibility.
	RandomizedElementTreeMutationKindRootVisibility
	// RandomizedElementTreeMutationKindChildColor changes a seeded random descendant's colors.
	RandomizedElementTreeMutationKindChildColor
	// RandomizedElementTreeMutationKindChildOpacity changes a seeded random descendant's opacity.
	RandomizedElementTreeMutationKindChildOpacity
	// RandomizedElementTreeMutationKindChildBorder changes a seeded random descendant's border.
	RandomizedElementTreeMutationKindChildBorder
	// RandomizedElementTreeMutationKindChildBounds changes a seeded random descendant's size.
	RandomizedElementTreeMutationKindChildBounds
	// RandomizedElementTreeMutationKindChildClipping toggles clipping on a seeded random descendant.
	RandomizedElementTreeMutationKindChildClipping
	// RandomizedElementTreeMutationKindChildVisibility toggles visibility on a seeded random descendant.
	RandomizedElementTreeMutationKindChildVisibility
	// RandomizedElementTreeMutationKindChildText changes a seeded random descendant's text.
	RandomizedElementTreeMutationKindChildText
	// RandomizedElementTreeMutationKindReorder reorders a seeded random descendant among its siblings.
	RandomizedElementTreeMutationKindReorder
	// RandomizedElementTreeMutationKindInsert inserts a generated descendant below a seeded random parent.
	RandomizedElementTreeMutationKindInsert
	// RandomizedElementTreeMutationKindRemove removes a seeded random descendant and its subtree.
	RandomizedElementTreeMutationKindRemove
	// RandomizedElementTreeMutationKindRemoveLeaf removes a seeded random childless descendant.
	RandomizedElementTreeMutationKindRemoveLeaf
)

// RandomizedElementTreeMutationKindAll lists every supported mutation kind.
var RandomizedElementTreeMutationKindAll = [17]RandomizedElementTreeMutationKind{
	RandomizedElementTreeMutationKindRootColor,
	RandomizedElementTreeMutationKindRootOpacity,
	RandomizedElementTreeMutationKindRootBorder,
	RandomizedElementTreeMutationKindRootBounds,
	RandomizedElementTreeMutationKindRootClipping,
	RandomizedElementTreeMutationKindRootVisibility,
	RandomizedElementTreeMutationKindChildColor,
	RandomizedElementTreeMutationKindChildOpacity,
	RandomizedElementTreeMutationKindChildBorder,
	RandomizedElementTreeMutationKindChildBounds,
	RandomizedElementTreeMutationKindChildClipping,
	RandomizedElementTreeMutationKindChildVisibility,
	RandomizedElementTreeMutationKindChildText,
	RandomizedElementTreeMutationKindReorder,
	RandomizedElementTreeMutationKindInsert,
	RandomizedElementTreeMutationKindRemove,
	RandomizedElementTreeMutationKindRemoveLeaf,
}

// RandomizedElementTreeMutation is a deterministic mutation applied to a RandomizedElementTree:
// one of the RandomizedElementTreeMutation* variant structs.
type RandomizedElementTreeMutation interface {
	isRandomizedElementTreeMutation()
}

// RandomizedElementTreeMutationRootColor changed the root background color.
type RandomizedElementTreeMutationRootColor struct{}

// RandomizedElementTreeMutationRootOpacity changed the root opacity.
type RandomizedElementTreeMutationRootOpacity struct{}

// RandomizedElementTreeMutationRootBorder changed the root border.
type RandomizedElementTreeMutationRootBorder struct{}

// RandomizedElementTreeMutationRootBounds changed the root width and height.
type RandomizedElementTreeMutationRootBounds struct{}

// RandomizedElementTreeMutationRootClipping toggled clipping on the root.
type RandomizedElementTreeMutationRootClipping struct{}

// RandomizedElementTreeMutationRootVisibility toggled root visibility.
type RandomizedElementTreeMutationRootVisibility struct{}

// RandomizedElementTreeMutationChildColor changed a descendant's colors.
type RandomizedElementTreeMutationChildColor struct{ ElementID uint64 }

// RandomizedElementTreeMutationChildOpacity changed a descendant's opacity.
type RandomizedElementTreeMutationChildOpacity struct{ ElementID uint64 }

// RandomizedElementTreeMutationChildBorder changed a descendant's border.
type RandomizedElementTreeMutationChildBorder struct{ ElementID uint64 }

// RandomizedElementTreeMutationChildBounds changed a descendant's width and height.
type RandomizedElementTreeMutationChildBounds struct{ ElementID uint64 }

// RandomizedElementTreeMutationChildClipping toggled clipping on a descendant.
type RandomizedElementTreeMutationChildClipping struct{ ElementID uint64 }

// RandomizedElementTreeMutationChildVisibility toggled a descendant's visibility.
type RandomizedElementTreeMutationChildVisibility struct{ ElementID uint64 }

// RandomizedElementTreeMutationChildText changed a descendant's text.
type RandomizedElementTreeMutationChildText struct{ ElementID uint64 }

// RandomizedElementTreeMutationReordered moved an element to a different position among its siblings.
type RandomizedElementTreeMutationReordered struct {
	ElementID uint64
	ParentID  Option[uint64]
	OldIndex  int
	NewIndex  int
}

// RandomizedElementTreeMutationInserted inserted a newly generated element.
type RandomizedElementTreeMutationInserted struct {
	ElementID uint64
	ParentID  Option[uint64]
	Index     int
}

// RandomizedElementTreeMutationRemoved removed an element and all of its descendants.
type RandomizedElementTreeMutationRemoved struct {
	ElementID           uint64
	ParentID            Option[uint64]
	Index               int
	RemovedElementCount int
}

func (RandomizedElementTreeMutationRootColor) isRandomizedElementTreeMutation()       {}
func (RandomizedElementTreeMutationRootOpacity) isRandomizedElementTreeMutation()     {}
func (RandomizedElementTreeMutationRootBorder) isRandomizedElementTreeMutation()      {}
func (RandomizedElementTreeMutationRootBounds) isRandomizedElementTreeMutation()      {}
func (RandomizedElementTreeMutationRootClipping) isRandomizedElementTreeMutation()    {}
func (RandomizedElementTreeMutationRootVisibility) isRandomizedElementTreeMutation()  {}
func (RandomizedElementTreeMutationChildColor) isRandomizedElementTreeMutation()      {}
func (RandomizedElementTreeMutationChildOpacity) isRandomizedElementTreeMutation()    {}
func (RandomizedElementTreeMutationChildBorder) isRandomizedElementTreeMutation()     {}
func (RandomizedElementTreeMutationChildBounds) isRandomizedElementTreeMutation()     {}
func (RandomizedElementTreeMutationChildClipping) isRandomizedElementTreeMutation()   {}
func (RandomizedElementTreeMutationChildVisibility) isRandomizedElementTreeMutation() {}
func (RandomizedElementTreeMutationChildText) isRandomizedElementTreeMutation()       {}
func (RandomizedElementTreeMutationReordered) isRandomizedElementTreeMutation()       {}
func (RandomizedElementTreeMutationInserted) isRandomizedElementTreeMutation()        {}
func (RandomizedElementTreeMutationRemoved) isRandomizedElementTreeMutation()         {}

// ChangeLocality is where the elements changed by one frame sit relative to each other.
type ChangeLocality uint8

const (
	// ChangeLocalityLocalized changes elements inside one random entity's subtree.
	ChangeLocalityLocalized ChangeLocality = iota
	// ChangeLocalitySpread changes elements anywhere in the tree, uniformly.
	ChangeLocalitySpread
)

// RecoloredChildren is what RandomizedElementTree.RecolorChildren changed.
type RecoloredChildren struct {
	// Recolored is the number of descendants whose background changed.
	Recolored int
	// Notified is the number of entity boundaries (or the root) notified, each once.
	Notified int
}

type randomizedElementEntities struct {
	entities map[uint64]Entity[randomizedElementEntity]
	dropped  bool
}

// RandomizedElementTree is a persistent root view for randomized GPUI element-tree tests and
// benchmarks.
type RandomizedElementTree struct {
	snapshot      *RandomizedElementTreeSnapshot
	entities      *randomizedElementEntities
	scrollHandles map[uint64]ScrollHandle
	workCounters  *RandomizedElementTreeWorkCounters
	config        RandomizedElementTreeConfig
	rng           *rand.Rand
}

// NewRandomizedElementTree generates a tree with elementCount descendants and a mutation stream
// from seed.
func NewRandomizedElementTree(seed uint64, elementCount int) *RandomizedElementTree {
	return randomizedElementTreeFromConfig(NewRandomizedElementTreeConfig(seed, elementCount))
}

// NewRandomizedElementTreeWithConfig generates a configured tree and creates its persistent child
// entities.
func NewRandomizedElementTreeWithConfig(config RandomizedElementTreeConfig, cx *Context[RandomizedElementTree]) *RandomizedElementTree {
	tree := randomizedElementTreeFromConfig(config)
	tree.synchronizeEntities(cx)
	return tree
}

func newRandomizedRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0))
}

func randomizedElementTreeFromConfig(config RandomizedElementTreeConfig) *RandomizedElementTree {
	rng := newRandomizedRNG(config.seed)
	var children []randomizedElementNode
	var deepestPath []int

	for elementIndex := range config.elementCount {
		var parentPath []int
		switch config.topology {
		case RandomizedElementTreeTopologyWide:
		case RandomizedElementTreeTopologyDeep:
			parentPath = slices.Clone(deepestPath)
		case RandomizedElementTreeTopologyMixed:
			parentPath = randomParentPath(children, rng)
		case RandomizedElementTreeTopologyNarrow:
			if randomBool(rng, 0.8) {
				parentPath = slices.Clone(deepestPath)
			}
		}
		node := randomRandomizedElementNode(uint64(elementIndex), &config, rng)
		_, deepestPath = insertRandomizedNode(&children, &parentPath, node)
	}

	snapshot := &RandomizedElementTreeSnapshot{
		root:          randomRootStyle(rng),
		children:      children,
		nextElementID: uint64(config.elementCount),
		renderOptions: randomizedRenderOptions{
			cachedViews: config.cachedViews,
			hoverStyles: config.hoverStyles,
			paths:       config.paths,
		},
	}
	snapshot.rebuildElementPaths()

	return &RandomizedElementTree{
		snapshot:      snapshot,
		entities:      &randomizedElementEntities{entities: map[uint64]Entity[randomizedElementEntity]{}},
		scrollHandles: map[uint64]ScrollHandle{},
		workCounters:  &RandomizedElementTreeWorkCounters{},
		config:        config,
		rng:           rng,
	}
}

// RandomizedElementTreeFromSnapshot reconstructs a tree from renderable state captured by Snapshot.
func RandomizedElementTreeFromSnapshot(snapshot RandomizedElementTreeSnapshot) *RandomizedElementTree {
	return &RandomizedElementTree{
		config:        NewRandomizedElementTreeConfig(0, snapshot.DescendantCount()),
		snapshot:      &snapshot,
		entities:      &randomizedElementEntities{entities: map[uint64]Entity[randomizedElementEntity]{}},
		scrollHandles: map[uint64]ScrollHandle{},
		workCounters:  &RandomizedElementTreeWorkCounters{},
		rng:           newRandomizedRNG(0),
	}
}

// RandomizedElementTreeFromSnapshotWithEntities reconstructs a snapshot while preserving its
// randomized entity boundaries.
func RandomizedElementTreeFromSnapshotWithEntities(snapshot RandomizedElementTreeSnapshot, cx *Context[RandomizedElementTree]) *RandomizedElementTree {
	tree := RandomizedElementTreeFromSnapshot(snapshot)
	tree.synchronizeEntities(cx)
	return tree
}

// Drop releases the tree's persistent child entities when the tree's entity is released.
func (t *RandomizedElementTree) Drop(cx *App) {
	for _, entity := range t.entities.entities {
		entity.Release()
	}
	clear(t.entities.entities)
	t.entities.dropped = true
}

// Snapshot captures all state that affects rendering and future element IDs.
func (t *RandomizedElementTree) Snapshot() RandomizedElementTreeSnapshot {
	return t.snapshot.Clone()
}

func (t *RandomizedElementTree) resetWorkCounters() {
	t.workCounters.reset()
}

func (t *RandomizedElementTree) workCountersSnapshot() RandomizedElementTreeWorkCounters {
	return t.workCounters.clone()
}

// Mutate applies one seeded random style or structural mutation without notifying persistent
// child entities.
func (t *RandomizedElementTree) Mutate() RandomizedElementTreeMutation {
	kind := RandomizedElementTreeMutationKindAll[t.rng.IntN(len(RandomizedElementTreeMutationKindAll))]
	return t.applyMutationWithoutNotification(kind).mutation
}

// ApplyRandomMutation applies one seeded random mutation and notifies its nearest persistent
// entity boundary.
func (t *RandomizedElementTree) ApplyRandomMutation(cx *Context[RandomizedElementTree]) RandomizedElementTreeMutation {
	kind := RandomizedElementTreeMutationKindAll[t.rng.IntN(len(RandomizedElementTreeMutationKindAll))]
	return t.ApplyMutation(kind, cx)
}

// ApplyMutation applies a selected mutation kind and notifies its nearest persistent entity
// boundary.
func (t *RandomizedElementTree) ApplyMutation(kind RandomizedElementTreeMutationKind, cx *Context[RandomizedElementTree]) RandomizedElementTreeMutation {
	applied := t.applyMutationWithoutNotification(kind)
	if applied.synchronizeEntities {
		t.synchronizeEntities(cx)
	}
	t.notifyOwner(applied.ownerEntityID, cx)
	return applied.mutation
}

// RandomElementID picks a seeded random element at any depth; ok is false if the tree is empty.
func (t *RandomizedElementTree) RandomElementID() (uint64, bool) {
	path, ok := randomNodePath(t.snapshot.children, t.rng)
	if !ok {
		return 0, false
	}
	node := randomizedNodeAtPath(t.snapshot.children, path)
	if node == nil {
		return 0, false
	}
	return node.id, true
}

// AlternateElementColor alternates an element's background between two colors, notifying its
// nearest persistent entity boundary. It returns false if the element is no longer in the tree.
func (t *RandomizedElementTree) AlternateElementColor(elementID uint64, cx *Context[RandomizedElementTree]) bool {
	snapshot := t.snapshot
	path, ok := snapshot.pathsByElementID[elementID]
	if !ok {
		return false
	}
	owner := nearestEntityID(snapshot.children, path)
	node := randomizedNodeAtPath(snapshot.children, path)
	if node == nil {
		return false
	}
	first := NewHsla(0., 1., 0.5, 1.)
	second := NewHsla(0.5, 1., 0.5, 1.)
	if node.style.background == first {
		node.style.background = second
	} else {
		node.style.background = first
	}
	t.notifyOwner(owner, cx)
	return true
}

// InsertChild generates and inserts an element below a seeded random parent without
// synchronizing or notifying persistent child entities.
func (t *RandomizedElementTree) InsertChild() RandomizedElementTreeMutation {
	return t.insertChildInternal().mutation
}

// RemoveChild removes a seeded random element and its subtree without synchronizing or notifying
// persistent child entities; ok is false if nothing was removed.
func (t *RandomizedElementTree) RemoveChild() (RandomizedElementTreeMutation, bool) {
	applied, ok := t.removeChildInternal(false)
	if !ok {
		return nil, false
	}
	return applied.mutation, true
}

func (t *RandomizedElementTree) applyMutationWithoutNotification(kind RandomizedElementTreeMutationKind) appliedRandomizedElementTreeMutation {
	s := t.snapshot
	switch kind {
	case RandomizedElementTreeMutationKindRootColor:
		s.root.background = randomColor(t.rng)
		return appliedRootMutation(RandomizedElementTreeMutationRootColor{})
	case RandomizedElementTreeMutationKindRootOpacity:
		s.root.opacity = randomOpacity(t.rng)
		return appliedRootMutation(RandomizedElementTreeMutationRootOpacity{})
	case RandomizedElementTreeMutationKindRootBorder:
		s.root.randomizeBorder(t.rng)
		return appliedRootMutation(RandomizedElementTreeMutationRootBorder{})
	case RandomizedElementTreeMutationKindRootBounds:
		s.root.width = randomF32(t.rng, 520.0, 980.0)
		s.root.height = randomF32(t.rng, 360.0, 760.0)
		return appliedRootMutation(RandomizedElementTreeMutationRootBounds{})
	case RandomizedElementTreeMutationKindRootClipping:
		s.root.clipped = !s.root.clipped
		return appliedRootMutation(RandomizedElementTreeMutationRootClipping{})
	case RandomizedElementTreeMutationKindRootVisibility:
		s.root.visible = !s.root.visible
		return appliedRootMutation(RandomizedElementTreeMutationRootVisibility{})
	case RandomizedElementTreeMutationKindChildColor:
		return t.mutateRandomChild(randomizedChildMutationColor)
	case RandomizedElementTreeMutationKindChildOpacity:
		return t.mutateRandomChild(randomizedChildMutationOpacity)
	case RandomizedElementTreeMutationKindChildBorder:
		return t.mutateRandomChild(randomizedChildMutationBorder)
	case RandomizedElementTreeMutationKindChildBounds:
		return t.mutateRandomChild(randomizedChildMutationBounds)
	case RandomizedElementTreeMutationKindChildClipping:
		return t.mutateRandomChild(randomizedChildMutationClipping)
	case RandomizedElementTreeMutationKindChildVisibility:
		return t.mutateRandomChild(randomizedChildMutationVisibility)
	case RandomizedElementTreeMutationKindChildText:
		return t.mutateRandomChild(randomizedChildMutationText)
	case RandomizedElementTreeMutationKindReorder:
		return t.reorderChild()
	case RandomizedElementTreeMutationKindInsert:
		return t.insertChildInternal()
	case RandomizedElementTreeMutationKindRemove:
		if applied, ok := t.removeChildInternal(false); ok {
			return applied
		}
		return t.insertChildInternal()
	default:
		if applied, ok := t.removeChildInternal(true); ok {
			return applied
		}
		return t.insertChildInternal()
	}
}

func (t *RandomizedElementTree) insertChildInternal() appliedRandomizedElementTreeMutation {
	s := t.snapshot
	elementID := s.nextElementID
	s.nextElementID++
	parentPath := randomParentPath(s.children, t.rng)
	parentID := randomizedNodeIDAtPath(s.children, parentPath)
	ownerEntityID := nearestEntityID(s.children, parentPath)
	node := randomRandomizedElementNode(elementID, &t.config, t.rng)
	index, _ := insertRandomizedNode(&s.children, &parentPath, node)
	s.rebuildElementPaths()

	return appliedRandomizedElementTreeMutation{
		mutation: RandomizedElementTreeMutationInserted{
			ElementID: elementID,
			ParentID:  parentID,
			Index:     index,
		},
		ownerEntityID:       ownerEntityID,
		synchronizeEntities: true,
	}
}

func (t *RandomizedElementTree) removeChildInternal(leavesOnly bool) (appliedRandomizedElementTreeMutation, bool) {
	s := t.snapshot
	var path []int
	var ok bool
	if leavesOnly {
		path, ok = randomLeafPath(s.children, t.rng)
	} else {
		path, ok = randomNodePath(s.children, t.rng)
	}
	if !ok || len(path) == 0 {
		return appliedRandomizedElementTreeMutation{}, false
	}
	index := path[len(path)-1]
	parentPath := path[:len(path)-1]
	parentID := randomizedNodeIDAtPath(s.children, parentPath)
	ownerEntityID := nearestEntityID(s.children, parentPath)
	siblings := randomizedChildrenAtPathMut(&s.children, parentPath)
	if siblings == nil || index >= len(*siblings) {
		return appliedRandomizedElementTreeMutation{}, false
	}

	removed := (*siblings)[index]
	*siblings = slices.Delete(*siblings, index, index+1)
	removedElementCount := removed.elementCount()
	elementID := removed.id
	s.rebuildElementPaths()
	removedNodes := []*randomizedElementNode{&removed}
	for len(removedNodes) > 0 {
		node := removedNodes[len(removedNodes)-1]
		removedNodes = removedNodes[:len(removedNodes)-1]
		delete(t.scrollHandles, node.id)
		removedNodes = append(removedNodes, randomizedNodeRefs(node.children)...)
	}
	return appliedRandomizedElementTreeMutation{
		mutation: RandomizedElementTreeMutationRemoved{
			ElementID:           elementID,
			ParentID:            parentID,
			Index:               index,
			RemovedElementCount: removedElementCount,
		},
		ownerEntityID:       ownerEntityID,
		synchronizeEntities: true,
	}, true
}

// RecolorChildren recolors count distinct seeded random descendants in one pass and notifies each
// affected entity boundary once (the root, for descendants owned by no entity).
func (t *RandomizedElementTree) RecolorChildren(count int, locality ChangeLocality, cx *Context[RandomizedElementTree]) RecoloredChildren {
	s := t.snapshot
	paths := randomizedNodePaths(s.children)
	count = min(count, len(paths))
	if locality == ChangeLocalityLocalized {
		var entityPaths [][]int
		for _, path := range paths {
			if node := randomizedNodeAtPath(s.children, path); node != nil && node.isEntity {
				entityPaths = append(entityPaths, path)
			}
		}
		i := t.rng.IntN(max(len(entityPaths), 1))
		if i < len(entityPaths) {
			entityPath := entityPaths[i]
			var inside, outside [][]int
			for _, path := range paths {
				if len(path) >= len(entityPath) && slices.Equal(path[:len(entityPath)], entityPath) {
					inside = append(inside, path)
				} else {
					outside = append(outside, path)
				}
			}
			paths = append(inside, outside...)
		}
	}
	sampleFrom := count
	if locality == ChangeLocalitySpread {
		sampleFrom = len(paths)
	}
	for index := range count {
		other := randomIntRange(t.rng, index, max(sampleFrom, index+1))
		paths[index], paths[other] = paths[other], paths[index]
	}
	owners := map[Option[uint64]]struct{}{}
	var ownerOrder []Option[uint64]
	for _, path := range paths[:count] {
		owner := nearestEntityID(s.children, path)
		if _, seen := owners[owner]; !seen {
			owners[owner] = struct{}{}
			ownerOrder = append(ownerOrder, owner)
		}
		if node := randomizedNodeAtPath(s.children, path); node != nil {
			node.style.background = randomColor(t.rng)
		}
	}
	for _, owner := range ownerOrder {
		t.notifyOwner(owner, cx)
	}
	return RecoloredChildren{Recolored: count, Notified: len(ownerOrder)}
}

func (t *RandomizedElementTree) mutateRandomChild(mutation randomizedChildMutation) appliedRandomizedElementTreeMutation {
	s := t.snapshot
	path, ok := randomNodePath(s.children, t.rng)
	if !ok {
		return t.insertChildInternal()
	}
	ownerEntityID := nearestEntityID(s.children, path)
	node := randomizedNodeAtPath(s.children, path)
	if node == nil {
		return t.insertChildInternal()
	}
	elementID := node.id

	var result RandomizedElementTreeMutation
	switch mutation {
	case randomizedChildMutationColor:
		node.style.background = randomColor(t.rng)
		result = RandomizedElementTreeMutationChildColor{ElementID: elementID}
	case randomizedChildMutationOpacity:
		node.style.opacity = randomOpacity(t.rng)
		result = RandomizedElementTreeMutationChildOpacity{ElementID: elementID}
	case randomizedChildMutationBorder:
		node.style.randomizeBorder(t.rng)
		result = RandomizedElementTreeMutationChildBorder{ElementID: elementID}
	case randomizedChildMutationBounds:
		if randomBool(t.rng, 0.02) {
			node.style.width = 0.0
		} else {
			node.style.width = randomF32(t.rng, 36.0, 220.0)
		}
		if randomBool(t.rng, 0.02) {
			node.style.height = 0.0
		} else {
			node.style.height = randomF32(t.rng, 28.0, 180.0)
		}
		result = RandomizedElementTreeMutationChildBounds{ElementID: elementID}
	case randomizedChildMutationClipping:
		node.style.clipped = !node.style.clipped
		result = RandomizedElementTreeMutationChildClipping{ElementID: elementID}
	case randomizedChildMutationVisibility:
		node.style.visible = !node.style.visible
		result = RandomizedElementTreeMutationChildVisibility{ElementID: elementID}
	default:
		node.style.label = (node.style.label + randomIntRange(t.rng, 1, len(randomizedLabels))) % len(randomizedLabels)
		result = RandomizedElementTreeMutationChildText{ElementID: elementID}
	}
	return appliedRandomizedElementTreeMutation{
		mutation:      result,
		ownerEntityID: ownerEntityID,
	}
}

func (t *RandomizedElementTree) reorderChild() appliedRandomizedElementTreeMutation {
	s := t.snapshot
	parentPaths := parentPathsWithMultipleChildren(s.children)
	if len(parentPaths) == 0 {
		return t.insertChildInternal()
	}

	parentPath := parentPaths[t.rng.IntN(len(parentPaths))]
	parentID := randomizedNodeIDAtPath(s.children, parentPath)
	ownerEntityID := nearestEntityID(s.children, parentPath)
	siblings := randomizedChildrenAtPathMut(&s.children, parentPath)
	if siblings == nil || len(*siblings) < 2 {
		return t.insertChildInternal()
	}
	childCount := len(*siblings)

	oldIndex := t.rng.IntN(childCount)
	newIndex := t.rng.IntN(childCount - 1)
	if newIndex >= oldIndex {
		newIndex++
	}

	child := (*siblings)[oldIndex]
	*siblings = slices.Delete(*siblings, oldIndex, oldIndex+1)
	*siblings = slices.Insert(*siblings, newIndex, child)
	s.rebuildElementPaths()
	return appliedRandomizedElementTreeMutation{
		mutation: RandomizedElementTreeMutationReordered{
			ElementID: child.id,
			ParentID:  parentID,
			OldIndex:  oldIndex,
			NewIndex:  newIndex,
		},
		ownerEntityID: ownerEntityID,
	}
}

func (t *RandomizedElementTree) synchronizeEntities(cx *Context[RandomizedElementTree]) {
	var entityIDs []uint64
	for _, elementID := range t.snapshot.ElementIDs() {
		if node := t.snapshot.node(elementID); node != nil && node.isEntity {
			entityIDs = append(entityIDs, elementID)
		}
	}
	entityIDSet := make(map[uint64]struct{}, len(entityIDs))
	for _, id := range entityIDs {
		entityIDSet[id] = struct{}{}
	}

	for elementID, entity := range t.entities.entities {
		if _, keep := entityIDSet[elementID]; !keep {
			delete(t.entities.entities, elementID)
			entity.Release()
		}
	}
	var missingEntityIDs []uint64
	for _, elementID := range entityIDs {
		if _, ok := t.entities.entities[elementID]; !ok {
			missingEntityIDs = append(missingEntityIDs, elementID)
		}
	}

	for _, elementID := range missingEntityIDs {
		entity := cx.New(func(*Context[randomizedElementEntity]) *randomizedElementEntity {
			return &randomizedElementEntity{
				elementID:     elementID,
				snapshot:      t.snapshot,
				entities:      t.entities,
				scrollHandles: t.scrollHandles,
				workCounters:  t.workCounters,
			}
		})
		t.entities.entities[elementID] = entity
	}
}

func (t *RandomizedElementTree) notifyOwner(ownerEntityID Option[uint64], cx *Context[RandomizedElementTree]) {
	if elementID, ok := ownerEntityID.Get(); ok {
		if owner, ok := t.entities.entities[elementID]; ok {
			owner.Update(cx, func(_ *randomizedElementEntity, cx *Context[randomizedElementEntity]) { cx.Notify() })
			return
		}
	}
	cx.Notify()
}

// Render renders the persistent root and its randomized descendants.
func (t *RandomizedElementTree) Render(_ *Window, _ *Context[RandomizedElementTree]) IntoElement {
	snapshot := t.snapshot
	root := &snapshot.root
	t.workCounters.recordRootRender()

	children := make([]IntoElement, 0, len(snapshot.children))
	for i := range snapshot.children {
		children = append(children, renderRandomizedNode(&snapshot.children[i], t.entities.entities, t.scrollHandles, t.workCounters, snapshot.renderOptions))
	}
	return NewDiv().
		Flex().
		When(snapshot.renderOptions.hoverStyles, func(element *Div) *Div { return element.Group("randomized-tree-root") }).
		When(root.vertical, func(element *Div) *Div { return element.FlexCol() }).
		When(root.wrap, func(element *Div) *Div { return element.FlexWrap() }).
		Gap(Px(root.gap)).
		P(Px(root.padding)).
		W(Px(root.width)).
		H(Px(root.height)).
		MaxWFull().
		MaxHFull().
		Bg(root.background).
		Border(Px(root.borderWidth)).
		BorderColor(root.borderColor).
		Rounded(Px(root.cornerRadius)).
		Opacity(root.opacity).
		When(root.clipped, func(element *Div) *Div { return element.OverflowHidden() }).
		When(!root.visible, func(element *Div) *Div { return element.Invisible() }).
		Children(children...)
}

type randomizedRenderOptions struct {
	cachedViews bool
	hoverStyles bool
	paths       bool
}

type randomizedElementEntity struct {
	elementID     uint64
	snapshot      *RandomizedElementTreeSnapshot
	entities      *randomizedElementEntities
	scrollHandles map[uint64]ScrollHandle
	workCounters  *RandomizedElementTreeWorkCounters
}

func (e *randomizedElementEntity) Render(_ *Window, _ *Context[randomizedElementEntity]) IntoElement {
	snapshot := e.snapshot
	var entities map[uint64]Entity[randomizedElementEntity]
	if !e.entities.dropped {
		entities = e.entities.entities
	}
	e.workCounters.recordEntityRender(e.elementID)

	node := snapshot.node(e.elementID)
	if node == nil {
		return NewDiv()
	}
	return renderRandomizedNodeBody(node, entities, entities != nil, e.scrollHandles, e.workCounters, snapshot.renderOptions)
}

type appliedRandomizedElementTreeMutation struct {
	mutation            RandomizedElementTreeMutation
	ownerEntityID       Option[uint64]
	synchronizeEntities bool
}

func appliedRootMutation(mutation RandomizedElementTreeMutation) appliedRandomizedElementTreeMutation {
	return appliedRandomizedElementTreeMutation{mutation: mutation}
}

type randomizedElementNode struct {
	id       uint64
	style    randomizedElementStyle
	isEntity bool
	features randomizedElementFeatures
	children []randomizedElementNode
}

func randomRandomizedElementNode(id uint64, config *RandomizedElementTreeConfig, rng *rand.Rand) randomizedElementNode {
	style := randomChildStyle(rng)
	isEntity := randomBool(rng, config.entityDensity)
	features := randomRandomizedElementFeatures(config.handlerDensity, rng)
	features.animated = isEntity && config.animationDensity > 0.0 && randomBool(rng, config.animationDensity)
	return randomizedElementNode{
		id:       id,
		style:    style,
		isEntity: isEntity,
		features: features,
	}
}

func (n *randomizedElementNode) elementCount() int {
	count := 0
	stack := []*randomizedElementNode{n}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		stack = append(stack, randomizedNodeRefs(node.children)...)
	}
	return count
}

func cloneRandomizedNodes(nodes []randomizedElementNode) []randomizedElementNode {
	if nodes == nil {
		return nil
	}
	out := make([]randomizedElementNode, len(nodes))
	for i, node := range nodes {
		out[i] = node
		out[i].children = cloneRandomizedNodes(node.children)
	}
	return out
}

func randomizedNodeRefs(nodes []randomizedElementNode) []*randomizedElementNode {
	refs := make([]*randomizedElementNode, len(nodes))
	for i := range nodes {
		refs[i] = &nodes[i]
	}
	return refs
}

type randomizedElementFeatures struct {
	actionHandler bool
	clickHandler  bool
	keyContext    bool
	animated      bool
}

func randomRandomizedElementFeatures(handlerDensity float64, rng *rand.Rand) randomizedElementFeatures {
	actionHandler := randomBool(rng, handlerDensity)
	clickHandler := randomBool(rng, handlerDensity)
	keyContext := randomBool(rng, handlerDensity)
	return randomizedElementFeatures{
		actionHandler: actionHandler,
		clickHandler:  clickHandler,
		keyContext:    keyContext,
	}
}

func (f randomizedElementFeatures) handlerCount() int {
	count := 0
	if f.actionHandler {
		count++
	}
	if f.clickHandler {
		count++
	}
	return count
}

type randomizedElementStyle struct {
	background   Hsla
	borderColor  Hsla
	opacity      float32
	borderWidth  float32
	cornerRadius float32
	width        float32
	height       float32
	padding      float32
	gap          float32
	clipped      bool
	visible      bool
	vertical     bool
	wrap         bool
	label        int
}

var randomizedLabels = [4]string{
	"Retained glyphs: abc Ω",
	"Retained glyphs: abc Ω, now with enough words that narrow parents wrap them",
	"Ω",
	"",
}

func randomRootStyle(rng *rand.Rand) randomizedElementStyle {
	var s randomizedElementStyle
	s.background = randomColor(rng)
	s.borderColor = randomColor(rng)
	s.opacity = randomOpacity(rng)
	s.borderWidth = randomF32(rng, 0.0, 4.0)
	s.cornerRadius = randomF32(rng, 0.0, 18.0)
	s.width = randomF32(rng, 520.0, 980.0)
	s.height = randomF32(rng, 360.0, 760.0)
	s.padding = randomF32(rng, 4.0, 18.0)
	s.gap = randomF32(rng, 2.0, 14.0)
	s.clipped = randomBool(rng, 0.5)
	s.visible = true
	s.vertical = randomBool(rng, 0.5)
	s.wrap = randomBool(rng, 0.7)
	return s
}

func randomChildStyle(rng *rand.Rand) randomizedElementStyle {
	var s randomizedElementStyle
	s.background = randomColor(rng)
	s.borderColor = randomColor(rng)
	s.opacity = randomOpacity(rng)
	s.borderWidth = randomF32(rng, 0.0, 4.0)
	s.cornerRadius = randomF32(rng, 0.0, 14.0)
	s.width = randomF32(rng, 36.0, 220.0)
	s.height = randomF32(rng, 28.0, 180.0)
	s.padding = randomF32(rng, 1.0, 10.0)
	s.gap = randomF32(rng, 1.0, 8.0)
	s.clipped = randomBool(rng, 0.3)
	s.visible = randomBool(rng, 0.9)
	s.vertical = randomBool(rng, 0.5)
	s.wrap = randomBool(rng, 0.4)
	return s
}

func (s *randomizedElementStyle) randomizeBorder(rng *rand.Rand) {
	s.borderColor = randomColor(rng)
	s.borderWidth = randomF32(rng, 0.0, 5.0)
	s.cornerRadius = randomF32(rng, 0.0, 18.0)
}

type randomizedChildMutation uint8

const (
	randomizedChildMutationColor randomizedChildMutation = iota
	randomizedChildMutationOpacity
	randomizedChildMutationBorder
	randomizedChildMutationBounds
	randomizedChildMutationClipping
	randomizedChildMutationVisibility
	randomizedChildMutationText
)

func renderRandomizedNode(node *randomizedElementNode, entities map[uint64]Entity[randomizedElementEntity], scrollHandles map[uint64]ScrollHandle, workCounters *RandomizedElementTreeWorkCounters, options randomizedRenderOptions) IntoElement {
	if node.isEntity {
		if entity, ok := entities[node.id]; ok {
			if options.cachedViews {
				var style StyleRefinement
				style.Size.Width = Some(Px(node.style.width).ToLength())
				style.Size.Height = Some(Px(node.style.height).ToLength())
				style.FlexShrink = Some[float32](0.0)
				return entity.Clone().Cached(style)
			}
			return entity.Clone()
		}
	}
	return renderRandomizedNodeBody(node, entities, true, scrollHandles, workCounters, options)
}

func randomizedStyled(style *StyleRefinement) *Styled[*StyleRefinement] {
	var styled Styled[*StyleRefinement]
	BindStyled(&styled, style)
	return &styled
}

func renderRandomizedNodeBody(node *randomizedElementNode, entities map[uint64]Entity[randomizedElementEntity], hasEntities bool, scrollHandles map[uint64]ScrollHandle, workCounters *RandomizedElementTreeWorkCounters, options randomizedRenderOptions) IntoElement {
	workCounters.recordElementRender(node.features.handlerCount())
	style := node.style
	type element = *Stateful[*Div]
	children := make([]IntoElement, 0, len(node.children))
	for i := range node.children {
		child := &node.children[i]
		if hasEntities {
			children = append(children, renderRandomizedNode(child, entities, scrollHandles, workCounters, options))
		} else {
			children = append(children, renderRandomizedNodeBody(child, nil, false, scrollHandles, workCounters, options))
		}
	}
	color := style.borderColor
	width := style.borderWidth
	bend := style.cornerRadius
	pathCanvas := NewCanvas(
		func(Bounds[Pixels], *Window, *App) struct{} { return struct{}{} },
		func(bounds Bounds[Pixels], _ struct{}, window *Window, _ *App) {
			if !options.paths {
				return
			}
			for index := range 6 {
				var path *PathBuilder
				if index%2 == 0 {
					path = PathBuilderFill()
				} else {
					path = PathBuilderStroke(Px(width + 0.5))
				}
				origin := bounds.Origin.Add(NewPoint(Px(float32(index)*3.0-8.0), Px(-4.0)))
				path.MoveTo(origin)
				path.CurveTo(
					origin.Add(NewPoint(bounds.Size.Width, bounds.Size.Height)),
					origin.Add(NewPoint(Px(bend), Px(-bend))),
				)
				path.LineTo(origin.Add(NewPoint(Px(-bend), bounds.Size.Height)))
				path.Close()
				built, err := path.Build()
				if err != nil {
					panic(fmt.Sprintf("generated path must tessellate: %v", err))
				}
				window.PaintPath(built, color.ToBackground())
			}
		},
	).Absolute().SizeFull()

	return NewDiv().
		ID(ElementIDNamedInteger("randomized-element", node.id)).
		Flex().
		FlexShrink0().
		When(style.vertical, func(e element) element { return e.FlexCol() }).
		When(style.wrap, func(e element) element { return e.FlexWrap() }).
		Gap(Px(style.gap)).
		P(Px(style.padding)).
		W(Px(style.width)).
		H(Px(style.height)).
		Bg(style.background).
		When(options.hoverStyles, func(e element) element {
			return e.
				Hover(func(s *StyleRefinement) *StyleRefinement {
					s = randomizedStyled(s).Bg(style.borderColor)
					return randomizedStyled(s).Border(Px(style.borderWidth + 1.0))
				}).
				GroupHover("randomized-tree-root", func(s *StyleRefinement) *StyleRefinement {
					return randomizedStyled(s).Opacity(0.65)
				})
		}).
		Border(Px(style.borderWidth)).
		BorderColor(style.borderColor).
		Rounded(Px(style.cornerRadius)).
		When(node.id%2 == 0, func(e element) element {
			return e.Shadow([]BoxShadow{
				NewBoxShadow(Px(-style.gap), Px(style.gap), style.borderColor).BlurRadius(Px(style.cornerRadius)),
			})
		}).
		Opacity(style.opacity).
		When(style.clipped, func(e element) element { return e.OverflowHidden() }).
		When(node.id%3 == 0, func(e element) element {
			handle, ok := scrollHandles[node.id]
			if !ok {
				handle = NewScrollHandle()
				scrollHandles[node.id] = handle
			}
			return e.OverflowScroll().TrackScroll(handle)
		}).
		When(!style.visible, func(e element) element { return e.Invisible() }).
		When(node.features.actionHandler, func(e element) element {
			return e.OnAction(func(RandomizedElementTreeAction, *Window, *App) {})
		}).
		When(node.features.clickHandler, func(e element) element {
			return e.CursorPointer().OnClick(func(*ClickEvent, *Window, *App) {})
		}).
		When(node.features.keyContext, func(e element) element {
			return e.
				KeyContext("RandomizedElementTree").
				TabIndex(int(node.id % 5)).
				TabStop(node.id%7 != 0)
		}).
		When(node.id%5 == 0, func(e element) element { return e.TabGroup() }).
		When(node.features.animated, func(e element) element {
			return e.Child(WithAnimation(
				NewDiv().
					Size(Px(10.)).
					FlexShrink0().
					RoundedFull().
					Bg(style.borderColor),
				ElementIDNamedInteger("randomized-spinner", node.id),
				NewAnimation(800*time.Millisecond).Repeat(),
				func(element *Div, delta float32) *Div { return element.Opacity(0.25 + 0.75*delta) },
			))
		}).
		When(node.id%4 == 1 || style.label != 0, func(e element) element {
			return e.Underline().ChildText(randomizedLabels[style.label%len(randomizedLabels)])
		}).
		Child(pathCanvas).
		Children(children...)
}

func insertRandomizedNode(rootChildren *[]randomizedElementNode, parentPath *[]int, node randomizedElementNode) (int, []int) {
	children := randomizedChildrenAtPathMut(rootChildren, *parentPath)
	if children == nil {
		*parentPath = (*parentPath)[:0]
		index := len(*rootChildren)
		*rootChildren = append(*rootChildren, node)
		return index, []int{index}
	}

	index := len(*children)
	*children = append(*children, node)
	insertedPath := append(slices.Clone(*parentPath), index)
	return index, insertedPath
}

func randomParentPath(rootChildren []randomizedElementNode, rng *rand.Rand) []int {
	paths := randomizedNodePaths(rootChildren)
	index := rng.IntN(len(paths) + 1)
	if index < len(paths) {
		return slices.Clone(paths[index])
	}
	return nil
}

func randomNodePath(rootChildren []randomizedElementNode, rng *rand.Rand) ([]int, bool) {
	paths := randomizedNodePaths(rootChildren)
	if len(paths) == 0 {
		return nil, false
	}
	return slices.Clone(paths[rng.IntN(len(paths))]), true
}

func randomLeafPath(rootChildren []randomizedElementNode, rng *rand.Rand) ([]int, bool) {
	var leaves [][]int
	for _, path := range randomizedNodePaths(rootChildren) {
		if node := randomizedNodeAtPath(rootChildren, path); node != nil && len(node.children) == 0 {
			leaves = append(leaves, path)
		}
	}
	if len(leaves) == 0 {
		return nil, false
	}
	return slices.Clone(leaves[rng.IntN(len(leaves))]), true
}

func parentPathsWithMultipleChildren(rootChildren []randomizedElementNode) [][]int {
	var parentPaths [][]int
	if len(rootChildren) >= 2 {
		parentPaths = append(parentPaths, []int{})
	}
	for _, path := range randomizedNodePaths(rootChildren) {
		if node := randomizedNodeAtPath(rootChildren, path); node != nil && len(node.children) >= 2 {
			parentPaths = append(parentPaths, path)
		}
	}
	return parentPaths
}

func randomizedNodePaths(rootChildren []randomizedElementNode) [][]int {
	type entry struct {
		path []int
		node *randomizedElementNode
	}
	var paths [][]int
	stack := make([]entry, 0, len(rootChildren))
	for index := len(rootChildren) - 1; index >= 0; index-- {
		stack = append(stack, entry{[]int{index}, &rootChildren[index]})
	}
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for index := len(e.node.children) - 1; index >= 0; index-- {
			childPath := append(slices.Clone(e.path), index)
			stack = append(stack, entry{childPath, &e.node.children[index]})
		}
		paths = append(paths, e.path)
	}
	return paths
}

func randomizedNodeAtPath(rootChildren []randomizedElementNode, path []int) *randomizedElementNode {
	if len(path) == 0 {
		return nil
	}
	index := path[len(path)-1]
	children := randomizedChildrenAtPath(rootChildren, path[:len(path)-1])
	if children == nil || index >= len(children) {
		return nil
	}
	return &children[index]
}

func randomizedNodeIDAtPath(rootChildren []randomizedElementNode, path []int) Option[uint64] {
	if node := randomizedNodeAtPath(rootChildren, path); node != nil {
		return Some(node.id)
	}
	return None[uint64]()
}

func randomizedChildrenAtPath(children []randomizedElementNode, path []int) []randomizedElementNode {
	for _, index := range path {
		if index >= len(children) {
			return nil
		}
		children = children[index].children
	}
	if children == nil {
		return []randomizedElementNode{}
	}
	return children
}

func randomizedChildrenAtPathMut(children *[]randomizedElementNode, path []int) *[]randomizedElementNode {
	for _, index := range path {
		if index >= len(*children) {
			return nil
		}
		children = &(*children)[index].children
	}
	return children
}

func nearestEntityID(rootChildren []randomizedElementNode, path []int) Option[uint64] {
	children := rootChildren
	nearest := None[uint64]()
	for _, index := range path {
		if index >= len(children) {
			return None[uint64]()
		}
		node := &children[index]
		if node.isEntity {
			nearest = Some(node.id)
		}
		children = node.children
	}
	return nearest
}

func normalizedDensity(density float64) float64 {
	if math.IsNaN(density) {
		return 0.0
	}
	return min(max(density, 0.0), 1.0)
}

func randomColor(rng *rand.Rand) Hsla {
	return Hsla{
		H: randomF32(rng, 0.0, 1.0),
		S: randomF32(rng, 0.25, 0.9),
		L: randomF32(rng, 0.18, 0.82),
		A: 1.0,
	}
}

func randomOpacity(rng *rand.Rand) float32 {
	return randomF32(rng, 0.25, 1.0)
}

func randomF32(rng *rand.Rand, start, end float32) float32 {
	return start + rng.Float32()*(end-start)
}

func randomBool(rng *rand.Rand, p float64) bool {
	if p >= 1.0 {
		return true
	}
	return rng.Float64() < p
}

func randomIntRange(rng *rand.Rand, start, end int) int {
	return start + rng.IntN(end-start)
}

func randomIntInclusive(rng *rand.Rand, start, end int) int {
	return start + rng.IntN(end-start+1)
}
