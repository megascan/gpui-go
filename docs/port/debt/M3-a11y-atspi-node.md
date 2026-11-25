# Debt: M3-a11y-atspi-node

- a11y/atspi/stubs.go holds real (non-panicking) context.rs code (`Context`, `AppContext`, lock helpers) so PlatformNode is testable; M3-a11y-atspi-adapter moves it to context.go.
- Nothing sets `Context.released` yet, so a PlatformNode reports Defunct only after GC until the adapter stage sets it on adapter drop (D-ATSPIWEAK).
- `Context` is exported though upstream is `pub(crate)`; the adapter stage decides the final visibility.
