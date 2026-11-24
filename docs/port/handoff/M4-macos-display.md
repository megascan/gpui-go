# M4-macos-display handoff

- platform/macos/pasteboard.go readImage calls gpui.Hash, still a panicking stub in root stubs.go (M0-app-contexts). TestPasteboard_ReadImage is `blocked:M0-app-contexts` and will panic on the macOS runner until Hash is ported.
- TestDisplayLink_ARendererDrawsNormallyAfterItsDisplayLinkIsInvalidated is a t.Skip until MetalRenderer exists (M4-apple-metal-renderer); port its body then.
- display.go binds CGGetActiveDisplayList/CGDisplayBounds/CGDisplayCopyDisplayMode/CFUUIDGetUUIDBytes locally; display_link.go binds CoreVideo and the dispatch source API locally (internal/apple/gcd has no dispatch sources).
