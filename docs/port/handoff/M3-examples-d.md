# Handoff: M3-examples-d

## To M3-exit: Window.HandleInput leaks the input handler when the element is not focused

- Where: window_dispatch.go `Window.HandleInput`. Upstream (window.rs `handle_input`) takes the handler by value; when `focus_handle.is_focused(self)` is false the handler is dropped, which drops the `ElementInputHandler`'s `Entity<V>`. The Go port only stores the handler when focused and never calls `Release` on it otherwise, so the view handle inside `ElementInputHandler` leaks and strict leak detection fails.
- Repro: in examples/input/main.go or examples/view_example/example_editor.go remove the `if !focusHandle.IsFocused(window) { inputHandler.Release() }` lines after `window.HandleInput` and run `go test ./examples/input -run NativeSelection` ("Handle for entity main.TextInput ... was collected without Release").
- Fix: in HandleInput's not-focused branch release the handler when it implements `Release()` (ElementInputHandler does). The example workaround is harmless after the fix (Release is idempotent) and can then be removed.

## To M3-exit: verify block writes binaries into the repo root

- The M3-examples-d Verify block runs a bare `go build ./examples/$e`, which leaves `view_example`, `testing`, `input` and `animation` binaries (about 44 MB each) in the worktree root; this stage deleted the ones its runs produced. Use `go build -o /dev/null` (same note as M3-examples-b/c).
