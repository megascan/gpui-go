# M4-macos-window-objc handoff

To M4-macos-native-views:
- Replace the stubs in platform/macos/stubs.go: nativeViewsAddWindowMethods() / nativeViewsAddViewMethods() return the []objc.MethodDef that upstream's add_window_methods / add_view_methods add (appended to the class method lists in buildWindowClass / buildWindowClasses); checkPointerAfterFrame(*lockedWindowState) is called with the lock released; nativeViewSetsCursor(*macWindowState) and gpuiViewIsFocused(*macWindowState) with the lock held; isRoutingKeyEquivalent(*lockedWindowState) with it released.

To M4-macos-a11y:
- a11yAdapter now also requires UpdateViewFocusState(isFocused bool) (a11yQueuedEvents, bool); windowDidChangeKeyStatus calls it with the lock held and raises the events after unlocking.

To M4-exit / owner of platform/macos/system_notifications.go:
- paritycheck -lint flags system_notifications.go:111 runtime.LockOSThread outside package platform (pre-existing).
