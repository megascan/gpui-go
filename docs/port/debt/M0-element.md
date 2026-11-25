# M0-element debt

- A `ViewElement` for an entity or `AnyView` that is built but never laid out (dropped before `RequestLayout`) never registers with `Window.deferArenaDrop`, so its handle is released only by the GC safety net; strict leak mode reports it. Upstream drops it with the arena.
- `(*AnyElement).DowncastMut[T]` matches only elements stored as `*T`; a value element (such as `Empty{}`) cannot be downcast to a pointer into the drawable.
- `ElementID` placeholder bodies in stubs.go (`NewElementID`, `Hash`, `String`, `GoString`) cover Integer, Name and NamedInteger only; M0-window-core replaces them.
- `ViewElement` component ids use `typeNameOf` (`pkgpath.Name`), not Rust's `type_name` path; ids only need to be unique per type.
