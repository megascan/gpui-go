package gpui

import (
	"errors"
	"image"
	"runtime"
	"sync"
	"weak"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

var testWindowFrameWakerSchedulesFrame bool

type testWindowState struct {
	mu                              sync.Mutex
	bounds                          Bounds[Pixels]
	handle                          AnyWindowHandle
	display                         PlatformDisplay
	title                           Option[string]
	edited                          bool
	documentPath                    Option[string]
	platform                        weak.Pointer[TestPlatform]
	spriteAtlas                     PlatformAtlas
	renderer                        PlatformHeadlessRenderer
	shouldCloseHandler              func() bool
	closeCallback                   func()
	hitTestWindowControlCallback    func() (WindowControlArea, bool)
	attachedNativeViews             *int
	focusedNativeView               Option[NativeViewID]
	hitTestNativeViewCallback       func(position Point[Pixels]) (NativeViewID, bool)
	nativeViewFocusedCallback       func(id NativeViewID)
	keyDownBeforeNativeViewCallback func(event KeyDownEvent) bool
	inputCallback                   func(input PlatformInput) DispatchEventResult
	activeStatusChangeCallback      func(active bool)
	visibility                      WindowVisibility
	visibilityCallback              func(visibility WindowVisibility)
	hoverStatusChangeCallback       func(hovered bool)
	resizeCallback                  func(size Size[Pixels], scaleFactor float32)
	visualViewport                  Option[Bounds[Pixels]]
	visualViewportCallback          func()
	insets                          WindowInsets
	insetsCallback                  func(insets WindowInsets)
	virtualKeyboardRequests         int
	virtualKeyboardDismissals       int
	movedCallback                   func()
	displayChangedCallback          func()
	appearanceChangeCallback        func()
	requestFrameCallback            func(options RequestFrameOptions)
	frameWakeCount                  *int
	frameScheduled                  bool
	frameCallbackPending            bool
	inputHandler                    *PlatformInputHandler
	textInputConfigurations         []TextInputConfiguration
	textInputStateChanges           []TextInputStateChange
	isFullscreen                    bool
	scaleFactor                     float32
	appearance                      WindowAppearance
	decorations                     Decorations
	externalDragFiles               []FileDragPath
	startExternalDragResult         bool
}

// TestWindow is the test platform's window. Copies made with Clone share one window.
type TestWindow struct {
	PlatformWindowDefaults
	state *testWindowState
}

// NewTestWindow creates a test window; renderer may be nil, in which case the window uses a HeadlessAtlas.
func NewTestWindow(handle AnyWindowHandle, params WindowParams, platform weak.Pointer[TestPlatform], display PlatformDisplay, renderer PlatformHeadlessRenderer) *TestWindow {
	var spriteAtlas PlatformAtlas
	if renderer != nil {
		spriteAtlas = renderer.SpriteAtlas()
	} else {
		spriteAtlas = &HeadlessAtlas{}
	}
	return newTestWindowFromState(&testWindowState{
		bounds:              params.Bounds,
		display:             display,
		platform:            platform,
		handle:              handle,
		spriteAtlas:         spriteAtlas,
		renderer:            renderer,
		visibility:          WindowVisibilityVisible,
		attachedNativeViews: new(int),
		frameWakeCount:      new(int),
		scaleFactor:         2.0,
		appearance:          WindowAppearanceLight,
	})
}

func newTestWindowFromState(state *testWindowState) *TestWindow {
	w := &TestWindow{state: state}
	BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)
	return w
}

// Clone returns another handle to the same window.
func (w *TestWindow) Clone() *TestWindow {
	return newTestWindowFromState(w.state)
}

func (w *TestWindow) lock() *testWindowState {
	w.state.mu.Lock()
	return w.state
}

func (w *TestWindow) unlock() {
	w.state.mu.Unlock()
}

// SimulateClose runs the close callback.
func (w *TestWindow) SimulateClose() {
	s := w.lock()
	callback := s.closeCallback
	s.closeCallback = nil
	w.unlock()
	if callback != nil {
		callback()
	}
}

// SimulateScheduledFrame delivers a scheduled frame and reports whether one was delivered.
func (w *TestWindow) SimulateScheduledFrame() bool {
	s := w.lock()
	if !s.frameScheduled {
		w.unlock()
		return false
	}
	s.frameScheduled = false
	s.frameCallbackPending = false
	callback := s.requestFrameCallback
	s.requestFrameCallback = nil
	if callback == nil {
		s.frameScheduled = true
		w.unlock()
		return false
	}
	w.unlock()

	callback(RequestFrameOptions{})
	s = w.lock()
	s.requestFrameCallback = callback
	w.unlock()
	return true
}

// FrameScheduled reports whether a frame is scheduled.
func (w *TestWindow) FrameScheduled() bool {
	defer w.unlock()
	return w.lock().frameScheduled
}

// SimulateVisibilityChange sets the visibility and reports it to the visibility callback.
func (w *TestWindow) SimulateVisibilityChange(visibility WindowVisibility) {
	s := w.lock()
	s.visibility = visibility
	callback := s.visibilityCallback
	s.visibilityCallback = nil
	w.unlock()
	if callback != nil {
		callback(visibility)
		w.lock().visibilityCallback = callback
		w.unlock()
	}
}

// SimulateMoveToDisplay moves the window to the display with the given ID.
func (w *TestWindow) SimulateMoveToDisplay(displayID DisplayID) {
	s := w.lock()
	s.display = TestDisplayWithID(displayID)
	callback := s.displayChangedCallback
	s.displayChangedCallback = nil
	w.unlock()
	if callback != nil {
		callback()
		w.lock().displayChangedCallback = callback
		w.unlock()
	}
}

// SimulateVisualViewportChange sets the visual viewport and reports the change.
func (w *TestWindow) SimulateVisualViewportChange(bounds Bounds[Pixels]) {
	s := w.lock()
	s.visualViewport = Some(bounds)
	callback := s.visualViewportCallback
	s.visualViewportCallback = nil
	w.unlock()
	if callback != nil {
		callback()
		w.lock().visualViewportCallback = callback
		w.unlock()
	}
}

// SimulateInsetsChange sets the insets and reports them to the insets callback.
func (w *TestWindow) SimulateInsetsChange(insets WindowInsets) {
	s := w.lock()
	s.insets = insets
	callback := s.insetsCallback
	s.insetsCallback = nil
	w.unlock()
	if callback != nil {
		callback(insets)
		w.lock().insetsCallback = callback
		w.unlock()
	}
}

// VirtualKeyboardRequests returns how many times the soft keyboard was shown.
func (w *TestWindow) VirtualKeyboardRequests() int {
	defer w.unlock()
	return w.lock().virtualKeyboardRequests
}

// VirtualKeyboardDismissals returns how many times the soft keyboard was hidden.
func (w *TestWindow) VirtualKeyboardDismissals() int {
	defer w.unlock()
	return w.lock().virtualKeyboardDismissals
}

// TextInputConfigurations returns every TextInputConfiguration forwarded to this window, in order.
func (w *TestWindow) TextInputConfigurations() []TextInputConfiguration {
	defer w.unlock()
	return append([]TextInputConfiguration(nil), w.lock().textInputConfigurations...)
}

// TextInputStateChanges returns every text input state change reported to this window, in order.
func (w *TestWindow) TextInputStateChanges() []TextInputStateChange {
	defer w.unlock()
	return append([]TextInputStateChange(nil), w.lock().textInputStateChanges...)
}

// SimulateResize sets the window size and reports it to the resize callback.
func (w *TestWindow) SimulateResize(size Size[Pixels]) {
	scaleFactor := w.ScaleFactor()
	s := w.lock()
	s.bounds.Size = size
	callback := s.resizeCallback
	s.resizeCallback = nil
	w.unlock()
	if callback == nil {
		return
	}
	callback(size, scaleFactor)
	w.lock().resizeCallback = callback
	w.unlock()
}

// SimulateScaleFactorChange simulates a display scale change through the resize callback,
// preserving logical bounds.
func (w *TestWindow) SimulateScaleFactorChange(scaleFactor float32) {
	s := w.lock()
	s.scaleFactor = scaleFactor
	size := s.bounds.Size
	w.unlock()
	w.SimulateResize(size)
}

// AttachedNativeViewCount returns how many native views are attached to the window.
func (w *TestWindow) AttachedNativeViewCount() int {
	defer w.unlock()
	return *w.lock().attachedNativeViews
}

// FocusedNativeView returns the native view the window last gave keyboard input to.
func (w *TestWindow) FocusedNativeView() (NativeViewID, bool) {
	defer w.unlock()
	return w.lock().focusedNativeView.Get()
}

// HitTestNativeView asks the window which native view a mouse event at position reaches.
func (w *TestWindow) HitTestNativeView(position Point[Pixels]) (NativeViewID, bool) {
	s := w.lock()
	callback := s.hitTestNativeViewCallback
	s.hitTestNativeViewCallback = nil
	w.unlock()
	if callback == nil {
		return 0, false
	}
	id, ok := callback(position)
	w.lock().hitTestNativeViewCallback = callback
	w.unlock()
	return id, ok
}

// SimulateKeyDownBeforeNativeView acts as if the user pressed a key while a native view had
// keyboard input, and returns whether the window used the key before the view saw it.
func (w *TestWindow) SimulateKeyDownBeforeNativeView(event KeyDownEvent) bool {
	s := w.lock()
	callback := s.keyDownBeforeNativeViewCallback
	s.keyDownBeforeNativeViewCallback = nil
	w.unlock()
	if callback == nil {
		return false
	}
	handled := callback(event)
	w.lock().keyDownBeforeNativeViewCallback = callback
	w.unlock()
	return handled
}

// SimulateNativeViewFocused acts as if the user moved keyboard input to a native view through the platform.
func (w *TestWindow) SimulateNativeViewFocused(id NativeViewID) {
	s := w.lock()
	s.focusedNativeView = Some(id)
	callback := s.nativeViewFocusedCallback
	s.nativeViewFocusedCallback = nil
	w.unlock()
	if callback == nil {
		return
	}
	callback(id)
	w.lock().nativeViewFocusedCallback = callback
	w.unlock()
}

func (w *TestWindow) simulateActiveStatusChange(active bool) {
	s := w.lock()
	callback := s.activeStatusChangeCallback
	s.activeStatusChangeCallback = nil
	w.unlock()
	if callback == nil {
		return
	}
	callback(active)
	w.lock().activeStatusChangeCallback = callback
	w.unlock()
}

// SimulateAppearanceChange sets the appearance and runs the appearance callback.
func (w *TestWindow) SimulateAppearanceChange(appearance WindowAppearance) {
	s := w.lock()
	s.appearance = appearance
	w.runAppearanceChangeCallback(s)
}

// SimulateDecorationsChange sets the decorations and runs the appearance callback.
func (w *TestWindow) SimulateDecorationsChange(decorations Decorations) {
	s := w.lock()
	s.decorations = decorations
	w.runAppearanceChangeCallback(s)
}

func (w *TestWindow) runAppearanceChangeCallback(s *testWindowState) {
	callback := s.appearanceChangeCallback
	s.appearanceChangeCallback = nil
	w.unlock()
	if callback == nil {
		return
	}
	callback()
	w.lock().appearanceChangeCallback = callback
	w.unlock()
}

// FrameWakeCount returns how many times this window's frame waker has been invoked.
func (w *TestWindow) FrameWakeCount() int {
	defer w.unlock()
	return *w.lock().frameWakeCount
}

// SimulateFrameRequest delivers a frame request to the window, as the platform's frame source would.
func (w *TestWindow) SimulateFrameRequest(options RequestFrameOptions) {
	s := w.lock()
	callback := s.requestFrameCallback
	s.requestFrameCallback = nil
	w.unlock()
	if callback == nil {
		return
	}
	callback(options)
	w.lock().requestFrameCallback = callback
	w.unlock()
}

// SimulateInput delivers the event to the input callback and reports whether it stopped propagating.
func (w *TestWindow) SimulateInput(event PlatformInput) bool {
	s := w.lock()
	callback := s.inputCallback
	s.inputCallback = nil
	w.unlock()
	if callback == nil {
		return false
	}
	result := callback(event)
	w.lock().inputCallback = callback
	w.unlock()
	return !result.Propagate
}

// ExternalDragFiles returns the paths of every external drag started from the window.
func (w *TestWindow) ExternalDragFiles() []FileDragPath {
	defer w.unlock()
	return append([]FileDragPath(nil), w.lock().externalDragFiles...)
}

// SetStartExternalDragResult sets what StartExternalDrag returns.
func (w *TestWindow) SetStartExternalDragResult(result bool) {
	w.lock().startExternalDragResult = result
	w.unlock()
}

// RawWindowHandle returns an empty handle: test windows are not backed by a platform window.
func (w *TestWindow) RawWindowHandle() RawWindowHandle {
	return RawWindowHandle{Kind: RawWindowHandleNone}
}

// VisualViewportBounds returns the simulated visual viewport, or the whole content area.
func (w *TestWindow) VisualViewportBounds() Bounds[Pixels] {
	s := w.lock()
	defer w.unlock()
	if viewport, ok := s.visualViewport.Get(); ok {
		return viewport
	}
	return NewBounds(Point[Pixels]{}, s.bounds.Size)
}

// OnVisualViewportChanged registers the visual viewport callback.
func (w *TestWindow) OnVisualViewportChanged(callback func()) {
	w.lock().visualViewportCallback = callback
	w.unlock()
}

// Insets returns the simulated insets.
func (w *TestWindow) Insets() WindowInsets {
	defer w.unlock()
	return w.lock().insets
}

// OnInsetsChanged registers the insets callback.
func (w *TestWindow) OnInsetsChanged(callback func(insets WindowInsets)) {
	w.lock().insetsCallback = callback
	w.unlock()
}

// ShowSoftKeyboard counts a soft keyboard request.
func (w *TestWindow) ShowSoftKeyboard() {
	w.lock().virtualKeyboardRequests++
	w.unlock()
}

// HideSoftKeyboard counts a soft keyboard dismissal.
func (w *TestWindow) HideSoftKeyboard() {
	w.lock().virtualKeyboardDismissals++
	w.unlock()
}

// Bounds returns the window bounds.
func (w *TestWindow) Bounds() Bounds[Pixels] {
	defer w.unlock()
	return w.lock().bounds
}

// WindowBounds returns the bounds as windowed bounds.
func (w *TestWindow) WindowBounds() WindowBounds {
	return WindowBoundsWindowed(w.Bounds())
}

// IsMaximized returns false.
func (w *TestWindow) IsMaximized() bool {
	return false
}

// ContentSize returns the size of the bounds.
func (w *TestWindow) ContentSize() Size[Pixels] {
	return w.Bounds().Size
}

// Resize sets the window size without running the resize callback.
func (w *TestWindow) Resize(size Size[Pixels]) {
	w.lock().bounds.Size = size
	w.unlock()
}

// ScaleFactor returns the scale factor, 2 by default.
func (w *TestWindow) ScaleFactor() float32 {
	defer w.unlock()
	return w.lock().scaleFactor
}

// Appearance returns the simulated appearance.
func (w *TestWindow) Appearance() WindowAppearance {
	defer w.unlock()
	return w.lock().appearance
}

// Display returns the window's display.
func (w *TestWindow) Display() PlatformDisplay {
	defer w.unlock()
	return w.lock().display
}

// MousePosition returns the origin.
func (w *TestWindow) MousePosition() Point[Pixels] {
	return Point[Pixels]{}
}

// Modifiers returns no modifiers.
func (w *TestWindow) Modifiers() Modifiers {
	return Modifiers{}
}

// Capslock returns capslock off.
func (w *TestWindow) Capslock() Capslock {
	return Capslock{}
}

// SetInputHandler stores the input handler.
func (w *TestWindow) SetInputHandler(inputHandler *PlatformInputHandler) {
	w.lock().inputHandler = inputHandler
	w.unlock()
}

// TakeInputHandler removes and returns the input handler.
func (w *TestWindow) TakeInputHandler() *PlatformInputHandler {
	s := w.lock()
	defer w.unlock()
	handler := s.inputHandler
	s.inputHandler = nil
	return handler
}

// SetTextInputConfiguration records the configuration.
func (w *TestWindow) SetTextInputConfiguration(configuration TextInputConfiguration) {
	s := w.lock()
	s.textInputConfigurations = append(s.textInputConfigurations, configuration)
	w.unlock()
}

// TextInputStateChanged records the change.
func (w *TestWindow) TextInputStateChanged(change TextInputStateChange) {
	s := w.lock()
	s.textInputStateChanges = append(s.textInputStateChanges, change)
	w.unlock()
}

func (w *TestWindow) upgradePlatform() *TestPlatform {
	defer w.unlock()
	return w.lock().platform.Value()
}

// Prompt queues the prompt on the test platform.
func (w *TestWindow) Prompt(level PromptLevel, msg string, detail Option[string], answers []PromptButton) *scheduler.OneshotReceiver[int] {
	platform := w.upgradePlatform()
	if platform == nil {
		panic("platform dropped")
	}
	return platform.prompt(msg, detail, answers)
}

// Activate makes the window the platform's active window.
func (w *TestWindow) Activate() {
	platform := w.upgradePlatform()
	if platform == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	platform.SetActiveWindow(w.Clone())
}

// IsActive returns false.
func (w *TestWindow) IsActive() bool {
	return false
}

// Visibility returns the simulated visibility.
func (w *TestWindow) Visibility() WindowVisibility {
	defer w.unlock()
	return w.lock().visibility
}

// IsHovered returns false.
func (w *TestWindow) IsHovered() bool {
	return false
}

// BackgroundAppearance returns WindowBackgroundAppearanceOpaque.
func (w *TestWindow) BackgroundAppearance() WindowBackgroundAppearance {
	return WindowBackgroundAppearanceOpaque
}

// IsSubpixelRenderingSupported returns false.
func (w *TestWindow) IsSubpixelRenderingSupported() bool {
	return false
}

// SetTitle records the title.
func (w *TestWindow) SetTitle(title string) {
	w.lock().title = Some(title)
	w.unlock()
}

// SetAppID does nothing.
func (w *TestWindow) SetAppID(appID string) {}

// SetBackgroundAppearance does nothing.
func (w *TestWindow) SetBackgroundAppearance(backgroundAppearance WindowBackgroundAppearance) {}

// SetEdited records the edited state.
func (w *TestWindow) SetEdited(edited bool) {
	w.lock().edited = edited
	w.unlock()
}

// SetDocumentPath records the document path.
func (w *TestWindow) SetDocumentPath(path Option[string]) {
	w.lock().documentPath = path
	w.unlock()
}

// ShowCharacterPalette is not implemented by the test window.
func (w *TestWindow) ShowCharacterPalette() {
	panic("not implemented")
}

// Minimize is not implemented by the test window.
func (w *TestWindow) Minimize() {
	panic("not implemented")
}

// Zoom is not implemented by the test window.
func (w *TestWindow) Zoom() {
	panic("not implemented")
}

// ToggleFullscreen flips the fullscreen state and simulates a resize from a foreground task.
func (w *TestWindow) ToggleFullscreen() {
	s := w.lock()
	s.isFullscreen = !s.isFullscreen
	platform := s.platform.Value()
	w.unlock()
	if platform != nil {
		window := w.Clone()
		platform.ForegroundExecutor().Spawn(func(sc *scheduler.Scope) (struct{}, error) {
			size := window.Bounds().Size
			window.SimulateResize(size)
			return struct{}{}, nil
		}).Detach()
	}
}

// IsFullscreen reports the fullscreen state.
func (w *TestWindow) IsFullscreen() bool {
	defer w.unlock()
	return w.lock().isFullscreen
}

// FrameWaker returns a waker that counts its calls, so tests can inspect wakes without
// delivering a frame synchronously.
func (w *TestWindow) FrameWaker() func() {
	frameWakeCount := w.lock().frameWakeCount
	w.unlock()
	window := weak.Make(w.state)
	return func() {
		*frameWakeCount++
		if testWindowFrameWakerSchedulesFrame {
			if state := window.Value(); state != nil {
				newTestWindowFromState(state).ScheduleFrame()
			}
		}
	}
}

// OnRequestFrame registers the frame request callback.
func (w *TestWindow) OnRequestFrame(callback func(options RequestFrameOptions)) {
	w.lock().requestFrameCallback = callback
	w.unlock()
}

// ScheduleFrame marks a frame as scheduled unless a drawn frame is still pending.
func (w *TestWindow) ScheduleFrame() {
	s := w.lock()
	if !s.frameCallbackPending {
		s.frameScheduled = true
	}
	w.unlock()
}

// OnInput registers the input callback.
func (w *TestWindow) OnInput(callback func(input PlatformInput) DispatchEventResult) {
	w.lock().inputCallback = callback
	w.unlock()
}

// OnActiveStatusChange registers the active status callback.
func (w *TestWindow) OnActiveStatusChange(callback func(active bool)) {
	w.lock().activeStatusChangeCallback = callback
	w.unlock()
}

// OnVisibilityChange registers the visibility callback.
func (w *TestWindow) OnVisibilityChange(callback func(visibility WindowVisibility)) {
	w.lock().visibilityCallback = callback
	w.unlock()
}

// OnHoverStatusChange registers the hover status callback.
func (w *TestWindow) OnHoverStatusChange(callback func(hovered bool)) {
	w.lock().hoverStatusChangeCallback = callback
	w.unlock()
}

// OnResize registers the resize callback.
func (w *TestWindow) OnResize(callback func(size Size[Pixels], scaleFactor float32)) {
	w.lock().resizeCallback = callback
	w.unlock()
}

// OnMoved registers the moved callback.
func (w *TestWindow) OnMoved(callback func()) {
	w.lock().movedCallback = callback
	w.unlock()
}

// OnDisplayChanged registers the display changed callback.
func (w *TestWindow) OnDisplayChanged(callback func()) {
	w.lock().displayChangedCallback = callback
	w.unlock()
}

// OnShouldClose registers the should-close handler.
func (w *TestWindow) OnShouldClose(callback func() bool) {
	w.lock().shouldCloseHandler = callback
	w.unlock()
}

// OnClose registers the close callback, which SimulateClose runs.
func (w *TestWindow) OnClose(callback func()) {
	w.lock().closeCallback = callback
	w.unlock()
}

// OnHitTestWindowControl registers the window control hit test callback.
func (w *TestWindow) OnHitTestWindowControl(callback func() (WindowControlArea, bool)) {
	w.lock().hitTestWindowControlCallback = callback
	w.unlock()
}

type testNativeViewAttachment struct {
	count  *int
	closed bool
}

func (a *testNativeViewAttachment) Close() {
	if a.closed {
		return
	}
	a.closed = true
	*a.count--
}

// AttachNativeView counts the view as attached without ever reading view, so tests may pass
// any value; closing the attachment detaches it.
func (w *TestWindow) AttachNativeView(id NativeViewID, view uintptr) NativeViewAttachment {
	attachedNativeViews := w.lock().attachedNativeViews
	w.unlock()
	*attachedNativeViews++
	return &testNativeViewAttachment{count: attachedNativeViews}
}

// SetFocusedNativeView records the focused native view.
func (w *TestWindow) SetFocusedNativeView(id Option[NativeViewID]) {
	w.lock().focusedNativeView = id
	w.unlock()
}

// OnKeyDownBeforeNativeView registers the key-down-before-native-view callback.
func (w *TestWindow) OnKeyDownBeforeNativeView(callback func(event KeyDownEvent) bool) {
	w.lock().keyDownBeforeNativeViewCallback = callback
	w.unlock()
}

// OnHitTestNativeView registers the native view hit test callback.
func (w *TestWindow) OnHitTestNativeView(callback func(position Point[Pixels]) (NativeViewID, bool)) {
	w.lock().hitTestNativeViewCallback = callback
	w.unlock()
}

// OnNativeViewFocused registers the native view focused callback.
func (w *TestWindow) OnNativeViewFocused(callback func(id NativeViewID)) {
	w.lock().nativeViewFocusedCallback = callback
	w.unlock()
}

// OnAppearanceChanged registers the appearance callback.
func (w *TestWindow) OnAppearanceChanged(callback func()) {
	w.lock().appearanceChangeCallback = callback
	w.unlock()
}

// Draw marks a frame as drawn and renders the scene when a headless renderer is configured.
func (w *TestWindow) Draw(scene *Scene) {
	scaleFactor := w.ScaleFactor()
	s := w.lock()
	defer w.unlock()
	s.frameCallbackPending = true
	s.frameScheduled = true
	deviceSize := s.bounds.Size.ToDevicePixels(scaleFactor)
	if s.renderer != nil {
		gpuiutil.Result(struct{}{}, s.renderer.RenderScene(scene, deviceSize)).WarnOnErr()
	}
}

// SpriteAtlas returns the window's sprite atlas.
func (w *TestWindow) SpriteAtlas() PlatformAtlas {
	defer w.unlock()
	return w.lock().spriteAtlas
}

// RenderToImage renders the scene with the headless renderer.
func (w *TestWindow) RenderToImage(scene *Scene) (*image.RGBA, error) {
	scaleFactor := w.ScaleFactor()
	s := w.lock()
	defer w.unlock()
	size := s.bounds.Size
	if s.renderer != nil {
		deviceSize := size.ToDevicePixels(scaleFactor)
		return s.renderer.RenderSceneToImage(scene, deviceSize)
	}
	return nil, errors.New("render_to_image not available: no HeadlessRenderer configured")
}

// AsTest returns the window itself.
func (w *TestWindow) AsTest() *TestWindow {
	return w
}

// GetRawHandle is not implemented by the test window on Windows.
func (w *TestWindow) GetRawHandle() uintptr {
	if runtime.GOOS == "windows" {
		panic("not implemented")
	}
	return w.PlatformWindowDefaults.GetRawHandle()
}

// ShowWindowMenu is not implemented by the test window.
func (w *TestWindow) ShowWindowMenu(position Point[Pixels]) {
	panic("not implemented")
}

// StartWindowMove is not implemented by the test window.
func (w *TestWindow) StartWindowMove() {
	panic("not implemented")
}

// CanStartExternalDrag returns true.
func (w *TestWindow) CanStartExternalDrag() bool {
	return true
}

// StartExternalDrag records the dragged files and returns the result set with SetStartExternalDragResult.
func (w *TestWindow) StartExternalDrag(payload ExternalDragPayload) bool {
	s := w.lock()
	defer w.unlock()
	switch payload := payload.(type) {
	case ExternalDragPayloadFiles:
		s.externalDragFiles = append(s.externalDragFiles, payload.Entries()...)
	}
	return s.startExternalDragResult
}

// UpdateIMEPosition does nothing.
func (w *TestWindow) UpdateIMEPosition(bounds Bounds[Pixels]) {}

// GPUSpecs returns no specs.
func (w *TestWindow) GPUSpecs() (GPUSpecs, bool) {
	return GPUSpecs{}, false
}

// WindowDecorations returns the simulated decorations.
func (w *TestWindow) WindowDecorations() Decorations {
	defer w.unlock()
	return w.lock().decorations
}
