# Debt: M1-linux-common

- `openCommands` omits open-5.4.5's WSL PowerShell launcher (D-LINUX-OPEN).
- `runOpenCommands` (and the portal calls in `OpenURIInternal` / `RevealPathInternal` / `InhibitIdleSleep`) block the background worker goroutine on `exec.Cmd.Wait` and on the portal `Response` signal, where upstream awaits them on smol; a portal that never answers keeps one worker busy (ashpd also waits without a timeout).
- `LinuxDispatcher.IsMainThread` parses `runtime.Stack` (copied from the root's unexported `goroutineID` / coroutine-creator helpers); costly if called on hot paths.
- `SpawnRealtime` does not raise the thread to SCHED_FIFO 65 (D-LINUX-RTPRIO).
- `newTextSystem` returns `gpui.NoopTextSystem` until M2-exit wires text/gotext.
