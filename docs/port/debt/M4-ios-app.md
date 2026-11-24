# M4-ios-app debt

- ios.rs is split: `CurrentPlatform` and the CVPixelBuffer surface-size registration live in ios.go (`ios`), `CallbackSlot` in callback_slot.go (`darwin`) so the application and window tests compile on darwin.
- The application's window registry holds strong `applicationWindow` interface values instead of `Weak<IosWindowState>`; windows must unregister on close (upstream's Drop does the same).
- `CURRENT_PLATFORM` and the delegate ivars hold `weak.Pointer[IosApplicationState]` (the application half of the platform state), not a weak pointer to the whole platform state.
