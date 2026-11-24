# M0-taffy-style handoff

For the owner of internal/cmd/paritycheck/scan (an exit stage). Seen while seeding docs/port/parity/ext-taffy.md from a scan of taffy-0.15.0/src; worked around by hand, no change needed for this stage:

- `cfg_attr(feature = "serde", derive(...))` is reported as a `serde` cfg on the item, although it does not gate the item (`cfgOf` treats `cfg_attr` like `cfg`).
- `cfg(not(feature = "x"))` is reported as `x`.
- Names of nested test modules are dropped (`mod tests { mod lhs_option_f32_rhs_f32 { fn test_maybe_min } }` is reported as `test_maybe_min`, so several rows share a name).
- `pub fn` inside a private module (`mod compat` in style/compact_length.rs) is listed as a public fn.
