# Debt: M0-foundation

- refinegen Debug output (`String()`) uses Go field names, not upstream's snake_case names.
- refineable.Eq falls back to reflect.DeepEqual for non-comparable types without an `Eq(T) bool` method; DeepEqual differs from Rust PartialEq for NaN inside slices and for func values.
- refinegen ignores `JsonSchema` / `Deserialize` / other directive args (schemas come from internal/jsonschema, D-SCHEMA); `Serialize` emits `json:"<snake or source json name>,omitzero"` tags.
- Nested refineable fields convert with `<Type>FromRefinement` (free function); a nested type from another package must export that function (refinegen-generated types do).
- `TruncateToBottomNSortedBy` uses its own quickselect; on ties the kept elements may differ from Rust's `select_nth_unstable_by` (equal under compare).
