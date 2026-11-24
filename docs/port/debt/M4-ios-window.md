# M4-ios-window debt

- platform/ios/integration_text_input_test.go locks the OS thread in a test-file `init` (upstream's harness is a `harness = false` binary on the main thread); the rule "LockOSThread only in package platform" is kept for non-test code. Exit: accept or route through a shared test helper.
- examples/ios_ios/export.go uses cgo (`//go:build ios && cgo`, `//export gpui_ios_main`) because `-buildmode=c-archive` exports only through cgo; the cgo allow-list names only render/wgpu and wlspike. CGO_ENABLED=0 builds skip the file.
- platform/ios/window.go keeps a package counter `windowReferenceQueries` incremented on every native window lookup; it is the Go form of the test fixture's counting `with_window` in upstream tests/text_input.rs.
- IosWindow.Close defers releasing the UIWindow / views / Metal renderer until no UIKit callback into the window is running (dispatch depth 0); this replaces upstream's Rc keeping IosWindowState alive across a synchronous close.
- Weak `inputDelegate` uses objc_storeWeak / objc_loadWeakRetained / objc_destroyWeak on a Go-heap slot (Go's GC does not move objects); internal/apple/objc has no weak helper.
- iOS code is compiled and vetted only (ios-vet.sh workaround); nothing in platform/ios/*ios* has run.
