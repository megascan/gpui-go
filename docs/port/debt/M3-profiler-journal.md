# M3-profiler-journal debt

- `maxJournalEntries` is 4 MiB / (sizeof slot + sizeof PresentedFrame): Go slots box entries behind an interface, so the bound approximates upstream's fixed 4 MiB ring rather than matching its slot count.
- D-JOURNALTHREAD: one process-wide journal; `endTurn` on depth 0 returns without the debug assertion because tests share the journal earlier Apps installed. Revisit once the dispatchers only journal the main thread.
- D-SUSPENDCLOCK: no suspend clock on any OS until the per-OS setters land (handoff).
- `durationMulF64` rounds via math.Round on the fractional nanoseconds, close to but not bit-identical with Rust's `Duration::from_secs_f64`.
