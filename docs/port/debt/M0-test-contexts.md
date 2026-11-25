# Debt: M0-test-contexts

- Leak report runs before RunTest drains the scheduler (see handoff); pending tasks holding handles show as leaks.
- `reportAppLeaks` waits for GC cleanups with a sentinel cleanup plus up to 1s per round and stops after two stable rounds (max 32); cleanups may run concurrently, so a slow cleanup could be missed.
- `releaseAppWindows` emulates dropping `App` by clearing the window slot map and flushing effects; window root views are not released yet (needs M0-window-core's window drop).
- `TestAppContext.drop` runs OnQuit callbacks, which upstream's test macro never calls (only `App::quit`); harmless in Go since IntoMut allocates nothing.
- `Entity.NextNotification` does not end its stream when the entity is released (upstream's sender is dropped with the observer).
