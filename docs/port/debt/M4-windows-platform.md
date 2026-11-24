# M4-windows-platform debt

- platform/windows/platform.go `scapScreenSources` panics (`gpui: unimplemented ScapScreenSources`) until the root scap glue exists (M3-root-capture), same as platform/linux/x11. Replace with the root's `ScapScreenSources(executor)`.
- `PromptForPaths` / `PromptForNewPath` forward `DialogReceiver` into a `scheduler.Oneshot` through a foreground task (`forwardDialog`); dropping the returned receiver cancels the dialog only when the forwarding task next polls `doneTx.IsCanceled()`. Upstream returns show_dialog's receiver directly. Collapses once `OneshotSender.Cancellation()` lands (see M4-windows-shell handoff).
- `trackedWindows.dropped` stands in for the failed `Weak<RwLock<..>>::upgrade` after the platform is dropped (set in `WindowsPlatform.Close`).
- `TestPlatform_Clipboard` is gated behind `GPUI_TEST_CLIPBOARD=1` (test isolation); CI must set it to run it.
