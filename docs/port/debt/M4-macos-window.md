# M4-macos-window debt

- platform/macos/stubs.go survives this stage (contrary to M4-macos-platform's debt note): it now holds panicking stubs for the second half of window.rs (class accessors windowClass/panelClass/viewClass/blurredViewClass/windowStateArchiverDelegateClass/windowStateUnarchiverClass, isGpuiWindow, getWindowState, addMouseTrackingArea, nativeCursor, visibility, isMacosVersionAtLeast, nsErrorDescription, stepCallback, metalFrame, displayIDForScreen) and for window/native_views.rs (nativeViews type, nativeViews* funcs). Until M4-macos-window-objc and M4-macos-native-views land, MacWindowOpen and most window methods panic.
- Window state "ivar": a package map objc.ID -> *lockedWindowState (storeWindowState / loadWindowState / clearWindowState in window.go), same pattern as D-MACOS-PLATFORMIVAR, because Go pointers cannot live in ObjC ivars.
- gpui.WindowKindLayerShell (Linux-only upstream, present on every GOOS in Go) opens as a Normal window.
- MergeAllWindows / MoveTabToNewWindow use gcd.MainQueue().Async(closure) for dispatch_async_f.
- Standard window buttons and titlebar container views are used as +0 references for the duration of a call instead of objc2's Retained.
- platform/macos/a11y.go is a nil seam (newA11yAdapter returns nil) until M4-macos-a11y.
- paritycheck -lint flags platform/macos/system_notifications.go:111 (runtime.LockOSThread outside package platform), written by M4-macos-platform; not changed here.
