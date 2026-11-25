# M4-macos-window handoff

To M4-macos-window-objc:
- Append window.rs 2651-4344 to platform/macos/window.go and delete from stubs.go only the window.rs stubs: windowClass, panelClass, viewClass, blurredViewClass, windowStateArchiverDelegateClass, windowStateUnarchiverClass (register build_classes lazily, once, e.g. sync.OnceValue; register windowStateArchiverWillEncodeObject and windowStateUnarchiverRestorationOptions, already ported in window.go), isGpuiWindow, getWindowState, addMouseTrackingArea, nativeCursor, visibility, isMacosVersionAtLeast (platform.go has nsOperatingSystemVersion and msgSendIsOperatingSystemAtLeast), nsErrorDescription, stepCallback (a gcd.Function for WindowFrameSource; data is the native view pointer), metalFrame, displayIDForScreen. Keep the signatures.
- Window state ivar contract: MacWindowOpen calls storeWindowState(nativeWindow, state) and storeWindowState(nativeView, state). getWindowState / window_state_if_set read with loadWindowState (panic with upstream's message when missing); dealloc_window / dealloc_view / drop_window_state remove with clearWindowState.
- MacWindow.Close is MacWindow's Drop; lockedWindowState embeds sync.Mutex and macWindowState (upstream Rc<Mutex<MacWindowState>>). startDisplayLink takes the *lockedWindowState and must be called with the lock held, as upstream.
- Leave the nativeViews type and the nativeViews* stubs in stubs.go for M4-macos-native-views; do not delete stubs.go.

To M4-macos-native-views:
- Replace the nativeViews stub type (keep fields background, hitTestCallback, focusedCallback, keyDownBeforeCallback) and nativeViewsPlanLayout / nativeViewsUpdateBackground / nativeViewsSceneDrawn / nativeViewsApplyLayout / nativeViewsAttach / nativeViewsSetFocused in window_native_views.go, then delete stubs.go if it is empty.

To M4-macos-a11y:
- platform/macos/a11y.go is the seam: newA11yAdapter(window, *a11yActivationHandler, *a11yActionHandler) returns an a11yAdapter (UpdateIfActive returning queued events with Raise, Close as drop). MacWindow.Close closes the adapter first, as upstream drops it.

Resolved: M4-macos-platform (MacWindowOpen signature kept; rendererContext = apple.Context from NewInstanceBufferPool) and M4-apple-metal-renderer (Close calls renderer.Destroy where upstream's Drop calls destroy()).
