# Debt: M1-linux-screencap

- `internal/linux/pw`: `Loop.Iterate` calls `pw_loop_enter`, `pw_loop_iterate`, `pw_loop_leave` as three purego calls from a goroutine that is not locked to an OS thread (only package platform may call `runtime.LockOSThread`). The goroutine can migrate between the calls, so the loop's recorded owner thread may differ from the iterating thread. The calls work today; if PipeWire starts checking the thread on iterate, move the three calls into one pinned thread.
- `internal/linux/pw`: when libpipewire does not export `pw_loop_enter/leave/iterate` (releases before 1.2), the binding calls `spa_loop_control_methods` through the vtable at offsets that only `pw_test.go` checks.
- `screencap.decodeCompoundText` duplicates a small part of `platform/linux/x11/internal/xim/ctext.go`, which the screencap package cannot import (internal visibility). See handoff.
- Wayland: the portal `Response` signal match is added after the method call returns, as upstream; a portal that answers before the match is installed loses the response ("Did not get response").
- `Capturer.Close` releases the Wayland D-Bus connection; nothing calls it until the root glue (M3-root-capture) does.
- X11 on Xwayland (this machine, :0): core `GetImage` of a full RandR display rect on the root window fails with BadMatch, while MIT-SHM `GetImage` succeeds. Upstream (core only) would fail there too; the core fallback is only reached when SHM is unavailable.
- The gated X11 test passes an explicit RandR display: Xwayland reports no primary output, so the default-target path (`GetOutputPrimary`) fails with BadOutput, as upstream would.
