# M4-macos-platform debt

- platform/macos/stubs.go: panicking stubs for MacWindow, MacWindowOpen, MacWindowActiveWindow, MacWindowOrderedWindows, setActiveWindowCursorStyle, nativeCursor and rendererContext (upstream renderer::Context, Arc<Mutex<InstanceBufferPool>>). M4-macos-window replaces them with the window.rs port and the platform/apple metal renderer context, then deletes stubs.go. Until then OpenWindow, ActiveWindow, WindowStack, SetCursorStyle and DebugState panic.
- invokeForeignBlock (system_notifications.go) calls Objective-C-created completion blocks through the Block_layout invoke pointer at offset 16 with purego.SyscallN, because purego's Block.Invoke only knows Go-created blocks. Move to internal/apple/objc if another package needs it.
- Menu items are built with alloc/init (+1) and released after being added to their parent; separatorItem is retained to match. The dock menu is held in MacPlatformState.dockMenu and released when replaced.
- ScreenCaptureFrame.Frame is a borrowed CVImageBufferRef (D-MACOS-SCKFRAME).
