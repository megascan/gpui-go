# M0-a11y-schema debt

- a11y JSON vs serde_json: f64 values print in Go's shortest form (`1` where serde_json writes `1.0`); values round-trip but bytes differ. Decoding `null` into a non-Option field, or omitting a required field of the plainly-decoded structs (CustomAction, TextPosition, TextSelection, Color, TextDecoration, Tree, TreeUpdate), is accepted where serde errors; Node and ActionRequest check missing fields by hand.
- `deny_unknown_fields` structs decode with `json.RejectUnknownMembers(true)`, which also applies to nested geometry types (serde accepts unknown fields there).
- `debugStr` approximates Rust's `str` Debug escaping (`\u{..}` for non-printable runes via unicode.IsPrint, not Rust's grapheme-extend table).
- Node getters for slice properties return the internal slice (Rust returns `&[T]`); callers must not mutate it.
