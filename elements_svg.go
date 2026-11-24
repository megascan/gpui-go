package gpui

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"os"
	"time"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// Svg is an SVG element.
type Svg struct {
	Styled[*Svg]
	Interactive[*Svg]
	InteractiveID[*Svg]
	Fluent[*Svg]
	interactivity  Interactivity
	transformation Option[TimedTransformation]
	path           Option[SharedString]
	externalPath   Option[SharedString]
	data           []byte
	dataPath       Option[SharedString]
}

// NewSvg creates a new SVG element.
func NewSvg() *Svg {
	s := &Svg{
		transformation: None[TimedTransformation](),
		path:           None[SharedString](),
		externalPath:   None[SharedString](),
		dataPath:       None[SharedString](),
	}
	s.interactivity.sourceLocation = CallerSourceLocation(1)
	BindStyled(&s.Styled, s)
	BindInteractive(&s.Interactive, s)
	BindInteractiveID(&s.InteractiveID, s)
	BindFluent(&s.Fluent, s)
	return s
}

// Path sets the path to the SVG file for this element.
func (s *Svg) Path(path SharedString) *Svg {
	s.path = Some(path)
	return s
}

// ExternalPath sets the path to the SVG file for this element.
func (s *Svg) ExternalPath(path SharedString) *Svg {
	s.externalPath = Some(path)
	return s
}

// Data sets the raw SVG data for this element. The SVG will be rendered directly from the
// provided bytes.
func (s *Svg) Data(data []byte) *Svg {
	var lengthPrefix [8]byte
	binary.LittleEndian.PutUint64(lengthPrefix[:], uint64(len(data)))
	message := make([]byte, 0, len(lengthPrefix)+len(data))
	message = append(message, lengthPrefix[:]...)
	message = append(message, data...)
	hash := sipHash13(message)
	path := SharedString(fmt.Sprintf("__binary_svg__%d", hash))
	s.data = append(make([]byte, 0, len(data)), data...)
	s.dataPath = Some(path)
	return s
}

// WithTransformation transforms the SVG element with the given transformation. Note that this
// won't effect the hitbox or layout of the element, only the rendering.
func (s *Svg) WithTransformation(transformation Transformation) *Svg {
	s.transformation = Some(TimedTransformationFromTransformation(transformation))
	return s
}

// WithTimedTransformation continuously animates the SVG between two transformations,
// phase-locked to the app's shared animation clock like AnimationRepeatSynced.
//
// The SVG is painted once. On later frames the window moves it in the scene it already has and
// presents that again, instead of re-rendering this view.
//
// This replaces WithTransformation, and like it doesn't affect layout or hit testing. The SVG is
// still clipped and ordered by the bounds of its starting transformation, so keep its motion
// close to them.
func (s *Svg) WithTimedTransformation(transformation TimedTransformation) *Svg {
	s.transformation = Some(transformation)
	return s
}

// ElementID returns the element's id, if it has one.
func (s *Svg) ElementID() (ElementID, bool) {
	return s.interactivity.ElementID.Get()
}

// SourceLocation returns where the element was created.
func (s *Svg) SourceLocation() *SourceLocation {
	return s.interactivity.SourceLocation()
}

// RequestLayout requests the element's layout.
func (s *Svg) RequestLayout(globalID *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	layoutID := s.interactivity.RequestLayout(globalID, inspectorID, window, cx, func(style Style, window *Window, cx *App) LayoutID {
		return window.RequestLayout(style, nil, cx)
	})
	return layoutID, struct{}{}
}

// Prepaint prepaints the element's interactivity.
func (s *Svg) Prepaint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, cx *App) Option[Hitbox] {
	return s.interactivity.Prepaint(globalID, inspectorID, bounds, bounds.Size, window, cx, func(_ *Style, _ Point[Pixels], hitbox Option[Hitbox], _ *Window, _ *App) Option[Hitbox] {
		return hitbox
	})
}

// Paint paints the SVG.
func (s *Svg) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, hitbox *Option[Hitbox], window *Window, cx *App) {
	var hitboxRef *Hitbox
	if hitbox.IsSome() {
		hitboxRef = hitbox.AsMut()
	}
	s.interactivity.Paint(globalID, inspectorID, bounds, hitboxRef, window, cx, func(style *Style, window *Window, cx *App) {
		center := bounds.Center()
		scaleFactor := window.ScaleFactor()
		animation := None[spriteAnimation]()
		if timed, ok := s.transformation.Get(); ok && timed.isAnimated() && !cx.ReduceMotion() {
			animation = Some(newSpriteAnimation(timed, center, scaleFactor))
		}
		pose := None[Transformation]()
		if timed, ok := s.transformation.Get(); ok {
			pose = Some(timed.from)
		}
		transformation := TransformationMatrixUnit()
		if p, ok := pose.Get(); ok {
			transformation = p.intoMatrix(center, scaleFactor)
		}
		alpha := float32(1.)
		if animation.IsNone() {
			if p, ok := pose.Get(); ok {
				alpha = p.alphaFactor()
			}
		}

		color, hasColor := style.Text.Color.Get()
		if dataPath, ok := s.dataPath.Get(); ok {
			if hasColor {
				if err := window.paintAnimatedSvg(bounds, dataPath, s.data, transformation, animation, color.Opacity(alpha), cx); err != nil {
					gpuiutil.LogErr(err)
				}
			}
		} else if path, ok := s.externalPath.Get(); ok && hasColor {
			result, ok := window.UseAsset[svgAsset](path, cx)
			if !ok {
				return
			}
			if result.Err != nil {
				gpuiutil.LogErr(result.Err)
				return
			}
			if err := window.paintAnimatedSvg(bounds, path, result.Value, transformation, animation, color.Opacity(alpha), cx); err != nil {
				gpuiutil.LogErr(err)
			}
		} else if path, ok := s.path.Get(); ok && hasColor {
			if err := window.paintAnimatedSvg(bounds, path, nil, transformation, animation, color.Opacity(alpha), cx); err != nil {
				gpuiutil.LogErr(err)
			}
		}
	})
}

// Style returns the element's base style.
func (s *Svg) Style() *StyleRefinement {
	return &s.interactivity.BaseStyle
}

// Interactivity returns the element's interactivity.
func (s *Svg) Interactivity() *Interactivity {
	return &s.interactivity
}

// IntoAnyElement converts the SVG into an AnyElement.
func (s *Svg) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, Option[Hitbox]](s)
}

// Transformation is a transformation to apply to an SVG element.
type Transformation struct {
	scale     Size[float32]
	translate Point[Pixels]
	rotate    Radians
	alpha     float32
}

// DefaultTransformation returns the identity transformation.
func DefaultTransformation() Transformation {
	return Transformation{
		scale:     NewSize[float32](1.0, 1.0),
		translate: NewPoint(Px(0.0), Px(0.0)),
		rotate:    Radians(0.0),
		alpha:     1.0,
	}
}

// TransformationScale creates a new Transformation with the specified scale along each axis.
func TransformationScale(scale Size[float32]) Transformation {
	t := DefaultTransformation()
	t.scale = scale
	return t
}

// TransformationTranslate creates a new Transformation with the specified translation.
func TransformationTranslate(translate Point[Pixels]) Transformation {
	t := DefaultTransformation()
	t.translate = translate
	return t
}

// TransformationRotate creates a new Transformation with the specified rotation in radians.
func TransformationRotate(rotate Radians) Transformation {
	t := DefaultTransformation()
	t.rotate = rotate
	return t
}

// TransformationAlpha creates a new Transformation that multiplies the SVG's alpha by alpha.
func TransformationAlpha(alpha float32) Transformation {
	t := DefaultTransformation()
	t.alpha = alpha
	return t
}

// WithAlpha updates the alpha factor of this transformation. Unlike an element's opacity, this
// only applies to the SVG itself.
func (t Transformation) WithAlpha(alpha float32) Transformation {
	t.alpha = alpha
	return t
}

func (t Transformation) alphaFactor() float32 {
	return t.alpha
}

// WithScaling updates the scaling factor of this transformation.
func (t Transformation) WithScaling(scale Size[float32]) Transformation {
	t.scale = scale
	return t
}

// WithTranslation updates the translation value of this transformation.
func (t Transformation) WithTranslation(translate Point[Pixels]) Transformation {
	t.translate = translate
	return t
}

// WithRotation updates the rotation angle of this transformation.
func (t Transformation) WithRotation(rotate Radians) Transformation {
	t.rotate = rotate
	return t
}

func (t Transformation) intoMatrix(center Point[Pixels], scaleFactor float32) TransformationMatrix {
	return TransformationMatrixUnit().
		Translate(center.Scale(scaleFactor).Add(t.translate.Scale(scaleFactor))).
		Rotate(t.rotate).
		Scale(t.scale).
		Translate(center.Scale(-scaleFactor))
}

// String formats the transformation like its Rust Debug output.
func (t Transformation) String() string {
	return fmt.Sprintf("Transformation { scale: %v, translate: %v, rotate: %v, alpha: %v }", t.scale, t.translate, t.rotate, t.alpha)
}

// Easing is how a value's rate of change varies over one pass of a TimedTransformation.
type Easing uint8

const (
	// EasingLinear is a constant rate.
	EasingLinear Easing = iota
	// EasingQuadratic starts slowly and accelerates.
	EasingQuadratic
	// EasingEaseInOut starts and ends slowly, and speeds up in the middle.
	EasingEaseInOut
	// EasingEaseOutQuint starts quickly and decelerates to a stop.
	EasingEaseOutQuint
	// EasingHeartbeat is two quick beats, a strong one then a weaker one, followed by a rest.
	// Unlike the other easings it ends where it started, so the value reaches `to` at the first
	// beat and is back at `from` by the end.
	EasingHeartbeat
)

var easingNames = [...]string{"Linear", "Quadratic", "EaseInOut", "EaseOutQuint", "Heartbeat"}

// String formats the easing like its Rust Debug output.
func (e Easing) String() string { return variantName(easingNames[:], e, "Easing") }

func (e Easing) apply(delta float32) float32 {
	switch e {
	case EasingQuadratic:
		return Quadratic(delta)
	case EasingEaseInOut:
		return EaseInOut(delta)
	case EasingEaseOutQuint:
		return EaseOutQuint()(delta)
	case EasingHeartbeat:
		return Heartbeat(delta)
	default:
		return Linear(delta)
	}
}

// TimedTransformation is a Transformation that repeatedly moves from one pose to another over a
// period, for Svg.WithTimedTransformation.
//
// Each value of the transformation has its own Easing, so for example an SVG can grow quickly at
// first while fading at a constant rate.
type TimedTransformation struct {
	period            time.Duration
	from              Transformation
	to                Transformation
	scaleEasing       Easing
	translationEasing Easing
	rotationEasing    Easing
	alphaEasing       Easing
	alternate         bool
	frameInterval     Option[time.Duration]
}

// NewTimedTransformation creates a timed transformation that repeats every period. It animates
// nothing until From and To differ.
func NewTimedTransformation(period time.Duration) TimedTransformation {
	return TimedTransformation{
		period:            period,
		from:              DefaultTransformation(),
		to:                DefaultTransformation(),
		scaleEasing:       EasingLinear,
		translationEasing: EasingLinear,
		rotationEasing:    EasingLinear,
		alphaEasing:       EasingLinear,
		alternate:         false,
		frameInterval:     None[time.Duration](),
	}
}

// TimedTransformationRotation creates a timed transformation that turns from clockwise, one full
// turn per period.
func TimedTransformationRotation(period time.Duration, from Transformation) TimedTransformation {
	return NewTimedTransformation(period).
		From(from).
		To(from.WithRotation(Radians(float32(from.rotate) + float32(2*math.Pi))))
}

// TimedTransformationFromTransformation returns a timed transformation that stays at
// transformation.
func TimedTransformationFromTransformation(transformation Transformation) TimedTransformation {
	return NewTimedTransformation(0).From(transformation).To(transformation)
}

func (t TimedTransformation) isAnimated() bool {
	return t.period != 0 && t.from != t.to
}

// From sets the transformation at the start of each period.
func (t TimedTransformation) From(transformation Transformation) TimedTransformation {
	t.from = transformation
	return t
}

// To sets the transformation reached at the end of each period.
func (t TimedTransformation) To(transformation Transformation) TimedTransformation {
	t.to = transformation
	return t
}

// WithEasing sets the easing of every value.
func (t TimedTransformation) WithEasing(easing Easing) TimedTransformation {
	t.scaleEasing = easing
	t.translationEasing = easing
	t.rotationEasing = easing
	t.alphaEasing = easing
	return t
}

// WithScaleEasing sets the easing of the scale.
func (t TimedTransformation) WithScaleEasing(easing Easing) TimedTransformation {
	t.scaleEasing = easing
	return t
}

// WithTranslationEasing sets the easing of the translation.
func (t TimedTransformation) WithTranslationEasing(easing Easing) TimedTransformation {
	t.translationEasing = easing
	return t
}

// WithRotationEasing sets the easing of the rotation.
func (t TimedTransformation) WithRotationEasing(easing Easing) TimedTransformation {
	t.rotationEasing = easing
	return t
}

// WithAlphaEasing sets the easing of the alpha.
func (t TimedTransformation) WithAlphaEasing(easing Easing) TimedTransformation {
	t.alphaEasing = easing
	return t
}

// Alternate moves from the start to the end and back again within each period, instead of
// jumping back to the start.
func (t TimedTransformation) Alternate() TimedTransformation {
	t.alternate = true
	return t
}

// WithMaxFps presents at most maxFps frames per second while nothing but timed transformations
// needs a frame. The least restrictive limit in a window wins. Values that are not finite and
// positive are ignored.
func (t TimedTransformation) WithMaxFps(maxFps float32) TimedTransformation {
	if !math.IsInf(float64(maxFps), 0) && !math.IsNaN(float64(maxFps)) && maxFps > 0.0 {
		t.frameInterval = Some(time.Duration(1.0 / float64(maxFps) * 1e9))
	}
	return t
}

func (t TimedTransformation) minFrameInterval() (time.Duration, bool) {
	return t.frameInterval.Get()
}

func (t TimedTransformation) at(elapsed time.Duration) Transformation {
	periodNanos := uint64(t.period)
	if periodNanos == 0 {
		return t.from
	}
	delta := durationAsSecsF32(time.Duration(uint64(elapsed)%periodNanos)) / durationAsSecsF32(t.period)
	if t.alternate {
		delta = 1.0 - float32(math.Abs(float64(2.0*delta-1.0)))
	}

	interpolate := func(from, to float32, easing Easing) float32 {
		return from + (to-from)*easing.apply(delta)
	}
	from, to := t.from, t.to
	return Transformation{
		scale: NewSize(
			interpolate(from.scale.Width, to.scale.Width, t.scaleEasing),
			interpolate(from.scale.Height, to.scale.Height, t.scaleEasing),
		),
		translate: NewPoint(
			Px(interpolate(float32(from.translate.X), float32(to.translate.X), t.translationEasing)),
			Px(interpolate(float32(from.translate.Y), float32(to.translate.Y), t.translationEasing)),
		),
		rotate: Radians(interpolate(float32(from.rotate), float32(to.rotate), t.rotationEasing)),
		alpha:  interpolate(from.alpha, to.alpha, t.alphaEasing),
	}
}

type svgAsset struct{}

// Load reads the SVG file at source.
func (svgAsset) Load(source SharedString, _ *App) func(s *scheduler.Scope) scheduler.Result[[]byte] {
	return func(*scheduler.Scope) scheduler.Result[[]byte] {
		bytes, err := os.ReadFile(string(source))
		if err != nil {
			return scheduler.Result[[]byte]{Err: err}
		}
		return scheduler.Result[[]byte]{Value: bytes}
	}
}

func sipHash13(message []byte) uint64 {
	v0 := uint64(0x736f6d6570736575)
	v1 := uint64(0x646f72616e646f6d)
	v2 := uint64(0x6c7967656e657261)
	v3 := uint64(0x7465646279746573)
	round := func() {
		v0 += v1
		v1 = bits.RotateLeft64(v1, 13)
		v1 ^= v0
		v0 = bits.RotateLeft64(v0, 32)
		v2 += v3
		v3 = bits.RotateLeft64(v3, 16)
		v3 ^= v2
		v0 += v3
		v3 = bits.RotateLeft64(v3, 21)
		v3 ^= v0
		v2 += v1
		v1 = bits.RotateLeft64(v1, 17)
		v1 ^= v2
		v2 = bits.RotateLeft64(v2, 32)
	}
	length := len(message)
	for len(message) >= 8 {
		m := binary.LittleEndian.Uint64(message)
		v3 ^= m
		round()
		v0 ^= m
		message = message[8:]
	}
	b := uint64(length) << 56
	for i, c := range message {
		b |= uint64(c) << (8 * i)
	}
	v3 ^= b
	round()
	v0 ^= b
	v2 ^= 0xff
	round()
	round()
	round()
	return v0 ^ v1 ^ v2 ^ v3
}
