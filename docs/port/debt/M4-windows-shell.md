# M4-windows-shell debt

- dialog.go: scheduler.OneshotSender has no `cancellation()` future and `OneshotReceiver.Close` does not wake the sender, so showDialog returns a local `DialogReceiver[T]` whose Close fires a cancellation signal the dialog task selects on. Replace with `OneshotSender.Cancellation()` once scheduler has it (handoff below).
- dialog.go: futures::lock::Mutex is a local `dialogMutex` (tryLock, cancellable lock future, whenIdle future). Unlock wakes every waiter (they re-race) rather than futures' single waiter hand-off.
- dialog.go: the "windows-native-dialog" thread is a native thread from win32.CreateThread with a NewCallback start routine (lint forbids runtime.LockOSThread outside package platform); the thread is unnamed. The thread-local CANCELLATION slot is a sync.Map keyed by GetCurrentThreadId.
- dialog.go: CancellationTimer's error uses xwin.GetLastError() after win32.SetTimer, which discards the call's own error value.
- direct_manipulation.go: ZoomToRect (f32 arguments) goes through purego.RegisterFunc (local dmFloatFn, same pattern as internal/win/d3d and dwrite).
- clipboard.go: image conversion uses stdlib png/jpeg/gif and x/image bmp/tiff/webp in place of the image crate; decode/encode error texts differ.
- destination_list.go: Rust Path::file_name is approximated by pathFileName (None for roots, volumes and `..`). Strings passed to COM are truncated at an embedded NUL (HSTRING would carry it, but the callee reads to the first NUL).
- system_notifications.go: DefaultHasher is a local SipHash-1-3 (zero keys), pinned against rustc output in system_notifications_test.go. The TypedEventHandler<ToastNotification, IInspectable> IID is the precomputed pinterface GUID {AB54DE2D-97D9-5528-B6AD-105AFE156530}. Activated responses go through one shared unbounded ChanSender instead of a clone per toast.
- Interface wrappers (DirectManipulation, ICustomDestinationList, IObjectArray/Collection, IShellLinkW, IPropertyStore, WinRT XmlDocument / Toast*) and PROPERTYKEY live in platform/windows; M4-ci-windows may move them to internal/win.
- destination_list.go: removed recent entries are matched with slices.Equal on path strings; Rust compares PathBuf components (trailing separators, / vs \ equal). Entries come back as written, so this rarely differs.
