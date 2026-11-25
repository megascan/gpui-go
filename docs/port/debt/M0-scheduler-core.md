# M0-scheduler-core debt

- S-THREADCHECK: local tasks are not checked for being polled or dropped on their spawning thread (`scheduler/src/executor.rs:780-859` panics). Go has no goroutine identity. Fix only if a cheap goroutine id appears.
- S-DEDICATEDEXIT: a `SpawnDedicatedThread` session goroutine exits when a `runtime.AddCleanup` on its dispatch handle closes the run queue, i.e. after GC finds every executor copy and task of the session unreachable. Upstream exits at the last `Arc` drop. Only thread lifetime depends on GC; tasks never stop through GC.
- Waker reference counts are not tracked. A coroutine parked on a primitive whose other side is abandoned without `Close`/`Send` keeps its goroutine until that primitive is closed; upstream frees the future when the last waker is dropped. Every primitive in this package wakes or drops its stored wakers on `Close`.
- `Chan[T](n)` with n > 0 holds exactly n values; futures' bounded mpsc holds n plus one per sender.
- `Select` is biased (`select_biased!`); futures' `select!` picks a random ready arm.
- A panic in a task body is re-raised by iter.Pull's `next`, so the stack trace printed by the poll loop starts at `next`, not at the original panic site.
- `Location` records file and line, no column.
- `BlockOn`, `BlockWithTimeout` and `Scheduler.Block` also exist on js/wasm (upstream `cfg(not(target_family = "wasm"))`).
- The realtime loop of `BackgroundExecutor.SpawnRealtime` exits when its task finishes or is dropped (upstream: when its sender's last copy drops, which is the same moment).
- `*Task[T]`'s own fields (`kind`, `spawned`, ...) are not synchronized: canceling one handle on one goroutine while another goroutine polls that same handle is a data race. Upstream's uniquely owned `Task` cannot express it; cancel from another goroutine only with a happens-before edge to the handle's owner (the task state behind the handle is mutex-protected).
- A body that recovers the cancel unwind (`IsCancelUnwind` reports true) keeps running inside `stop()` until it returns; every later await in it panics again and its output is discarded. Go cannot forbid recover; Rust drops the future without running it further.
- A panic in a task on a `SpawnDedicatedThread` session or a realtime loop is recovered, printed to stderr with its stack (Rust's default panic hook), and ends that goroutine: queued runnables are closed and later spawns on the session resolve to canceled tasks, as a Rust thread unwinding drops its receiver.
- A pop from a bounded `Chan` wakes every parked sender (each Send future keeps one slot, replaced on re-poll and removed on cancel or completion); futures' mpsc unparks one sender per freed slot.
- `Select` with no cases panics ("scheduler: Select with no cases"); `select_biased!` with no branches does not compile.
- `DedicatedExecutor` spawns share the handle's sender instead of a clone each; the session's lifetime is tied to `Close`, which cancels it, so only the sender count differs.
