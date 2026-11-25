package gpui

import (
	"cmp"
	"encoding/json/jsontext"
	"fmt"
	"iter"
	"math"
	"math/bits"
	"slices"
	"time"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/jsonschema"
)

// DrawOrder is the position of a primitive in the scene's drawing order.
type DrawOrder = uint32

// PaddedBool32 is a boolean stored as a uint32 so that GPU-facing structs contain no
// padding bytes. It is 0 or 1 by construction; shaders read it as a u32.
type PaddedBool32 uint32

// PaddedBool32FromBool converts a bool into a PaddedBool32.
func PaddedBool32FromBool(value bool) PaddedBool32 {
	if value {
		return 1
	}
	return 0
}

// AsBool returns the wrapped boolean.
func (b PaddedBool32) AsBool() bool {
	return b != 0
}

type layer struct {
	primitives    []Range
	bounds        Bounds[ScaledPixels]
	maxKind       primitiveKind
	maxSharedKind primitiveKind
	hasNativeView bool
}

// Scene is the list of primitives painted in one frame, ordered for the renderer.
// The zero value is an empty scene.
type Scene struct {
	paintOperations []paintOperation
	primitiveBounds boundsTree[ScaledPixels]
	layerStack      []int
	layers          []layer
	// Shadows are the scene's shadows, in drawing order after Finish.
	Shadows []Shadow
	// Quads are the scene's quads, in drawing order after Finish.
	Quads []Quad
	// Paths are the scene's paths, in drawing order after Finish.
	Paths []PaintedPath
	// Underlines are the scene's underlines, in drawing order after Finish.
	Underlines []Underline
	// MonochromeSprites are the scene's monochrome sprites, in drawing order after Finish.
	MonochromeSprites []MonochromeSprite
	// SubpixelSprites are the scene's subpixel sprites, in drawing order after Finish.
	SubpixelSprites []SubpixelSprite
	// PolychromeSprites are the scene's polychrome sprites, in drawing order after Finish.
	PolychromeSprites []PolychromeSprite
	// Surfaces are the scene's surfaces, in drawing order after Finish.
	Surfaces []PaintSurface
	// NativeViews are where the scene's native views show through, in drawing order after Finish.
	NativeViews []PaintNativeView
	animations  []spriteAnimation
}

// Clear empties the scene, keeping its allocations.
func (s *Scene) Clear() {
	s.paintOperations = s.paintOperations[:0]
	s.primitiveBounds.clear()
	s.layerStack = s.layerStack[:0]
	s.layers = s.layers[:0]
	s.Paths = s.Paths[:0]
	s.Shadows = s.Shadows[:0]
	s.Quads = s.Quads[:0]
	s.Underlines = s.Underlines[:0]
	s.MonochromeSprites = s.MonochromeSprites[:0]
	s.SubpixelSprites = s.SubpixelSprites[:0]
	s.PolychromeSprites = s.PolychromeSprites[:0]
	s.Surfaces = s.Surfaces[:0]
	s.NativeViews = s.NativeViews[:0]
	s.animations = s.animations[:0]
}

func (s *Scene) insertAnimatedSprite(sprite MonochromeSprite, animation spriteAnimation) {
	sprite.Animation = SpriteAnimationID(uint32(len(s.animations)))
	animation.baseColor = sprite.Color
	if s.tryInsertPrimitive(sprite.IntoPrimitive()) {
		s.animations = append(s.animations, animation)
	}
}

func (s *Scene) animationFrameInterval() (time.Duration, bool) {
	if len(s.animations) == 0 {
		return 0, false
	}
	var minInterval time.Duration
	for i, animation := range s.animations {
		interval, _ := animation.transformation.minFrameInterval()
		if i == 0 || interval < minInterval {
			minInterval = interval
		}
	}
	return minInterval, true
}

func (s *Scene) advanceAnimations(elapsed time.Duration) {
	for _, animation := range s.animations {
		if animation.spriteIndex < 0 || animation.spriteIndex >= len(s.MonochromeSprites) {
			continue
		}
		sprite := &s.MonochromeSprites[animation.spriteIndex]
		pose := animation.transformation.at(elapsed)
		sprite.Transformation = pose.intoMatrix(animation.center, animation.scaleFactor)
		sprite.Color = animation.baseColor.Opacity(pose.alphaFactor())
	}
}

// Len returns the number of paint operations recorded in the scene.
func (s *Scene) Len() int {
	return len(s.paintOperations)
}

// PushLayer starts a layer: its primitives share one place in the drawing order.
func (s *Scene) PushLayer() {
	s.layerStack = append(s.layerStack, len(s.layers))
	s.layers = append(s.layers, layer{
		bounds:        Bounds[ScaledPixels]{},
		maxKind:       primitiveKindShadow,
		maxSharedKind: primitiveKindNativeView,
	})
	s.paintOperations = append(s.paintOperations, paintOperation{kind: paintOperationStartLayer})
}

// PopLayer ends the layer started by the matching PushLayer.
func (s *Scene) PopLayer() {
	if len(s.layerStack) > 0 {
		s.layerStack = s.layerStack[:len(s.layerStack)-1]
	}
	if len(s.layerStack) == 0 {
		for i := range s.layers {
			s.insertLayer(s.layers[i])
		}
		clear(s.layers)
		s.layers = s.layers[:0]
	}
	s.paintOperations = append(s.paintOperations, paintOperation{kind: paintOperationEndLayer})
}

func (s *Scene) insertLayer(l layer) {
	if len(l.primitives) == 0 {
		return
	}
	if l.hasNativeView {
		s.insertLayerAroundNativeViews(l)
		return
	}
	order := s.primitiveBounds.insert(l.bounds, l.maxKind, l.maxSharedKind)
	for _, r := range l.primitives {
		for index := r.Start; index < r.End; index++ {
			if op := s.paintOperations[index]; op.kind == paintOperationPrimitive {
				s.pushPrimitive(op.primitive, order)
			}
		}
	}
}

func (s *Scene) insertLayerAroundNativeViews(l layer) {
	var part []Primitive
	for _, r := range l.primitives {
		for index := r.Start; index < r.End; index++ {
			op := s.paintOperations[index]
			if op.kind != paintOperationPrimitive {
				continue
			}
			primitive := op.primitive
			if primitive.kind() == primitiveKindNativeView {
				taken := part
				part = nil
				s.insertLayerPart(taken)
				s.insertLayerPart([]Primitive{primitive})
			} else {
				part = append(part, primitive)
			}
		}
	}
	s.insertLayerPart(part)
}

func (s *Scene) insertLayerPart(primitives []Primitive) {
	if len(primitives) == 0 {
		return
	}
	bounds := clippedBounds(primitives[0])
	maxKind := primitiveKindShadow
	maxSharedKind := primitiveKindNativeView
	for _, primitive := range primitives {
		kind := primitive.kind()
		bounds = bounds.Join(clippedBounds(primitive))
		maxKind = max(maxKind, kind)
		maxSharedKind = min(maxSharedKind, kind.maxSharedKind())
	}
	order := s.primitiveBounds.insert(bounds, maxKind, maxSharedKind)
	for _, primitive := range primitives {
		s.pushPrimitive(primitive, order)
	}
}

// InsertPrimitive adds a primitive to the scene, unless its content mask clips it away.
func (s *Scene) InsertPrimitive(primitive IntoPrimitive) {
	s.tryInsertPrimitive(primitive.IntoPrimitive())
}

func (s *Scene) tryInsertPrimitive(primitive Primitive) bool {
	clipped := clippedBounds(primitive)
	if clipped.IsEmpty() {
		return false
	}

	kind := primitive.kind()
	if len(s.layerStack) > 0 && s.layerStack[len(s.layerStack)-1] < len(s.layers) {
		l := &s.layers[s.layerStack[len(s.layerStack)-1]]
		l.bounds = l.bounds.Join(clipped)
		l.maxKind = max(l.maxKind, kind)
		l.maxSharedKind = min(l.maxSharedKind, kind.maxSharedKind())
		l.hasNativeView = l.hasNativeView || kind == primitiveKindNativeView
		index := len(s.paintOperations)
		if n := len(l.primitives); n > 0 && l.primitives[n-1].End == index {
			l.primitives[n-1].End++
		} else {
			l.primitives = append(l.primitives, Range{Start: index, End: index + 1})
		}
	} else {
		order := s.primitiveBounds.insert(clipped, kind, kind.maxSharedKind())
		s.pushPrimitive(primitive, order)
	}
	s.paintOperations = append(s.paintOperations, paintOperation{kind: paintOperationPrimitive, primitive: primitive})
	return true
}

func (s *Scene) pushPrimitive(primitive Primitive, order DrawOrder) {
	switch p := primitive.(type) {
	case PrimitiveShadow:
		shadow := p.Shadow
		shadow.Order = order
		s.Shadows = append(s.Shadows, shadow)
	case PrimitiveQuad:
		quad := p.Quad
		quad.Order = order
		s.Quads = append(s.Quads, quad)
	case PrimitivePath:
		path := p.PaintedPath
		path.order = order
		s.Paths = append(s.Paths, path)
	case PrimitiveUnderline:
		underline := p.Underline
		underline.Order = order
		s.Underlines = append(s.Underlines, underline)
	case PrimitiveMonochromeSprite:
		sprite := p.MonochromeSprite
		sprite.Order = order
		s.MonochromeSprites = append(s.MonochromeSprites, sprite)
	case PrimitiveSubpixelSprite:
		sprite := p.SubpixelSprite
		sprite.Order = order
		s.SubpixelSprites = append(s.SubpixelSprites, sprite)
	case PrimitivePolychromeSprite:
		sprite := p.PolychromeSprite
		sprite.Order = order
		s.PolychromeSprites = append(s.PolychromeSprites, sprite)
	case PrimitiveSurface:
		surface := p.PaintSurface
		surface.Order = order
		s.Surfaces = append(s.Surfaces, surface)
	case PrimitiveNativeView:
		nativeView := p.PaintNativeView
		nativeView.Order = order
		s.NativeViews = append(s.NativeViews, nativeView)
	}
}

// Replay re-records the paint operations in r of prevScene into this scene.
func (s *Scene) Replay(r Range, prevScene *Scene) {
	for _, operation := range prevScene.paintOperations[r.Start:r.End] {
		switch operation.kind {
		case paintOperationPrimitive:
			if sprite, ok := operation.primitive.(PrimitiveMonochromeSprite); ok &&
				uint64(sprite.Animation) < uint64(len(prevScene.animations)) {
				s.insertAnimatedSprite(sprite.MonochromeSprite, prevScene.animations[sprite.Animation])
			} else {
				s.InsertPrimitive(operation.primitive)
			}
		case paintOperationStartLayer:
			s.PushLayer()
		case paintOperationEndLayer:
			s.PopLayer()
		}
	}
}

// Finish sorts the primitives into drawing order; call it once painting is done.
func (s *Scene) Finish() {
	if gpuiutil.DebugAssertions && len(s.layerStack) != 0 {
		panic("unbalanced scene layers")
	}
	keys := &[]sortInPlaceKey{}
	sortInPlace(s.Shadows, keys, func(shadow *Shadow) (uint32, uint32) { return shadow.Order, 0 })
	sortInPlace(s.Quads, keys, func(quad *Quad) (uint32, uint32) { return quad.Order, 0 })
	slices.SortStableFunc(s.Paths, func(a, b PaintedPath) int { return cmp.Compare(a.order, b.order) })
	sortInPlace(s.Underlines, keys, func(underline *Underline) (uint32, uint32) { return underline.Order, 0 })
	sortInPlace(s.MonochromeSprites, keys, func(sprite *MonochromeSprite) (uint32, uint32) {
		return sprite.Order, sprite.Tile.TextureID.Index
	})
	sortInPlace(s.SubpixelSprites, keys, func(sprite *SubpixelSprite) (uint32, uint32) {
		return sprite.Order, sprite.Tile.TextureID.Index
	})
	sortInPlace(s.PolychromeSprites, keys, func(sprite *PolychromeSprite) (uint32, uint32) {
		return sprite.Order, sprite.Tile.TextureID.Index
	})
	slices.SortStableFunc(s.Surfaces, func(a, b PaintSurface) int { return cmp.Compare(a.Order, b.Order) })
	slices.SortStableFunc(s.NativeViews, func(a, b PaintNativeView) int { return cmp.Compare(a.Order, b.Order) })
	s.keepFirstPaintedNativeViews()

	if len(s.animations) != 0 {
		for index, sprite := range s.MonochromeSprites {
			if uint64(sprite.Animation) < uint64(len(s.animations)) {
				s.animations[sprite.Animation].spriteIndex = index
			}
		}
	}
}

func (s *Scene) keepFirstPaintedNativeViews() {
	nativeViews := s.NativeViews
	isRepeated := false
	for index, nativeView := range nativeViews {
		for _, earlier := range nativeViews[:index] {
			if earlier.ID == nativeView.ID {
				isRepeated = true
				break
			}
		}
		if isRepeated {
			break
		}
	}
	if !isRepeated {
		return
	}
	type firstPaintedView struct {
		id       NativeViewID
		hitboxID HitboxID
	}
	var firstPainted []firstPaintedView
	for _, operation := range s.paintOperations {
		if operation.kind != paintOperationPrimitive {
			continue
		}
		nativeView, ok := operation.primitive.(PrimitiveNativeView)
		if !ok {
			continue
		}
		if !slices.ContainsFunc(firstPainted, func(v firstPaintedView) bool { return v.id == nativeView.ID }) {
			firstPainted = append(firstPainted, firstPaintedView{id: nativeView.ID, hitboxID: nativeView.HitboxID})
		}
	}
	s.NativeViews = slices.DeleteFunc(s.NativeViews, func(nativeView PaintNativeView) bool {
		return !slices.Contains(firstPainted, firstPaintedView{id: nativeView.ID, hitboxID: nativeView.HitboxID})
	})
}

// Batches returns the scene's primitives grouped into batches the renderer draws in order.
// Call Finish first.
func (s *Scene) Batches() iter.Seq[PrimitiveBatch] {
	return func(yield func(PrimitiveBatch) bool) {
		if s == nil {
			return
		}
		it := batchIterator{scene: s}
		for {
			batch, ok := it.next()
			if !ok || !yield(batch) {
				return
			}
		}
	}
}

type sortInPlaceKey struct {
	key   uint64
	index uint32
}

func sortInPlace[T any](primitives []T, keys *[]sortInPlaceKey, key func(*T) (uint32, uint32)) {
	isSorted := true
	for i := 1; i < len(primitives); i++ {
		prevMajor, prevMinor := key(&primitives[i-1])
		major, minor := key(&primitives[i])
		if prevMajor > major || (prevMajor == major && prevMinor > minor) {
			isSorted = false
			break
		}
	}
	if isSorted {
		return
	}
	k := (*keys)[:0]
	for index := range primitives {
		major, minor := key(&primitives[index])
		k = append(k, sortInPlaceKey{key: uint64(major)<<32 | uint64(minor), index: uint32(index)})
	}
	slices.SortFunc(k, func(a, b sortInPlaceKey) int {
		if c := cmp.Compare(a.key, b.key); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})
	*keys = k

	for start := range primitives {
		if int(k[start].index) == start {
			continue
		}
		displaced := primitives[start]
		target := start
		for {
			source := int(k[target].index)
			k[target].index = uint32(target)
			if source == start {
				primitives[target] = displaced
				break
			}
			primitives[target] = primitives[source]
			target = source
		}
	}
}

// QuadDepthScale returns the depth step between consecutive quads in a frame with quadCount quads.
func QuadDepthScale(quadCount int) float32 {
	denominator := uint64(min(quadCount, (1<<24)-1) + 1)
	if denominator > 1 {
		denominator = 1 << bits.Len64(denominator-1)
	}
	return 1.0 / float32(denominator)
}

// QuadDepth maps a quad's index in Scene.Quads to depth, with greater values closer.
func QuadDepth(quadID uint32, scale float32) float32 {
	return minF32(float32(float32(quadID+1)*scale), 1.0)
}

type primitiveKind uint8

const (
	primitiveKindShadow primitiveKind = iota
	primitiveKindQuad
	primitiveKindPath
	primitiveKindUnderline
	primitiveKindMonochromeSprite
	primitiveKindSubpixelSprite
	primitiveKindPolychromeSprite
	primitiveKindSurface
	primitiveKindNativeView
)

func (k primitiveKind) maxSharedKind() primitiveKind {
	switch k {
	case primitiveKindMonochromeSprite:
		return primitiveKindUnderline
	case primitiveKindSubpixelSprite:
		return primitiveKindMonochromeSprite
	case primitiveKindPolychromeSprite:
		return primitiveKindSubpixelSprite
	}
	return k
}

type paintOperationKind uint8

const (
	paintOperationPrimitive paintOperationKind = iota
	paintOperationStartLayer
	paintOperationEndLayer
)

type paintOperation struct {
	kind      paintOperationKind
	primitive Primitive
}

// Primitive is one painted primitive: a PrimitiveShadow, PrimitiveQuad, PrimitivePath,
// PrimitiveUnderline, PrimitiveMonochromeSprite, PrimitiveSubpixelSprite,
// PrimitivePolychromeSprite, PrimitiveSurface or PrimitiveNativeView.
type Primitive interface {
	IntoPrimitive
	// Bounds returns the primitive's bounds before transformation and clipping.
	Bounds() Bounds[ScaledPixels]
	// ContentMask returns the clip applied to the primitive.
	ContentMask() ContentMask[ScaledPixels]
	kind() primitiveKind
	transformation() TransformationMatrix
}

// IntoPrimitive is implemented by the primitive structs and the Primitive variants (D-INTO).
type IntoPrimitive interface {
	IntoPrimitive() Primitive
}

// PrimitiveShadow is the Shadow variant of Primitive.
type PrimitiveShadow struct{ Shadow }

// PrimitiveQuad is the Quad variant of Primitive.
type PrimitiveQuad struct{ Quad }

// PrimitivePath is the Path variant of Primitive.
type PrimitivePath struct{ PaintedPath }

// PrimitiveUnderline is the Underline variant of Primitive.
type PrimitiveUnderline struct{ Underline }

// PrimitiveMonochromeSprite is the MonochromeSprite variant of Primitive.
type PrimitiveMonochromeSprite struct{ MonochromeSprite }

// PrimitiveSubpixelSprite is the SubpixelSprite variant of Primitive.
type PrimitiveSubpixelSprite struct{ SubpixelSprite }

// PrimitivePolychromeSprite is the PolychromeSprite variant of Primitive.
type PrimitivePolychromeSprite struct{ PolychromeSprite }

// PrimitiveSurface is the Surface variant of Primitive.
type PrimitiveSurface struct{ PaintSurface }

// PrimitiveNativeView is the NativeView variant of Primitive.
type PrimitiveNativeView struct{ PaintNativeView }

func (PrimitiveShadow) kind() primitiveKind           { return primitiveKindShadow }
func (PrimitiveQuad) kind() primitiveKind             { return primitiveKindQuad }
func (PrimitivePath) kind() primitiveKind             { return primitiveKindPath }
func (PrimitiveUnderline) kind() primitiveKind        { return primitiveKindUnderline }
func (PrimitiveMonochromeSprite) kind() primitiveKind { return primitiveKindMonochromeSprite }
func (PrimitiveSubpixelSprite) kind() primitiveKind   { return primitiveKindSubpixelSprite }
func (PrimitivePolychromeSprite) kind() primitiveKind { return primitiveKindPolychromeSprite }
func (PrimitiveSurface) kind() primitiveKind          { return primitiveKindSurface }
func (PrimitiveNativeView) kind() primitiveKind       { return primitiveKindNativeView }

// Bounds returns the shadow's bounds.
func (p PrimitiveShadow) Bounds() Bounds[ScaledPixels] { return p.Shadow.Bounds }

// Bounds returns the quad's bounds.
func (p PrimitiveQuad) Bounds() Bounds[ScaledPixels] { return p.Quad.Bounds }

// Bounds returns the path's painted bounds.
func (p PrimitivePath) Bounds() Bounds[ScaledPixels] { return p.PaintedPath.bounds }

// Bounds returns the underline's bounds.
func (p PrimitiveUnderline) Bounds() Bounds[ScaledPixels] { return p.Underline.Bounds }

// Bounds returns the sprite's bounds.
func (p PrimitiveMonochromeSprite) Bounds() Bounds[ScaledPixels] { return p.MonochromeSprite.Bounds }

// Bounds returns the sprite's bounds.
func (p PrimitiveSubpixelSprite) Bounds() Bounds[ScaledPixels] { return p.SubpixelSprite.Bounds }

// Bounds returns the sprite's bounds.
func (p PrimitivePolychromeSprite) Bounds() Bounds[ScaledPixels] { return p.PolychromeSprite.Bounds }

// Bounds returns the surface's bounds.
func (p PrimitiveSurface) Bounds() Bounds[ScaledPixels] { return p.PaintSurface.Bounds }

// Bounds returns the native view's bounds.
func (p PrimitiveNativeView) Bounds() Bounds[ScaledPixels] { return p.PaintNativeView.Bounds }

// ContentMask returns the shadow's content mask.
func (p PrimitiveShadow) ContentMask() ContentMask[ScaledPixels] { return p.Shadow.ContentMask }

// ContentMask returns the quad's content mask.
func (p PrimitiveQuad) ContentMask() ContentMask[ScaledPixels] { return p.Quad.ContentMask }

// ContentMask returns the path's content mask.
func (p PrimitivePath) ContentMask() ContentMask[ScaledPixels] {
	return p.PaintedPath.paint.ContentMask
}

// ContentMask returns the underline's content mask.
func (p PrimitiveUnderline) ContentMask() ContentMask[ScaledPixels] {
	return p.Underline.ContentMask
}

// ContentMask returns the sprite's content mask.
func (p PrimitiveMonochromeSprite) ContentMask() ContentMask[ScaledPixels] {
	return p.MonochromeSprite.ContentMask
}

// ContentMask returns the sprite's content mask.
func (p PrimitiveSubpixelSprite) ContentMask() ContentMask[ScaledPixels] {
	return p.SubpixelSprite.ContentMask
}

// ContentMask returns the sprite's content mask.
func (p PrimitivePolychromeSprite) ContentMask() ContentMask[ScaledPixels] {
	return p.PolychromeSprite.ContentMask
}

// ContentMask returns the surface's content mask.
func (p PrimitiveSurface) ContentMask() ContentMask[ScaledPixels] {
	return p.PaintSurface.ContentMask
}

// ContentMask returns the native view's content mask.
func (p PrimitiveNativeView) ContentMask() ContentMask[ScaledPixels] {
	return p.PaintNativeView.ContentMask
}

func (PrimitiveShadow) transformation() TransformationMatrix { return TransformationMatrixUnit() }
func (PrimitiveQuad) transformation() TransformationMatrix   { return TransformationMatrixUnit() }
func (PrimitivePath) transformation() TransformationMatrix   { return TransformationMatrixUnit() }
func (PrimitiveUnderline) transformation() TransformationMatrix {
	return TransformationMatrixUnit()
}
func (p PrimitiveMonochromeSprite) transformation() TransformationMatrix {
	return p.MonochromeSprite.Transformation
}
func (p PrimitiveSubpixelSprite) transformation() TransformationMatrix {
	return p.SubpixelSprite.Transformation
}
func (PrimitivePolychromeSprite) transformation() TransformationMatrix {
	return TransformationMatrixUnit()
}
func (PrimitiveSurface) transformation() TransformationMatrix { return TransformationMatrixUnit() }
func (PrimitiveNativeView) transformation() TransformationMatrix {
	return TransformationMatrixUnit()
}

// IntoPrimitive returns p.
func (p PrimitiveShadow) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitiveQuad) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitivePath) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitiveUnderline) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitiveMonochromeSprite) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitiveSubpixelSprite) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitivePolychromeSprite) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitiveSurface) IntoPrimitive() Primitive { return p }

// IntoPrimitive returns p.
func (p PrimitiveNativeView) IntoPrimitive() Primitive { return p }

func clippedBounds(p Primitive) Bounds[ScaledPixels] {
	transformation := p.transformation()
	return transformation.transformBounds(p.Bounds()).Intersect(p.ContentMask().Bounds)
}

type batchIterator struct {
	scene                  *Scene
	shadowsStart           int
	quadsStart             int
	pathsStart             int
	underlinesStart        int
	monochromeSpritesStart int
	subpixelSpritesStart   int
	polychromeSpritesStart int
	surfacesStart          int
	nativeViewsStart       int
}

type batchOrderAndKind struct {
	order DrawOrder
	some  bool
	kind  primitiveKind
}

func (o batchOrderAndKind) sortOrder() uint32 {
	if o.some {
		return o.order
	}
	return math.MaxUint32
}

func orderAndKindLess(order uint32, kind primitiveKind, maxOrder uint32, maxKind primitiveKind) bool {
	return order < maxOrder || (order == maxOrder && kind < maxKind)
}

func peekOrder[T any](items []T, start int, order func(*T) DrawOrder, kind primitiveKind) batchOrderAndKind {
	if start < len(items) {
		return batchOrderAndKind{order: order(&items[start]), some: true, kind: kind}
	}
	return batchOrderAndKind{kind: kind}
}

func (it *batchIterator) next() (PrimitiveBatch, bool) {
	s := it.scene
	ordersAndKinds := [...]batchOrderAndKind{
		peekOrder(s.Shadows, it.shadowsStart, func(v *Shadow) DrawOrder { return v.Order }, primitiveKindShadow),
		peekOrder(s.Quads, it.quadsStart, func(v *Quad) DrawOrder { return v.Order }, primitiveKindQuad),
		peekOrder(s.Paths, it.pathsStart, func(v *PaintedPath) DrawOrder { return v.order }, primitiveKindPath),
		peekOrder(s.Underlines, it.underlinesStart, func(v *Underline) DrawOrder { return v.Order }, primitiveKindUnderline),
		peekOrder(s.MonochromeSprites, it.monochromeSpritesStart, func(v *MonochromeSprite) DrawOrder { return v.Order }, primitiveKindMonochromeSprite),
		peekOrder(s.SubpixelSprites, it.subpixelSpritesStart, func(v *SubpixelSprite) DrawOrder { return v.Order }, primitiveKindSubpixelSprite),
		peekOrder(s.PolychromeSprites, it.polychromeSpritesStart, func(v *PolychromeSprite) DrawOrder { return v.Order }, primitiveKindPolychromeSprite),
		peekOrder(s.Surfaces, it.surfacesStart, func(v *PaintSurface) DrawOrder { return v.Order }, primitiveKindSurface),
		peekOrder(s.NativeViews, it.nativeViewsStart, func(v *PaintNativeView) DrawOrder { return v.Order }, primitiveKindNativeView),
	}
	slices.SortFunc(ordersAndKinds[:], func(a, b batchOrderAndKind) int {
		if c := cmp.Compare(a.sortOrder(), b.sortOrder()); c != 0 {
			return c
		}
		return cmp.Compare(a.kind, b.kind)
	})

	first := ordersAndKinds[0]
	second := ordersAndKinds[1]
	if !first.some {
		return nil, false
	}
	batchKind := first.kind
	maxOrder, maxKind := second.sortOrder(), second.kind
	below := func(order DrawOrder) bool { return orderAndKindLess(order, batchKind, maxOrder, maxKind) }

	switch batchKind {
	case primitiveKindShadow:
		start := it.shadowsStart
		end := start + 1
		for end < len(s.Shadows) && below(s.Shadows[end].Order) {
			end++
		}
		it.shadowsStart = end
		return PrimitiveBatchShadows{Range{Start: start, End: end}}, true
	case primitiveKindQuad:
		start := it.quadsStart
		end := start + 1
		for end < len(s.Quads) && below(s.Quads[end].Order) {
			end++
		}
		it.quadsStart = end
		return PrimitiveBatchQuads{Range{Start: start, End: end}}, true
	case primitiveKindPath:
		start := it.pathsStart
		end := start + 1
		for end < len(s.Paths) && below(s.Paths[end].order) {
			end++
		}
		it.pathsStart = end
		return PrimitiveBatchPaths{Range{Start: start, End: end}}, true
	case primitiveKindUnderline:
		start := it.underlinesStart
		end := start + 1
		for end < len(s.Underlines) && below(s.Underlines[end].Order) {
			end++
		}
		it.underlinesStart = end
		return PrimitiveBatchUnderlines{Range{Start: start, End: end}}, true
	case primitiveKindMonochromeSprite:
		start := it.monochromeSpritesStart
		textureID := s.MonochromeSprites[start].Tile.TextureID
		end := start + 1
		for end < len(s.MonochromeSprites) && below(s.MonochromeSprites[end].Order) &&
			s.MonochromeSprites[end].Tile.TextureID == textureID {
			end++
		}
		it.monochromeSpritesStart = end
		return PrimitiveBatchMonochromeSprites{TextureID: textureID, Range: Range{Start: start, End: end}}, true
	case primitiveKindSubpixelSprite:
		start := it.subpixelSpritesStart
		textureID := s.SubpixelSprites[start].Tile.TextureID
		end := start + 1
		for end < len(s.SubpixelSprites) && below(s.SubpixelSprites[end].Order) &&
			s.SubpixelSprites[end].Tile.TextureID == textureID {
			end++
		}
		it.subpixelSpritesStart = end
		return PrimitiveBatchSubpixelSprites{TextureID: textureID, Range: Range{Start: start, End: end}}, true
	case primitiveKindPolychromeSprite:
		start := it.polychromeSpritesStart
		textureID := s.PolychromeSprites[start].Tile.TextureID
		end := start + 1
		for end < len(s.PolychromeSprites) && below(s.PolychromeSprites[end].Order) &&
			s.PolychromeSprites[end].Tile.TextureID == textureID {
			end++
		}
		it.polychromeSpritesStart = end
		return PrimitiveBatchPolychromeSprites{TextureID: textureID, Range: Range{Start: start, End: end}}, true
	case primitiveKindSurface:
		start := it.surfacesStart
		end := start + 1
		for end < len(s.Surfaces) && below(s.Surfaces[end].Order) {
			end++
		}
		it.surfacesStart = end
		return PrimitiveBatchSurfaces{Range{Start: start, End: end}}, true
	default:
		start := it.nativeViewsStart
		end := start + 1
		for end < len(s.NativeViews) && below(s.NativeViews[end].Order) {
			end++
		}
		it.nativeViewsStart = end
		return PrimitiveBatchNativeViews{Range{Start: start, End: end}}, true
	}
}

// PrimitiveBatch is a run of primitives of one kind that the renderer draws together:
// PrimitiveBatchShadows, PrimitiveBatchQuads, PrimitiveBatchPaths, PrimitiveBatchUnderlines,
// PrimitiveBatchMonochromeSprites, PrimitiveBatchSubpixelSprites,
// PrimitiveBatchPolychromeSprites, PrimitiveBatchSurfaces or PrimitiveBatchNativeViews.
type PrimitiveBatch interface {
	isBatch()
	// Label describes the batch for debugging and profiling.
	Label() string
}

// PrimitiveBatchShadows is a range of Scene.Shadows.
type PrimitiveBatchShadows struct{ Range }

// PrimitiveBatchQuads is a range of Scene.Quads.
type PrimitiveBatchQuads struct{ Range }

// PrimitiveBatchPaths is a range of Scene.Paths.
type PrimitiveBatchPaths struct{ Range }

// PrimitiveBatchUnderlines is a range of Scene.Underlines.
type PrimitiveBatchUnderlines struct{ Range }

// PrimitiveBatchMonochromeSprites is a range of Scene.MonochromeSprites on one atlas texture.
type PrimitiveBatchMonochromeSprites struct {
	// TextureID is the atlas texture every sprite in the range samples.
	TextureID AtlasTextureID
	// Range is the range of Scene.MonochromeSprites.
	Range Range
}

// PrimitiveBatchSubpixelSprites is a range of Scene.SubpixelSprites on one atlas texture.
type PrimitiveBatchSubpixelSprites struct {
	// TextureID is the atlas texture every sprite in the range samples.
	TextureID AtlasTextureID
	// Range is the range of Scene.SubpixelSprites.
	Range Range
}

// PrimitiveBatchPolychromeSprites is a range of Scene.PolychromeSprites on one atlas texture.
type PrimitiveBatchPolychromeSprites struct {
	// TextureID is the atlas texture every sprite in the range samples.
	TextureID AtlasTextureID
	// Range is the range of Scene.PolychromeSprites.
	Range Range
}

// PrimitiveBatchSurfaces is a range of Scene.Surfaces.
type PrimitiveBatchSurfaces struct{ Range }

// PrimitiveBatchNativeViews is a range of Scene.NativeViews.
type PrimitiveBatchNativeViews struct{ Range }

func (PrimitiveBatchShadows) isBatch()           {}
func (PrimitiveBatchQuads) isBatch()             {}
func (PrimitiveBatchPaths) isBatch()             {}
func (PrimitiveBatchUnderlines) isBatch()        {}
func (PrimitiveBatchMonochromeSprites) isBatch() {}
func (PrimitiveBatchSubpixelSprites) isBatch()   {}
func (PrimitiveBatchPolychromeSprites) isBatch() {}
func (PrimitiveBatchSurfaces) isBatch()          {}
func (PrimitiveBatchNativeViews) isBatch()       {}

func rangeLen(r Range) int { return r.End - r.Start }

// Label describes the batch.
func (b PrimitiveBatchShadows) Label() string { return fmt.Sprintf("shadows (%d)", rangeLen(b.Range)) }

// Label describes the batch.
func (b PrimitiveBatchQuads) Label() string { return fmt.Sprintf("quads (%d)", rangeLen(b.Range)) }

// Label describes the batch.
func (b PrimitiveBatchPaths) Label() string { return fmt.Sprintf("paths (%d)", rangeLen(b.Range)) }

// Label describes the batch.
func (b PrimitiveBatchUnderlines) Label() string {
	return fmt.Sprintf("underlines (%d)", rangeLen(b.Range))
}

// Label describes the batch.
func (b PrimitiveBatchMonochromeSprites) Label() string {
	return fmt.Sprintf("monochrome sprites (%d) on atlas %d", rangeLen(b.Range), b.TextureID.Index)
}

// Label describes the batch.
func (b PrimitiveBatchSubpixelSprites) Label() string {
	return fmt.Sprintf("subpixel sprites (%d) on atlas %d", rangeLen(b.Range), b.TextureID.Index)
}

// Label describes the batch.
func (b PrimitiveBatchPolychromeSprites) Label() string {
	return fmt.Sprintf("polychrome sprites (%d) on atlas %d", rangeLen(b.Range), b.TextureID.Index)
}

// Label describes the batch.
func (b PrimitiveBatchSurfaces) Label() string {
	return fmt.Sprintf("surfaces (%d)", rangeLen(b.Range))
}

// Label describes the batch.
func (b PrimitiveBatchNativeViews) Label() string {
	return fmt.Sprintf("native views (%d)", rangeLen(b.Range))
}

// Quad is a filled, bordered rectangle; its layout is shared with the GPU.
type Quad struct {
	// Order is the quad's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// BorderStyle is the style of the border.
	BorderStyle BorderStyle
	// Bounds are the quad's bounds.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the quad.
	ContentMask ContentMask[ScaledPixels]
	// Background is the quad's fill.
	Background Background
	// BorderColor is the color of the border.
	BorderColor Hsla
	// CornerRadii are the radii of the corners.
	CornerRadii Corners[ScaledPixels]
	// BorderWidths are the widths of the border on each edge.
	BorderWidths Edges[ScaledPixels]
}

// IntoPrimitive wraps the quad as a Primitive.
func (q Quad) IntoPrimitive() Primitive { return PrimitiveQuad{q} }

// Underline is a straight or wavy underline; its layout is shared with the GPU.
type Underline struct {
	// Order is the underline's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// Pad aligns the struct to 8 bytes.
	Pad uint32
	// Bounds are the underline's bounds.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the underline.
	ContentMask ContentMask[ScaledPixels]
	// Color is the underline's color.
	Color Hsla
	// Thickness is the line's thickness.
	Thickness ScaledPixels
	// Wavy reports whether the underline is wavy.
	Wavy PaddedBool32
}

// IntoPrimitive wraps the underline as a Primitive.
func (u Underline) IntoPrimitive() Primitive { return PrimitiveUnderline{u} }

// Shadow is a blurred rounded-rectangle shadow; its layout is shared with the GPU.
type Shadow struct {
	// Order is the shadow's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// BlurRadius is the blur radius.
	BlurRadius ScaledPixels
	// Bounds are the shadow's bounds.
	Bounds Bounds[ScaledPixels]
	// CornerRadii are the radii of the shadow's corners.
	CornerRadii Corners[ScaledPixels]
	// ContentMask clips the shadow.
	ContentMask ContentMask[ScaledPixels]
	// Color is the shadow's color.
	Color Hsla
	// ShapeBounds are the bounds of the shape casting the shadow.
	ShapeBounds Bounds[ScaledPixels]
	// ShapeCornerRadii are the corner radii of the shape casting the shadow.
	ShapeCornerRadii Corners[ScaledPixels]
	// Inset reports whether the shadow is drawn inside the shape.
	Inset PaddedBool32
	// Pad aligns the struct to 8 bytes.
	Pad uint32
}

// IntoPrimitive wraps the shadow as a Primitive.
func (s Shadow) IntoPrimitive() Primitive { return PrimitiveShadow{s} }

// BorderStyle is the style of a border; it is 4 bytes wide because it is GPU data (D-GPUENUM).
type BorderStyle uint32

const (
	// BorderStyleSolid is a solid border.
	BorderStyleSolid BorderStyle = 0
	// BorderStyleDashed is a dashed border.
	BorderStyleDashed BorderStyle = 1
)

var borderStyleNames = [...]string{"Solid", "Dashed"}

// String formats the border style like its Rust Debug output.
func (b BorderStyle) String() string {
	if int(b) < len(borderStyleNames) {
		return borderStyleNames[b]
	}
	return fmt.Sprintf("BorderStyle(%d)", uint32(b))
}

// MarshalJSONTo encodes the border style as its variant name.
func (b BorderStyle) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(b.String()))
}

// UnmarshalJSONFrom decodes the border style from its variant name.
func (b *BorderStyle) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	i, err := readJSONVariant(dec, borderStyleNames[:])
	if err != nil {
		return err
	}
	*b = BorderStyle(i)
	return nil
}

// JSONSchema describes the border style as one of its variant names.
func (BorderStyle) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	enum := make([]any, len(borderStyleNames))
	for i, name := range borderStyleNames {
		enum[i] = name
	}
	return &jsonschema.Schema{Description: "The style of a border.", Type: jsonschema.Types{"string"}, Enum: enum}
}

// TransformationMatrix is a 2D transformation applied to an element; its layout is shared with the GPU.
// The zero value is not the unit matrix: use TransformationMatrixUnit.
type TransformationMatrix struct {
	// RotationScale is the 2x2 rotation and scale matrix, stored row-major.
	RotationScale [2][2]float32
	// Translation is the translation vector.
	Translation [2]float32
}

// TransformationMatrixUnit returns the unit matrix, which has no effect.
func TransformationMatrixUnit() TransformationMatrix {
	return TransformationMatrix{
		RotationScale: [2][2]float32{{1.0, 0.0}, {0.0, 1.0}},
		Translation:   [2]float32{0.0, 0.0},
	}
}

// DefaultTransformationMatrix returns the unit matrix.
func DefaultTransformationMatrix() TransformationMatrix {
	return TransformationMatrixUnit()
}

// Translate moves the origin by the given point.
func (m TransformationMatrix) Translate(point Point[ScaledPixels]) TransformationMatrix {
	return m.Compose(TransformationMatrix{
		RotationScale: [2][2]float32{{1.0, 0.0}, {0.0, 1.0}},
		Translation:   [2]float32{float32(point.X), float32(point.Y)},
	})
}

// Rotate rotates clockwise by angle radians around the origin.
func (m TransformationMatrix) Rotate(angle Radians) TransformationMatrix {
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))
	return m.Compose(TransformationMatrix{
		RotationScale: [2][2]float32{{cos, -sin}, {sin, cos}},
		Translation:   [2]float32{0.0, 0.0},
	})
}

// Scale scales around the origin.
func (m TransformationMatrix) Scale(size Size[float32]) TransformationMatrix {
	return m.Compose(TransformationMatrix{
		RotationScale: [2][2]float32{{size.Width, 0.0}, {0.0, size.Height}},
		Translation:   [2]float32{0.0, 0.0},
	})
}

// Compose returns the transformation that applies other first, then m.
func (m TransformationMatrix) Compose(other TransformationMatrix) TransformationMatrix {
	if other == TransformationMatrixUnit() {
		return m
	}
	a := m.RotationScale
	b := other.RotationScale
	return TransformationMatrix{
		RotationScale: [2][2]float32{
			{
				float32(a[0][0]*b[0][0]) + float32(a[0][1]*b[1][0]),
				float32(a[0][0]*b[0][1]) + float32(a[0][1]*b[1][1]),
			},
			{
				float32(a[1][0]*b[0][0]) + float32(a[1][1]*b[1][0]),
				float32(a[1][0]*b[0][1]) + float32(a[1][1]*b[1][1]),
			},
		},
		Translation: [2]float32{
			float32(m.Translation[0]+float32(a[0][0]*other.Translation[0])) + float32(a[0][1]*other.Translation[1]),
			float32(m.Translation[1]+float32(a[1][0]*other.Translation[0])) + float32(a[1][1]*other.Translation[1]),
		},
	}
}

func (m TransformationMatrix) transformBounds(bounds Bounds[ScaledPixels]) Bounds[ScaledPixels] {
	if m == TransformationMatrixUnit() {
		return bounds
	}
	xx, xy := m.RotationScale[0][0], m.RotationScale[0][1]
	yx, yy := m.RotationScale[1][0], m.RotationScale[1][1]
	translateX, translateY := m.Translation[0], m.Translation[1]
	corners := [4]Point[ScaledPixels]{bounds.Origin, bounds.TopRight(), bounds.BottomLeft(), bounds.BottomRight()}
	for i, corner := range corners {
		x, y := float32(corner.X), float32(corner.Y)
		corners[i] = Point[ScaledPixels]{
			X: ScaledPixels(float32(float32(xx*x)+float32(xy*y)) + translateX),
			Y: ScaledPixels(float32(float32(yx*x)+float32(yy*y)) + translateY),
		}
	}
	topLeft := corners[0]
	bottomRight := corners[0]
	for _, corner := range corners[1:] {
		topLeft = topLeft.Min(corner)
		bottomRight = bottomRight.Max(corner)
	}
	return BoundsFromCorners(topLeft, bottomRight)
}

// Apply applies the transformation to a point.
func (m TransformationMatrix) Apply(point Point[Pixels]) Point[Pixels] {
	input := [2]float32{float32(point.X), float32(point.Y)}
	output := m.Translation
	for i := range output {
		for k := range input {
			output[i] += float32(m.RotationScale[i][k] * input[k])
		}
	}
	return Point[Pixels]{X: Pixels(output[0]), Y: Pixels(output[1])}
}

// MonochromeSprite is a single-channel sprite tinted with a color; its layout is shared with the GPU.
type MonochromeSprite struct {
	// Order is the sprite's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// Animation is the scene animation moving the sprite; set it to SpriteAnimationIDNone.
	Animation SpriteAnimationID
	// Tile is the sprite's atlas tile.
	Tile AtlasTile
	// Bounds are the sprite's bounds.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the sprite.
	ContentMask ContentMask[ScaledPixels]
	// Color is the sprite's color.
	Color Hsla
	// Transformation is applied to the sprite; set it to TransformationMatrixUnit for none.
	Transformation TransformationMatrix
}

// IntoPrimitive wraps the sprite as a Primitive.
func (s MonochromeSprite) IntoPrimitive() Primitive { return PrimitiveMonochromeSprite{s} }

// SpriteAnimationID refers to an animation of the scene a sprite is inserted into.
// It fits the sprite's padding, so renderers ignore it. The zero value is not None.
type SpriteAnimationID uint32

// SpriteAnimationIDNone is no animation.
const SpriteAnimationIDNone SpriteAnimationID = math.MaxUint32

// DefaultSpriteAnimationID returns SpriteAnimationIDNone.
func DefaultSpriteAnimationID() SpriteAnimationID {
	return SpriteAnimationIDNone
}

type spriteAnimation struct {
	transformation TimedTransformation
	center         Point[Pixels]
	scaleFactor    float32
	baseColor      Hsla
	spriteIndex    int
}

func newSpriteAnimation(transformation TimedTransformation, center Point[Pixels], scaleFactor float32) spriteAnimation {
	return spriteAnimation{
		transformation: transformation,
		center:         center,
		scaleFactor:    scaleFactor,
		baseColor:      Hsla{},
		spriteIndex:    math.MaxInt,
	}
}

// SubpixelSprite is a sprite with per-channel coverage; its layout is shared with the GPU.
type SubpixelSprite struct {
	// Order is the sprite's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// Pad aligns the struct to 8 bytes.
	Pad uint32
	// Tile is the sprite's atlas tile.
	Tile AtlasTile
	// Bounds are the sprite's bounds.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the sprite.
	ContentMask ContentMask[ScaledPixels]
	// Color is the sprite's color.
	Color Hsla
	// Transformation is applied to the sprite; set it to TransformationMatrixUnit for none.
	Transformation TransformationMatrix
}

// IntoPrimitive wraps the sprite as a Primitive.
func (s SubpixelSprite) IntoPrimitive() Primitive { return PrimitiveSubpixelSprite{s} }

// PolychromeSprite is a full-color sprite; its layout is shared with the GPU.
type PolychromeSprite struct {
	// Order is the sprite's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// Pad aligns the tile to 8 bytes.
	Pad uint32
	// Tile is the sprite's atlas tile.
	Tile AtlasTile
	// Grayscale reports whether the sprite is drawn in grayscale.
	Grayscale PaddedBool32
	// Opacity is the sprite's opacity.
	Opacity float32
	// Bounds are the sprite's bounds.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the sprite.
	ContentMask ContentMask[ScaledPixels]
	// CornerRadii are the radii of the sprite's corners.
	CornerRadii Corners[ScaledPixels]
}

// IntoPrimitive wraps the sprite as a Primitive.
func (s PolychromeSprite) IntoPrimitive() Primitive { return PrimitivePolychromeSprite{s} }

// PaintSurface is a platform video surface painted into the scene.
type PaintSurface struct {
	// Order is the surface's draw order, assigned when it is inserted into a Scene.
	Order DrawOrder
	// Bounds are the surface's bounds.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the surface.
	ContentMask ContentMask[ScaledPixels]
	// ImageBuffer is the CVPixelBufferRef shown on macOS and iOS, zero elsewhere.
	// The Apple back end owns its retain and release.
	ImageBuffer uintptr
}

// IntoPrimitive wraps the surface as a Primitive.
func (s PaintSurface) IntoPrimitive() Primitive { return PrimitiveSurface{s} }

// PaintNativeView is where a platform view shows through the scene. The renderer erases
// what was painted earlier within the bounds; what is painted later covers the view.
type PaintNativeView struct {
	// Order is the view's position among the scene's other primitives.
	Order DrawOrder
	// ID is the view these bounds belong to.
	ID NativeViewID
	// HitboxID is the hitbox that decides whether a mouse position reaches the view.
	HitboxID HitboxID
	// Bounds is where the view is laid out, before clipping.
	Bounds Bounds[ScaledPixels]
	// ContentMask is the clip applied to the view.
	ContentMask ContentMask[ScaledPixels]
	// Background is shown where the view has not caught up with these bounds; transparent for none.
	Background Hsla
}

// IntoPrimitive wraps the native view as a Primitive.
func (v PaintNativeView) IntoPrimitive() Primitive { return PrimitiveNativeView{v} }

// PathCurve is a monotone quadratic curve of a path fill; its layout is shared with the GPU.
type PathCurve struct {
	// P0 is the curve's upper endpoint.
	P0 Point[float32]
	// P1 is the curve's lower endpoint.
	P1 Point[float32]
	// A is the quadratic coefficient.
	A Point[float32]
	// B is the linear coefficient.
	B Point[float32]
	// Sign is 1 for a downward curve and -1 for an upward one.
	Sign float32
	// Pad aligns the struct to 8 bytes.
	Pad uint32
}

// PathPaint is how a path's tiles are filled; its layout is shared with the GPU.
type PathPaint struct {
	// Bounds are the path's bounds, in device pixels.
	Bounds Bounds[ScaledPixels]
	// ContentMask clips the path.
	ContentMask ContentMask[ScaledPixels]
	// Color is the path's fill.
	Color Background
	// EvenOdd reports whether the path is filled by the even-odd rule.
	EvenOdd PaddedBool32
	// Pad aligns the struct to 8 bytes.
	Pad uint32
}

// PathTile is a rectangle of a path that one curve list shades; its layout is shared with the GPU.
type PathTile struct {
	// Paint is the index of the tile's PathPaint.
	Paint uint32
	// CurveStart is the index of the tile's first curve.
	CurveStart uint32
	// CurveCount is the number of curves the tile lists.
	CurveCount uint32
	// Backdrop is the winding at Reference of the curves the tile does not list.
	Backdrop int32
	// TopLeft is the tile's top-left corner.
	TopLeft Point[float32]
	// BottomRight is the tile's bottom-right corner.
	BottomRight Point[float32]
	// Reference is the point the backdrop is measured at.
	Reference Point[float32]
	// CurvesTopLeft is the top-left corner of the box around the tile's curves.
	CurvesTopLeft Point[float32]
	// CurvesBottomRight is the bottom-right corner of the box around the tile's curves.
	CurvesBottomRight Point[float32]
}

// PaintedPath is a path binned into tiles, ready to be drawn.
type PaintedPath struct {
	order  DrawOrder
	bounds Bounds[ScaledPixels]
	tiles  *pathTiles
	paint  PathPaint
}

// NewPaintedPath bins path into tiles, to be filled with color within contentMask.
func NewPaintedPath(path Path, scaleFactor float32, contentMask ContentMask[Pixels], color Background) PaintedPath {
	scaledMask := ContentMask[ScaledPixels]{Bounds: contentMask.Bounds.Scale(scaleFactor)}
	tiles := path.bin(scaleFactor)
	tiles.retainVisible(scaledMask.Bounds)
	paint := PathPaint{
		Bounds:      tiles.bounds(),
		ContentMask: scaledMask,
		Color:       color,
		EvenOdd:     PaddedBool32FromBool(path.evenOdd()),
		Pad:         0,
	}
	return PaintedPath{
		order:  0,
		bounds: tiles.paintedBounds(),
		tiles:  tiles,
		paint:  paint,
	}
}

// IntoPrimitive wraps the path as a Primitive.
func (p PaintedPath) IntoPrimitive() Primitive { return PrimitivePath{p} }

// TileCount returns the number of tiles the path is drawn with.
func (p PaintedPath) TileCount() int {
	return len(p.tiles.tiles())
}

// AppendTo appends the path's paint, curves and tiles to the renderer's upload buffers.
func (p PaintedPath) AppendTo(paints *[]PathPaint, curves *[]PathCurve, tiles *[]PathTile) {
	paintIndex := uint32(len(*paints))
	curveStart := uint32(len(*curves))
	*paints = append(*paints, p.paint)
	*curves = append(*curves, p.tiles.curves()...)
	for _, tile := range p.tiles.tiles() {
		tile.Paint = paintIndex
		tile.CurveStart = curveStart + tile.CurveStart
		*tiles = append(*tiles, tile)
	}
}
