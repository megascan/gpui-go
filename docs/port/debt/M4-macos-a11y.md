# Debt: M4-macos-a11y

- a11y/macos runs only on macOS; locally it is vetted and cross-compiled for darwin/arm64 and darwin/amd64, never executed. node_test.go runs on the macOS CI runner.
- The dynamic subclass relies on purego's `SendSuper`, which uses the receiver's current class. A view class whose Go IMPs call `objc.SendSuper` would recurse after `object_setClass` swaps in `AccessKitSubclassOf<X>`. GPUI's window content view is the plain NSView AppKit creates, so this does not occur today; anything that subclasses the adapted view with SendSuper-calling IMPs must use an explicit superclass instead.
- Contexts referenced by unraised QueuedEvents are released with the adapter (D-MACA11YWEAK), not when the last reference goes away.
