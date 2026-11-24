package gpui

import (
	"slices"
	"sync/atomic"
	"time"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/scheduler"
)

// RenderSvgParams identifies a rasterized SVG in the sprite atlas (M0-elements-media ports it).
type RenderSvgParams struct {
	// Path is the SVG's asset path.
	Path SharedString
	// Size is the rasterized size.
	Size Size[DevicePixels]
}

// NativeViewID identifies a native view within its window (M0-window-a11y ports it).
type NativeViewID uint64

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

// A11ySubtreeBuilder adds synthetic accessibility nodes below an element (M0-window-a11y ports it).
type A11ySubtreeBuilder struct {
	parentID a11y.NodeID
	nodes    *a11yNodeBuilder
	creator  nodeCreator
}

func newA11ySubtreeBuilder(parentID a11y.NodeID, nodes *a11yNodeBuilder) A11ySubtreeBuilder {
	panic("gpui: unimplemented newA11ySubtreeBuilder")
}

func (b A11ySubtreeBuilder) withCreator(creator nodeCreator) A11ySubtreeBuilder {
	panic("gpui: unimplemented A11ySubtreeBuilder.withCreator")
}

type windowA11y struct {
	forceDisabled   bool
	activeThisFrame bool
	windowTitle     Option[SharedString]
	nodes           a11yNodeBuilder
	focusIDs        map[a11y.NodeID]FocusID
	nodeBounds      map[a11y.NodeID]Bounds[Pixels]
	actionListeners map[a11y.NodeID][]a11yActionEntry
	viewTypeNames   map[EntityID]string
}

type a11yActionListener func(data a11y.ActionData, window *Window, cx *App)

type a11yActionEntry struct {
	action   a11y.Action
	listener a11yActionListener
}

func newWindowA11y(activeFlag *atomic.Bool, forceDisabled bool, windowTitle Option[SharedString]) windowA11y {
	return windowA11y{
		forceDisabled:   forceDisabled,
		focusIDs:        make(map[a11y.NodeID]FocusID),
		nodeBounds:      make(map[a11y.NodeID]Bounds[Pixels]),
		actionListeners: make(map[a11y.NodeID][]a11yActionEntry),
		viewTypeNames:   make(map[EntityID]string),
	}
}

func (a *windowA11y) isEnabled() bool {
	return !a.forceDisabled
}

func (a *windowA11y) debugTreeJSON() (string, bool) {
	panic("gpui: unimplemented windowA11y.debugTreeJSON")
}

func (a *windowA11y) isActive() bool {
	return a.activeThisFrame
}

func (a *windowA11y) setWindowTitle(title SharedString) {
	a.windowTitle = Some(title)
}

func (a *windowA11y) syncActiveFlag() {}

func (a *windowA11y) beginFrame() {
	panic("gpui: unimplemented windowA11y.beginFrame")
}

func (a *windowA11y) endFrame(frame frameDebugInfo) a11y.TreeUpdate {
	panic("gpui: unimplemented windowA11y.endFrame")
}

type frameDebugInfo struct {
	viewportSize Size[Pixels]
	scaleFactor  float32
	tabStopCount int
}

func (w *Window) syncNativeViewFocus() {
	w.nativeViews = slices.DeleteFunc(w.nativeViews, func(view weak.Pointer[nativeViewState]) bool { return view.Value() == nil })
	if len(w.nativeViews) == 0 {
		return
	}
	panic("gpui: unimplemented Window.syncNativeViewFocus")
}

func (w *Window) endCompositionBeforeFocusingNativeView(focus FocusID, cx *App) {
	if len(w.nativeViews) == 0 {
		return
	}
	panic("gpui: unimplemented Window.endCompositionBeforeFocusingNativeView")
}

// SmoothSvgScaleFactor is the factor SVGs are rasterized at above their device size (M0-elements-media ports svg_renderer.rs).
const SmoothSvgScaleFactor float32 = 2.

func (r SvgRenderer) renderAlphaMask(params RenderSvgParams, bytes []byte) (Size[DevicePixels], []byte, bool, error) {
	panic("gpui: unimplemented SvgRenderer.renderAlphaMask")
}

type a11yNodeBuilder struct{}

func (b *a11yNodeBuilder) push(id a11y.NodeID, node a11y.Node) bool {
	panic("gpui: unimplemented a11yNodeBuilder.push")
}

func (b *a11yNodeBuilder) pop() {
	panic("gpui: unimplemented a11yNodeBuilder.pop")
}

func (b *a11yNodeBuilder) recordNodeInfo(id a11y.NodeID, info nodeDebugInfo) {
	panic("gpui: unimplemented a11yNodeBuilder.recordNodeInfo")
}

type nodeDebugInfo struct {
	synthetic      bool
	view           Option[string]
	elementID      Option[string]
	sourceLocation *SourceLocation
}

type nodeCreator struct {
	view           Option[string]
	elementID      Option[string]
	sourceLocation *SourceLocation
}

// ScrollHandle tracks the scroll state of an element (M0-div-paint ports it).
type ScrollHandle struct {
	state *struct{}
}

// ScrollAnchor scrolls a parent so a child becomes visible (M0-div-paint ports it).
type ScrollAnchor struct {
	handle ScrollHandle
}

// ImageCacheProvider creates an image cache during request layout (M0-elements-media ports it).
type ImageCacheProvider interface {
	Provide(window *Window, cx *App) AnyImageCache
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

// IntoAnyElement converts the string into a text element (M0-text-elements ports it).
func (s SharedString) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented SharedString.IntoAnyElement")
}

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

// NewHangDetector creates a detector over the foreground journal (M3-profiler-hang ports it).
func NewHangDetector(journal ForegroundJournal, threshold time.Duration, frameBudget time.Duration) *HangDetector {
	panic("gpui: unimplemented NewHangDetector")
}

func observePower(cx *App) {}

// NewVisualTestPlatform wraps platform with a TestDispatcher seeded with seed (M3-root-capture ports it).
func NewVisualTestPlatform(platform Platform, seed uint64) *VisualTestPlatform {
	panic("gpui: unimplemented NewVisualTestPlatform")
}

// Dispatcher returns the platform's test dispatcher (M3-root-capture ports it).
func (p *VisualTestPlatform) Dispatcher() *TestDispatcher {
	panic("gpui: unimplemented VisualTestPlatform.Dispatcher")
}

const rootNodeID a11y.NodeID = 0

type nativeViewState struct{}

func (w *Window) hitTestNativeView(position Point[Pixels]) (NativeViewID, bool) {
	panic("gpui: unimplemented Window.hitTestNativeView")
}

func (w *Window) dispatchKeyDownBeforeNativeView(event KeyDownEvent, cx *App) bool {
	panic("gpui: unimplemented Window.dispatchKeyDownBeforeNativeView")
}

func (w *Window) focusNativeView(id NativeViewID, cx *App) {
	panic("gpui: unimplemented Window.focusNativeView")
}

// DebugFrameOverlay draws frame timings over the window (M3-profiler-hang ports it).
type DebugFrameOverlay struct{}

// NewDebugFrameOverlay creates an overlay (M3-profiler-hang ports it). The placeholder body returns the zero value.
func NewDebugFrameOverlay() DebugFrameOverlay {
	return DebugFrameOverlay{}
}

// DebugFrameOverlayMode is what the debug frame overlay shows (M3-profiler-hang ports it).
type DebugFrameOverlayMode uint8

// Next returns the next mode in the cycle (M3-profiler-hang ports it).
func (m DebugFrameOverlayMode) Next() DebugFrameOverlayMode {
	panic("gpui: unimplemented DebugFrameOverlayMode.Next")
}

func (o *DebugFrameOverlay) mode() DebugFrameOverlayMode {
	return 0
}

func (o *DebugFrameOverlay) setMode(mode DebugFrameOverlayMode) {}

func (o *DebugFrameOverlay) resetStats() {}

func (o *DebugFrameOverlay) recordFrame(drawDuration time.Duration) {}

func (o *DebugFrameOverlay) paint(scene *Scene, viewportSize Size[Pixels], scaleFactor float32) {}

// PlatformSignal is a platform frame callback recorded in the foreground journal (M3-profiler-journal ports it).
type PlatformSignal struct {
	// WindowID is the window receiving the callback.
	WindowID WindowID
	// SignalAt is when the platform requested the callback, if its clock is supported.
	SignalAt Option[scheduler.Instant]
	// HandledAt is when foreground handling began.
	HandledAt scheduler.Instant
	// Source is whether the callback came from the platform or local scheduling.
	Source FrameRequestSource
}

// FrameSkipReason is why a frame request returned without drawing a new frame (M3-profiler-journal ports it).
type FrameSkipReason uint8

const (
	// FrameSkipReasonNoRenderNeeded means the callback completed without needing a new render.
	FrameSkipReasonNoRenderNeeded FrameSkipReason = iota
	// FrameSkipReasonInactiveFrameRateLimit means an inactive window deferred drawing.
	FrameSkipReasonInactiveFrameRateLimit
	// FrameSkipReasonThermalFrameRateLimit means thermal throttling deferred drawing.
	FrameSkipReasonThermalFrameRateLimit
	// FrameSkipReasonLowPowerFrameRateLimit means low power mode deferred drawing.
	FrameSkipReasonLowPowerFrameRateLimit
	// FrameSkipReasonAnimationFrameRateLimit means a scene animation's frame rate cap deferred presenting.
	FrameSkipReasonAnimationFrameRateLimit
)

func recordWindowVisibility(id WindowID, visibility WindowVisibility) {}

func recordPlatformSignal(signal PlatformSignal) {}

func recordReentrantFrameSkipped(windowID WindowID, at scheduler.Instant) {}

func recordFrameSkipped(windowID WindowID, at scheduler.Instant, reason FrameSkipReason) {}

// ForegroundJournal is the foreground work journal of the app's thread (M3-profiler-journal ports it).
type ForegroundJournal struct{}

func installForegroundJournal() ForegroundJournal {
	return ForegroundJournal{}
}

// VisualTestPlatform combines a real platform's rendering with a TestDispatcher (M3-root-capture ports it).
type VisualTestPlatform struct {
	Platform
}
