package gpui

import (
	"encoding/json/jsontext"
	"reflect"
	"sync"
	"time"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/scheduler"
)

// DefaultWindowSize is the default size of a new window (M0-window-core ports it).
var DefaultWindowSize = NewSize(Px(1536.), Px(1095.))

// Keymap is a collection of key bindings (M0-keymap ports it).
type Keymap struct{}

// PlatformGestures are the platform's gesture recognition services (M0-gestures-spring ports it).
type PlatformGestures interface {
	Tuning() GestureTuning
	NativeRecognizers() GestureKinds
}

// GestureTuning holds feel constants for the portable gesture recognizers (M0-gestures-spring ports it).
type GestureTuning struct{}

// GestureKinds is the set of gesture kinds a platform recognizes natively (M0-gestures-spring ports it).
type GestureKinds struct{}

// RenderSvgParams identifies a rasterized SVG in the sprite atlas (M0-elements-media ports it).
type RenderSvgParams struct {
	// Path is the SVG's asset path.
	Path SharedString
	// Size is the rasterized size.
	Size Size[DevicePixels]
}

// WindowControlArea is a window control area that corresponds to the platform window (M0-window-core ports it).
type WindowControlArea uint8

const (
	// WindowControlAreaDrag allows dragging the platform window.
	WindowControlAreaDrag WindowControlArea = iota
	// WindowControlAreaClose allows closing the platform window.
	WindowControlAreaClose
	// WindowControlAreaMax allows maximizing the platform window.
	WindowControlAreaMax
	// WindowControlAreaMin allows minimizing the platform window.
	WindowControlAreaMin
)

// NativeViewID identifies a native view within its window (M0-window-a11y ports it).
type NativeViewID uint64

// IntoAnyElement renders the entity as a view; *T must implement Render[T] (M0-element ports it).
func (e Entity[T]) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Entity.IntoAnyElement")
}

// AnyView is a type-erased handle to an entity that implements Render (M0-element ports it).
type AnyView struct {
	entity AnyEntity
}

func anyViewDowncast[V any](view AnyView) (Entity[V], bool) {
	panic("gpui: unimplemented anyViewDowncast")
}

// View is an entity or component that can render itself (M0-element ports it).
type View interface {
	EntityID() (EntityID, bool)
	Render(window *Window, cx *App) IntoElement
}

// Render is implemented by entity states that can be drawn as views (M0-element ports it).
type Render[T any] interface {
	Render(window *Window, cx *Context[T]) IntoElement
}

// RenderOnce is implemented by components that render once into an element tree (M0-element ports it).
type RenderOnce interface {
	Render(window *Window, cx *App) IntoElement
}

// WindowID is the identifier of a window (M0-window-core ports it).
type WindowID uint64

// AnyWindowHandle is a type-erased handle to a window (M0-window-core ports it).
type AnyWindowHandle struct {
	id        WindowID
	stateType reflect.Type
}

// WindowHandle is a handle to a window whose root view has type V (M0-window-core ports it).
type WindowHandle[V any] struct {
	AnyWindowHandle
}

// Window is a GPUI window (M0-window-core ports it).
type Window struct {
	handle             AnyWindowHandle
	displayID          Option[DisplayID]
	lastWindowBounds   WindowBounds
	root               Option[AnyView]
	focus              Option[FocusID]
	invalidator        windowInvalidator
	removed            bool
	platformWindow     PlatformWindow
	nextFrameCallbacks *[]FrameCallback
	needsPresent       *bool
	refreshing         bool

	displayObservers      singleSubscriberSet[anyObserver]
	focusListeners        singleSubscriberSet[anyWindowFocusListener]
	focusLostListeners    singleSubscriberSet[anyObserver]
	boundsObservers       singleSubscriberSet[anyObserver]
	appearanceObservers   singleSubscriberSet[anyObserver]
	buttonLayoutObservers singleSubscriberSet[anyObserver]
	visibilityObservers   singleSubscriberSet[func(visibility WindowVisibility, window *Window, cx *App) bool]
	activationObservers   singleSubscriberSet[anyObserver]
	pendingInputObservers singleSubscriberSet[anyObserver]
}

type anyObserver func(window *Window, cx *App) bool

type anyWindowFocusListener func(event *windowFocusEvent, window *Window, cx *App) bool

type windowFocusEvent struct {
	previousFocusPath []FocusID
	currentFocusPath  []FocusID
}

func (e *windowFocusEvent) isFocusIn(focusID FocusID) bool {
	panic("gpui: unimplemented windowFocusEvent.isFocusIn")
}

func (e *windowFocusEvent) isFocusOut(focusID FocusID) bool {
	panic("gpui: unimplemented windowFocusEvent.isFocusOut")
}

// FocusOutEvent is provided when subscribing for Context.OnFocusOut events (M0-window-core ports it).
type FocusOutEvent struct {
	// Blurred is a weak focus handle representing what was blurred.
	Blurred WeakFocusHandle
}

// WeakFocusHandle is a weak reference to a focus handle (M0-window-core ports it).
type WeakFocusHandle struct {
	id      FocusID
	handles weak.Pointer[focusMap]
}

func newWeakFocusHandle(id FocusID, handles *focusMap) WeakFocusHandle {
	panic("gpui: unimplemented newWeakFocusHandle")
}

func (w *Window) newFocusListener(value anyWindowFocusListener) (Subscription, func()) {
	panic("gpui: unimplemented Window.newFocusListener")
}

// Focus moves focus to the given handle (M0-window-core ports it).
func (w *Window) Focus(handle FocusHandle, cx *App) {
	panic("gpui: unimplemented Window.Focus")
}

// OnAction registers an action listener for the next frame (M0-window-dispatch ports it).
func (w *Window) OnAction(actionType reflect.Type, listener func(action any, phase DispatchPhase, window *Window, cx *App)) {
	panic("gpui: unimplemented Window.OnAction")
}

// Spawn spawns f on the main thread with an AsyncWindowContext for this window (M0-window-core ports it).
func (w *Window) Spawn[R any](cx *App, f func(cx *AsyncWindowContext) (R, error)) *Task[R] {
	panic("gpui: unimplemented Window.Spawn")
}

// SpawnWithPriority spawns f on the main thread with the given priority (M0-window-core ports it).
func (w *Window) SpawnWithPriority[R any](priority Priority, cx *App, f func(cx *AsyncWindowContext) (R, error)) *Task[R] {
	panic("gpui: unimplemented Window.SpawnWithPriority")
}

// Prompt presents a platform dialog and returns the index of the clicked button (M0-window-core ports it).
func (w *Window) Prompt(level PromptLevel, message string, detail Option[string], answers []PromptButton, cx *App) *scheduler.OneshotReceiver[int] {
	panic("gpui: unimplemented Window.Prompt")
}

// HasPendingKeystrokes reports whether a multi-stroke binding is pending (M0-window-dispatch ports it).
func (w *Window) HasPendingKeystrokes() bool {
	panic("gpui: unimplemented Window.HasPendingKeystrokes")
}

// FocusID identifies a focusable element of a window (M0-window-core ports it).
type FocusID uint64

// SvgRenderer renders SVGs into images (M0-elements-media ports it).
type SvgRenderer struct {
	assetSource AssetSource
}

// NewSvgRenderer creates an SVG renderer loading referenced assets from assetSource (M0-elements-media ports it).
// The placeholder body only stores the source, so App construction works before M0-elements-media.
func NewSvgRenderer(assetSource AssetSource) SvgRenderer {
	return SvgRenderer{assetSource: assetSource}
}

// RenderSingleFrame renders an SVG into a single-frame image (M0-elements-media ports it).
func (r SvgRenderer) RenderSingleFrame(bytes []byte, scaleFactor float32) (*RenderImage, error) {
	panic("gpui: unimplemented SvgRenderer.RenderSingleFrame")
}

// ImageSource is a source of image content (M0-elements-media ports it).
type ImageSource interface {
	isImageSource()
}

// ImageSourceImage is cached image data (M0-elements-media ports it).
type ImageSourceImage struct {
	// Image is the image.
	Image *Image
}

func (ImageSourceImage) isImageSource() {}

// AnyImageCache is a type-erased image cache (M0-elements-media ports it).
type AnyImageCache struct{}

func imageSourceUseData(source ImageSource, cache Option[AnyImageCache], window *Window, cx *App) (*RenderImage, bool, error) {
	panic("gpui: unimplemented imageSourceUseData")
}

func imageSourceGetData(source ImageSource, cache Option[AnyImageCache], window *Window, cx *App) (*RenderImage, bool, error) {
	panic("gpui: unimplemented imageSourceGetData")
}

func imageSourceRemoveAsset(source ImageSource, cx *App) {
	panic("gpui: unimplemented imageSourceRemoveAsset")
}

func imageSourceIsAssetCached(source ImageSource, cx *App) bool {
	panic("gpui: unimplemented imageSourceIsAssetCached")
}

// RemSize returns the window's rem size (M0-window-paint ports it).
func (w *Window) RemSize() Pixels {
	panic("gpui: unimplemented Window.RemSize")
}

// Draw renders the window's root views into the next frame (M0-window-paint ports it).
func (w *Window) Draw(cx *App) ArenaClearNeeded {
	panic("gpui: unimplemented Window.Draw")
}

// RequestLayout adds a node to the layout tree for the current frame (M0-window-dispatch ports it).
func (w *Window) RequestLayout(style Style, children []LayoutID, cx *App) LayoutID {
	panic("gpui: unimplemented Window.RequestLayout")
}

// WithElementState gives f the element state of type S stored under id (M0-window-paint ports it).
func (w *Window) WithElementState[S, R any](id *GlobalElementID, f func(state Option[S], window *Window) (R, S)) R {
	panic("gpui: unimplemented Window.WithElementState")
}

// PaintQuad paints a quad into the scene for the next frame (M0-window-paint ports it).
func (w *Window) PaintQuad(quad PaintQuad) {
	panic("gpui: unimplemented Window.PaintQuad")
}

// InsertHitbox registers a hitbox for the current frame (M0-window-dispatch ports it).
func (w *Window) InsertHitbox(bounds Bounds[Pixels], behavior HitboxBehavior) Hitbox {
	panic("gpui: unimplemented Window.InsertHitbox")
}

// DispatchKeystroke dispatches a keystroke as if it had been typed (M0-window-dispatch ports it).
func (w *Window) DispatchKeystroke(keystroke Keystroke, cx *App) bool {
	panic("gpui: unimplemented Window.DispatchKeystroke")
}

// DispatchEvent dispatches a platform input event (M0-window-dispatch ports it).
func (w *Window) DispatchEvent(event PlatformInput, cx *App) DispatchEventResult {
	panic("gpui: unimplemented Window.DispatchEvent")
}

// Refresh schedules a redraw of the window (M0-window-paint ports it).
func (w *Window) Refresh() {
	panic("gpui: unimplemented Window.Refresh")
}

// OnNextFrame schedules f to run before the next frame (M0-window-paint ports it).
func (w *Window) OnNextFrame(f func(window *Window, cx *App)) {
	panic("gpui: unimplemented Window.OnNextFrame")
}

// RequestAnimationFrame schedules a redraw for the next frame (M0-window-paint ports it).
func (w *Window) RequestAnimationFrame() {
	panic("gpui: unimplemented Window.RequestAnimationFrame")
}

// ReplaceRoot replaces the window's root view; *V must implement Render[V] (M0-window-paint ports it).
func (w *Window) ReplaceRoot[V any](cx *App, buildView func(window *Window, cx *Context[V]) *V) Entity[V] {
	panic("gpui: unimplemented Window.ReplaceRoot")
}

// ArenaClearNeeded is returned by Window.Draw (M0-window-core ports it).
type ArenaClearNeeded struct{}

// DispatchPhase is the phase of event dispatch (M0-window-core ports it).
type DispatchPhase uint8

const (
	// DispatchPhaseBubble dispatches from the innermost element outwards.
	DispatchPhaseBubble DispatchPhase = iota
	// DispatchPhaseCapture dispatches from the window inwards.
	DispatchPhaseCapture
)

// FocusHandle is a ref-counted handle to a focusable element (M0-window-core ports it).
type FocusHandle struct {
	id FocusID
	// TabIndex is the element's position in the tab order.
	TabIndex int
	// TabStop reports whether the element is reachable with tab navigation.
	TabStop bool
}

// Focus moves focus to the element (M0-window-core ports it).
func (h FocusHandle) Focus(window *Window, cx *App) {
	panic("gpui: unimplemented FocusHandle.Focus")
}

// Clone returns a new counted handle (M0-window-core ports it).
func (h FocusHandle) Clone() FocusHandle {
	panic("gpui: unimplemented FocusHandle.Clone")
}

// Release drops this handle (M0-window-core ports it).
func (h FocusHandle) Release() {
	panic("gpui: unimplemented FocusHandle.Release")
}

// Focusable is implemented by views that have a focus handle (M0-window-core ports it).
type Focusable interface {
	FocusHandle(cx *App) FocusHandle
}

// ContentMask restricts painting to the given bounds (M0-window-core ports it).
type ContentMask[T Number] struct {
	// Bounds is the area painting is clipped to.
	Bounds Bounds[T]
}

// ElementID is the identifier of an element among its siblings (M0-window-core ports it).
type ElementID struct {
	kind uint8
	name SharedString
	num  uint64
}

// ElementIDSource is the set of types an ElementID can be built from (M0-window-core ports it).
type ElementIDSource interface {
	~string | ~int | ~uint64 | ElementID
}

// GlobalElementID is the path of element ids from the window root (M0-element ports it).
type GlobalElementID struct {
	path []ElementID
}

// InspectorElementID identifies an element in the inspector (M0-element ports it).
type InspectorElementID struct{}

// SourceLocation is a source position recorded when an element is constructed (M0-element ports it).
type SourceLocation struct {
	// File is the source file name.
	File string
	// Line is the 1-based line number.
	Line uint32
	// Column is the 1-based column number.
	Column uint32
}

// Hitbox is a region that receives mouse events (M0-window-core ports it).
type Hitbox struct{}

// HitboxBehavior configures how a hitbox affects hit testing (M0-window-core ports it).
type HitboxBehavior uint8

// PaintQuad describes a quad to paint (M0-window-core ports it).
type PaintQuad struct{}

// DispatchEventResult reports what happened during event dispatch (M0-window-dispatch ports it).
type DispatchEventResult struct {
	// Propagate reports whether the event should continue to the platform.
	Propagate bool
	// DefaultPrevented reports whether a handler prevented the default action.
	DefaultPrevented bool
}

// A11ySubtreeBuilder adds synthetic accessibility nodes below an element (M0-window-a11y ports it).
type A11ySubtreeBuilder struct {
	parentID a11y.NodeID
}

// Element is implemented by types that participate in laying out and painting the contents of a window (M0-element ports it).
type Element[RL, PP any] interface {
	ElementID() (ElementID, bool)
	SourceLocation() *SourceLocation
	RequestLayout(id *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, RL)
	Prepaint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *RL, window *Window, cx *App) PP
	Paint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *RL, prepaint *PP, window *Window, cx *App)
}

// ElementA11y is the optional accessibility hook of an Element (M0-element ports it).
type ElementA11y interface {
	A11yRole() (a11y.Role, bool)
	WriteA11yInfo(node *a11y.Node)
}

// ElementA11ySyntheticChildren is the optional hook that adds synthetic accessibility children (M0-element ports it).
type ElementA11ySyntheticChildren[PP any] interface {
	ElementA11ySyntheticChildren(prepaint *PP, builder *A11ySubtreeBuilder)
}

// IntoElement is implemented by any type that can be converted into an element (M0-element ports it).
type IntoElement interface {
	IntoAnyElement() AnyElement
}

// AnyElement is a type-erased element (M0-element ports it).
type AnyElement struct {
	element any
}

// NewAnyElement wraps a typed element (M0-element ports it).
func NewAnyElement[RL, PP any](element Element[RL, PP]) AnyElement {
	panic("gpui: unimplemented NewAnyElement")
}

// IntoAnyElement returns the element itself.
func (e AnyElement) IntoAnyElement() AnyElement {
	return e
}

// RequestLayout requests layout for the element (M0-element ports it).
func (e *AnyElement) RequestLayout(window *Window, cx *App) LayoutID {
	panic("gpui: unimplemented AnyElement.RequestLayout")
}

// LayoutAsRoot lays the element out as the root of a layout tree (M0-element ports it).
func (e *AnyElement) LayoutAsRoot(availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels] {
	panic("gpui: unimplemented AnyElement.LayoutAsRoot")
}

// PrepaintAt prepaints the element at the given origin (M0-element ports it).
func (e *AnyElement) PrepaintAt(origin Point[Pixels], window *Window, cx *App) {
	panic("gpui: unimplemented AnyElement.PrepaintAt")
}

// Paint paints the element (M0-element ports it).
func (e *AnyElement) Paint(window *Window, cx *App) {
	panic("gpui: unimplemented AnyElement.Paint")
}

// LayoutID identifies a node in the layout tree (M0-element ports it).
type LayoutID struct {
	node uint64
}

// AvailableSpace is the space available to an element during layout (M0-element ports it).
type AvailableSpace struct {
	kind     uint8
	definite Pixels
}

// Style is the computed style of an element (M0-style ports it).
type Style struct{}

// StyleRefinement is a partial Style (M0-style ports it).
type StyleRefinement struct{}

// Styler is implemented by elements that embed Styled (M0-style ports it).
type Styler interface {
	Style() *StyleRefinement
}

// Styled is the fluent styling mixin; bind it with BindStyled (M0-style ports it).
type Styled[S Styler] struct {
	self S
}

// BindStyled binds a Styled mixin to the element that embeds it.
func BindStyled[S Styler](m *Styled[S], self S) {
	m.self = self
}

// Flex sets the element to display flex (M0-style ports it).
func (s *Styled[S]) Flex() S {
	panic("gpui: unimplemented Styled.Flex")
}

// SizeFull sets the element's width and height to 100% (M0-style ports it).
func (s *Styled[S]) SizeFull() S {
	panic("gpui: unimplemented Styled.SizeFull")
}

// Interactivity holds the listeners and interactive style of an element (M0-div-api ports it).
type Interactivity struct{}

// InteractiveElement is implemented by elements that embed Interactive (M0-div-api ports it).
type InteractiveElement interface {
	Interactivity() *Interactivity
}

// Interactive is the InteractiveElement mixin; bind it with BindInteractive (M0-div-api ports it).
type Interactive[S InteractiveElement] struct {
	self S
}

// BindInteractive binds an Interactive mixin to the element that embeds it.
func BindInteractive[S InteractiveElement](m *Interactive[S], self S) {
	m.self = self
}

// OnClick registers a click listener (M0-div-api ports it).
func (i *Interactive[S]) OnClick(listener func(event *ClickEvent, window *Window, cx *App)) S {
	panic("gpui: unimplemented Interactive.OnClick")
}

// InteractiveID carries InteractiveElement's ID method, which cannot live on Interactive (D-IDMIXIN); bind it with BindInteractiveID (M0-div-api ports it).
type InteractiveID[S InteractiveElement] struct {
	self S
}

// BindInteractiveID binds an InteractiveID mixin to the element that embeds it.
func BindInteractiveID[S InteractiveElement](m *InteractiveID[S], self S) {
	m.self = self
}

// ID assigns the element an id, making it stateful (M0-div-api ports it).
func (i *InteractiveID[S]) ID[I ElementIDSource](id I) *Stateful[S] {
	panic("gpui: unimplemented InteractiveID.ID")
}

// StatefulInteractiveElement is implemented by stateful interactive elements (M0-div-api ports it).
type StatefulInteractiveElement interface {
	InteractiveElement
}

// StatefulInteractive is the StatefulInteractiveElement mixin; bind it with BindStatefulInteractive (M0-div-api ports it).
type StatefulInteractive[S StatefulInteractiveElement] struct {
	self S
}

// BindStatefulInteractive binds a StatefulInteractive mixin to the element that embeds it.
func BindStatefulInteractive[S StatefulInteractiveElement](m *StatefulInteractive[S], self S) {
	m.self = self
}

// ParentElement is implemented by elements that embed Parent (M0-element ports it).
type ParentElement interface {
	Extend(elements []AnyElement)
}

// Parent is the ParentElement mixin; bind it with BindParent (M0-element ports it).
type Parent[S ParentElement] struct {
	self S
}

// BindParent binds a Parent mixin to the element that embeds it.
func BindParent[S ParentElement](m *Parent[S], self S) {
	m.self = self
}

// Child adds a child element (M0-element ports it).
func (p *Parent[S]) Child(child IntoElement) S {
	panic("gpui: unimplemented Parent.Child")
}

// Children adds child elements (M0-element ports it).
func (p *Parent[S]) Children(children ...IntoElement) S {
	panic("gpui: unimplemented Parent.Children")
}

// ChildText adds a text child (M0-element ports it).
func (p *Parent[S]) ChildText(text string) S {
	panic("gpui: unimplemented Parent.ChildText")
}

// Div is the basic container element (M0-div-api ports it).
type Div struct {
	Styled[*Div]
	Interactive[*Div]
	InteractiveID[*Div]
	Parent[*Div]
	Fluent[*Div]
	interactivity Interactivity
	children      []AnyElement
}

// NewDiv constructs a Div (M0-div-api ports it).
func NewDiv() *Div {
	panic("gpui: unimplemented NewDiv")
}

// Style returns the element's style refinement (M0-div-api ports it).
func (d *Div) Style() *StyleRefinement {
	panic("gpui: unimplemented Div.Style")
}

// Interactivity returns the element's interactivity (M0-div-api ports it).
func (d *Div) Interactivity() *Interactivity {
	panic("gpui: unimplemented Div.Interactivity")
}

// Extend adds child elements (M0-div-api ports it).
func (d *Div) Extend(elements []AnyElement) {
	panic("gpui: unimplemented Div.Extend")
}

// IntoAnyElement converts the Div into an AnyElement (M0-div-paint ports it).
func (d *Div) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Div.IntoAnyElement")
}

// Stateful wraps an element that has an id; it is IntoElement only and delegates to its element (D-COND, M0-div-paint ports it).
type Stateful[E any] struct {
	Styled[*Stateful[E]]
	Interactive[*Stateful[E]]
	StatefulInteractive[*Stateful[E]]
	Parent[*Stateful[E]]
	Fluent[*Stateful[E]]
	element E
}

// Style returns the inner element's style refinement; E must implement Styler (M0-div-paint ports it).
func (s *Stateful[E]) Style() *StyleRefinement {
	panic("gpui: unimplemented Stateful.Style")
}

// Interactivity returns the inner element's interactivity; E must implement InteractiveElement (M0-div-paint ports it).
func (s *Stateful[E]) Interactivity() *Interactivity {
	panic("gpui: unimplemented Stateful.Interactivity")
}

// Extend adds children to the inner element; E must implement ParentElement (M0-div-paint ports it).
func (s *Stateful[E]) Extend(elements []AnyElement) {
	panic("gpui: unimplemented Stateful.Extend")
}

// IntoAnyElement returns the inner element's AnyElement (M0-div-paint ports it).
func (s *Stateful[E]) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Stateful.IntoAnyElement")
}

// HitboxID identifies a hitbox within a frame (M0-window-core ports it).
type HitboxID uint64

// TimedTransformation is a transformation animated over time (M0-elements-media ports it).
type TimedTransformation struct{}

func (t TimedTransformation) minFrameInterval() (time.Duration, bool) {
	panic("gpui: unimplemented TimedTransformation.minFrameInterval")
}

func (t TimedTransformation) at(elapsed time.Duration) Transformation {
	panic("gpui: unimplemented TimedTransformation.at")
}

// Transformation is a scale, translation, rotation and alpha applied to an SVG (M0-elements-media ports it).
type Transformation struct{}

func (t Transformation) alphaFactor() float32 {
	panic("gpui: unimplemented Transformation.alphaFactor")
}

func (t Transformation) intoMatrix(center Point[Pixels], scaleFactor float32) TransformationMatrix {
	panic("gpui: unimplemented Transformation.intoMatrix")
}

// ShapedLine is a line of text that has been shaped and decorated; it embeds its layout
// (upstream's Deref) (M0-text-elements ports it).
type ShapedLine struct {
	*LineLayout
	// Text is the text that was shaped for this line.
	Text           SharedString
	decorationRuns []DecorationRun
}

// WrappedLine is a line of text that has been shaped, decorated and wrapped; it embeds its
// layout (upstream's Deref) (M0-text-elements ports it).
type WrappedLine struct {
	*WrappedLineLayout
	// Text is the text that was shaped for this line.
	Text           SharedString
	decorationRuns []DecorationRun
}

// DecorationRun sets the text decoration for a run of text (M0-text-elements ports it).
type DecorationRun struct {
	// Len is the length of the run in UTF-8 bytes.
	Len uint32
	// Color is the color of the run.
	Color Hsla
	// BackgroundColor is the background color of the run.
	BackgroundColor Option[Hsla]
	// Underline is the underline style of the run.
	Underline Option[UnderlineStyle]
	// Strikethrough is the strikethrough style of the run.
	Strikethrough Option[StrikethroughStyle]
}

// UnderlineStyle is the properties of an underline (M0-style ports it).
type UnderlineStyle struct {
	// Thickness is the thickness of the underline.
	Thickness Pixels
	// Color is the color of the underline.
	Color Option[Hsla]
	// Wavy reports whether the underline is wavy, like in a spell checker.
	Wavy bool
}

// StrikethroughStyle is the properties of a strikethrough (M0-style ports it).
type StrikethroughStyle struct {
	// Thickness is the thickness of the strikethrough.
	Thickness Pixels
	// Color is the color of the strikethrough.
	Color Option[Hsla]
}

// TestAppContext is a test context with a TestPlatform (M0-test-contexts ports it).
type TestAppContext struct{}

// TestAppContextBuild creates a new TestAppContext on the dispatcher (M0-test-contexts ports it).
func TestAppContextBuild(dispatcher *TestDispatcher, fnName Option[string]) *TestAppContext {
	panic("gpui: unimplemented TestAppContextBuild")
}

// Update runs f with the App and flushes its effects (M0-test-contexts ports it).
func (cx *TestAppContext) Update(f func(cx *App)) {
	panic("gpui: unimplemented TestAppContext.Update")
}

// RunUntilParked runs the scheduler until no task can make progress (M0-test-contexts ports it).
func (cx *TestAppContext) RunUntilParked() {
	panic("gpui: unimplemented TestAppContext.RunUntilParked")
}

func (cx *TestAppContext) drop() {
	panic("gpui: unimplemented TestAppContext.drop")
}

// IntoAnyElement converts the string into a text element (M0-text-elements ports it).
func (s SharedString) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented SharedString.IntoAnyElement")
}

// Appearance returns the window's appearance (M0-window-paint ports it).
func (w *Window) Appearance() WindowAppearance {
	panic("gpui: unimplemented Window.Appearance")
}

// Empty is an element that renders nothing (M0-element ports it).
type Empty struct{}

// IntoAnyElement converts the element into an AnyElement (M0-element ports it).
func (Empty) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Empty.IntoAnyElement")
}

// LongPressEvent is a long-press gesture recognized from touch input (M0-gestures-spring ports it).
type LongPressEvent struct{}

// TouchDragEvent is a direct touch drag claimed by an element (M0-gestures-spring ports it).
type TouchDragEvent struct{}

// BackGestureEvent is a back-navigation gesture recognized by the platform (M0-gestures-spring ports it).
type BackGestureEvent struct{}

// HangDetector detects hangs from the foreground journal (M3-profiler-hang ports it).
type HangDetector struct{}

// TakeActiveTime returns the time the user was active since the previous call (M3-profiler-hang ports it).
func (d *HangDetector) TakeActiveTime(now scheduler.Instant, maxGap time.Duration) time.Duration {
	panic("gpui: unimplemented HangDetector.TakeActiveTime")
}

// FirstPresentAt returns when the first frame was presented (M3-profiler-hang ports it).
func (d *HangDetector) FirstPresentAt() (scheduler.Instant, bool) {
	panic("gpui: unimplemented HangDetector.FirstPresentAt")
}

// Poll returns the incidents sealed since the previous poll (M3-profiler-hang ports it).
func (d *HangDetector) Poll() []HangIncident {
	panic("gpui: unimplemented HangDetector.Poll")
}

// HangIncident is a detected hang (M3-profiler-hang ports it).
type HangIncident struct{}

// InputTiming is one input dispatch (M3-profiler-journal ports it).
type InputTiming struct {
	Kind               string
	Start              scheduler.Instant
	End                scheduler.Instant
	CausedInvalidation bool
}

func beginForegroundTurn() {}

func endForegroundTurn() {}

func recordTaskPoll(timing TaskTiming) {}

func recordFramePending(windowID WindowID, dirtyAt scheduler.Instant) {}

func recordWindowClosed(windowID WindowID) {}

func powerInterruptedSince(start scheduler.Instant) bool {
	return false
}

func frameSampleIsValid(id WindowID, start scheduler.Instant) bool {
	return true
}

func recordInput(timing InputTiming) {}

func recordAction(timing ActionTiming) {}

func recordView(timing ViewTiming) {}

func recordDraw(timing FrameTiming) {}

func recordPresent(timing PresentTiming, frame Option[FrameTiming]) {}

func newWindow(handle AnyWindowHandle, options WindowOptions, cx *App) (*Window, error) {
	panic("gpui: unimplemented newWindow")
}

// NewWindowHandle creates a handle from a window id without checking the root type (M0-window-core ports it).
func NewWindowHandle[V any](id WindowID) WindowHandle[V] {
	panic("gpui: unimplemented NewWindowHandle")
}

// AsU64 converts the window id to a uint64 (M0-window-core ports it).
func (id WindowID) AsU64() uint64 {
	panic("gpui: unimplemented WindowID.AsU64")
}

// WindowID returns the id of the window (M0-window-core ports it).
func (h AnyWindowHandle) WindowID() WindowID {
	panic("gpui: unimplemented AnyWindowHandle.WindowID")
}

// Update updates the root view of the window; it fails when the window has been closed (M0-window-core ports it).
func (h AnyWindowHandle) Update(cx AppContext, update func(root AnyView, window *Window, cx *App)) error {
	panic("gpui: unimplemented AnyWindowHandle.Update")
}

// Clear clears the element arena the draw ran against (M0-window-core ports it).
func (a ArenaClearNeeded) Clear(cx *App) {
	panic("gpui: unimplemented ArenaClearNeeded.Clear")
}

type windowInvalidator struct{}

func (w windowInvalidator) invalidateView(entity EntityID, cx *App) bool {
	panic("gpui: unimplemented windowInvalidator.invalidateView")
}

func (w windowInvalidator) isDirty() bool {
	panic("gpui: unimplemented windowInvalidator.isDirty")
}

func (w windowInvalidator) setDirty(dirty bool) {
	panic("gpui: unimplemented windowInvalidator.setDirty")
}

type focusMap struct {
	mu             sync.RWMutex
	droppedHandles []FocusID
}

func (m *focusMap) removeHandle(id FocusID) {
	panic("gpui: unimplemented focusMap.removeHandle")
}

func newFocusHandle(handles *focusMap) FocusHandle {
	panic("gpui: unimplemented newFocusHandle")
}

// FrameCallback is a callback run before the next frame of a window (M0-window-core ports it).
type FrameCallback func(window *Window, cx *App)

// WindowHandle returns a handle to the window (M0-window-core ports it).
func (w *Window) WindowHandle() AnyWindowHandle {
	panic("gpui: unimplemented Window.WindowHandle")
}

// Blur removes focus from the window's focused element (M0-window-paint ports it).
func (w *Window) Blur(cx *App) {
	panic("gpui: unimplemented Window.Blur")
}

// DispatchAction dispatches an action to the focused element of the window (M0-window-paint ports it).
func (w *Window) DispatchAction(action Action, cx *App) {
	panic("gpui: unimplemented Window.DispatchAction")
}

// IsActionAvailable reports whether the action has a handler in the window's focus path (M0-window-paint ports it).
func (w *Window) IsActionAvailable(action Action, cx *App) bool {
	panic("gpui: unimplemented Window.IsActionAvailable")
}

// DropImage removes an image from the window's sprite atlas (M0-window-paint ports it).
func (w *Window) DropImage(data *RenderImage) error {
	panic("gpui: unimplemented Window.DropImage")
}

func (w *Window) clearPendingKeystrokes(cx *App) {
	panic("gpui: unimplemented Window.clearPendingKeystrokes")
}

// KeyBinding is a keybinding and its associated metadata (M0-keymap ports it).
type KeyBinding struct{}

// KeyContext is a set of identifiers and key-value pairs that keymap contexts match against (M0-keymap ports it).
type KeyContext struct{}

// AddBindings adds bindings to the keymap (M0-keymap ports it).
func (k *Keymap) AddBindings(bindings []KeyBinding) {
	panic("gpui: unimplemented Keymap.AddBindings")
}

// Clear removes every binding from the keymap (M0-keymap ports it).
func (k *Keymap) Clear() {
	panic("gpui: unimplemented Keymap.Clear")
}

// AllBindingsForInput returns the bindings that match input, ignoring context (M0-keymap ports it).
func (k *Keymap) AllBindingsForInput(input []Keystroke) []KeyBinding {
	panic("gpui: unimplemented Keymap.AllBindingsForInput")
}

// PromptHandle is a handle to a prompt that the prompt renderer resolves (M0-window-dispatch ports it).
type PromptHandle struct{}

// RenderablePromptHandle is a prompt rendered as a view (M0-window-dispatch ports it).
type RenderablePromptHandle struct{}

func (r *inspectorElementRegistry) register[T any, R IntoElement](factory func(window *Window, cx *App) func(id InspectorElementID, state *T, window *Window, cx *App) R) {
	panic("gpui: unimplemented inspectorElementRegistry.register")
}

// NewHangDetector creates a detector over the foreground journal (M3-profiler-hang ports it).
func NewHangDetector(journal ForegroundJournal, threshold time.Duration, frameBudget time.Duration) *HangDetector {
	panic("gpui: unimplemented NewHangDetector")
}

func defaultFocusMap() *focusMap {
	return &focusMap{}
}

// Defer schedules f to run at the end of the current effect cycle with this window (M0-window-paint ports it).
func (w *Window) Defer(cx *App, f func(window *Window, cx *App)) {
	panic("gpui: unimplemented Window.Defer")
}

func (w *Window) appearanceChanged(cx *App) {
	panic("gpui: unimplemented Window.appearanceChanged")
}

func (w *Window) notifyDisplayObservers(cx *App) {
	panic("gpui: unimplemented Window.notifyDisplayObservers")
}

func (w *Window) debugState() jsontext.Value {
	panic("gpui: unimplemented Window.debugState")
}

// ActivateWindow brings the window to the front of the application (M0-window-dispatch ports it).
func (w *Window) ActivateWindow() {
	panic("gpui: unimplemented Window.ActivateWindow")
}

type promptBuilder interface {
	isPromptBuilder()
}

type promptBuilderDefault struct{}

func (promptBuilderDefault) isPromptBuilder() {}

// DefaultKeymap returns an empty keymap (M0-keymap ports it). The placeholder body returns a zero Keymap.
func DefaultKeymap() *Keymap {
	return &Keymap{}
}

// Inspector manages inspector state (M0-element ports it).
type Inspector struct{}

// InspectorRenderer renders the inspector UI; it is set on the App (M0-element ports it).
type InspectorRenderer func(inspector *Inspector, window *Window, cx *Context[Inspector]) AnyElement

type inspectorElementRegistry struct{}

func defaultInspectorElementRegistry() inspectorElementRegistry {
	return inspectorElementRegistry{}
}

func anyViewFromEntity[V any](entity Entity[V]) AnyView {
	panic("gpui: unimplemented anyViewFromEntity")
}

// ForegroundJournal is the foreground work journal of the app's thread (M3-profiler-journal ports it).
type ForegroundJournal struct{}

func installForegroundJournal() ForegroundJournal {
	return ForegroundJournal{}
}

func observePower(cx *App) {}

type promptBuilderCustom struct {
	renderer func(level PromptLevel, message string, detail Option[string], actions []PromptButton, handle PromptHandle, window *Window, cx *App) RenderablePromptHandle
}

func (promptBuilderCustom) isPromptBuilder() {}
