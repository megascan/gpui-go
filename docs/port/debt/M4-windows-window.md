# M4-windows-window debt

- window.go Prompt: scheduler has no `OneshotSender.Cancellation()` future, so the select against `done_tx.cancellation()` polls `doneTx.IsCanceled()` only when the dialog future wakes the task; dropping the receiver does not close an open task dialog until it returns.
- window.go: GWLP_USERDATA holds an id into a package registry of `weak.Pointer[WindowsWindowInner]` (upstream stores a boxed `Weak`); CreateWindowExW's lpParam is an id into a create-context map, so no Go pointer passes through native memory.
- window.go: TASKDIALOGCONFIG / TASKDIALOG_BUTTON (pack(1)) are laid out by hand in byte slices with 64-bit offsets; amd64 and arm64 only.
- window.go: SetWindowCompositionAttribute is resolved with a LazySystemDLL proc instead of GetModuleHandleA + GetProcAddress; RtlGetVersion comes from x/sys (no status to check).
- window.go Close: the spawned destroy task releases DirectManipulation and destroys the renderer before DestroyWindow (STAGES brief); upstream drops both with the last Rc after DestroyWindow. Draw is skipped once the renderer is destroyed.
- IDataObject / IDropTargetHelper wrappers live in window.go; M4-ci-windows may move them to internal/win.
