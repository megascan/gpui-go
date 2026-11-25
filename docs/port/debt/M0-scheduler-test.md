# M0-scheduler-test debt

- S-GOID: wrong-thread detection parses goroutine ids from runtime.Stack; a non-owner goroutine is checked once for `created by iter.Pull` and cached per scheduler. A foreign goroutine waking during a poll of a coroutine is not distinguished from the coroutine itself only if it was created by iter.Pull.
- D-RNG: seeds expand through Rust's PCG32 seed_from_u64 stream into math/rand/v2 PCG; draws (random_range, random_bool, random::<bool>) differ from rand's StdRng, so seeds give different interleavings than upstream. Statistical tests (test_block_with_timeout's None/Some mix, priority, interleaving) pass with this seeding.
- Pending traces are Go call stacks captured at TracingWaker clone (frames above Clone), not resolved Rust backtraces.
- test_send_local_executor_worker_submission_and_wakeup cannot assert the polling thread (S-THREADCHECK).
