# M4-macos-window-objc debt

- platform/macos/stubs.go now also holds panicking stubs for native_views.rs symbols window.rs calls: nativeViewsAddWindowMethods, nativeViewsAddViewMethods (class registration), checkPointerAfterFrame, nativeViewSetsCursor, isRoutingKeyEquivalent, gpuiViewIsFocused. Class registration (and so MacWindowOpen) panics until M4-macos-native-views replaces them.
- Super calls go through objc.SendSuper (superclass of the receiver's class), so GPUIPanel's dealloc/close/tab overrides reach NSPanel's implementation before NSWindow's; upstream names NSWindow explicitly. Equivalent since NSPanel does not override these.
- attributedSubstringForProposedRange: writes actualRange only when AppKit passes a non-nil pointer (upstream writes unconditionally).
- a11yAdapter gained UpdateViewFocusState(isFocused) (accesskit_macos SubclassingAdapter::update_view_focus_state); M4-macos-a11y must implement it.
- Pre-existing, not from this stage: `paritycheck -area macos -lint` reports platform/macos/system_notifications.go:111 (runtime.LockOSThread outside package platform) and platform/ios/gpui_ios_impl.go (build constraint).
