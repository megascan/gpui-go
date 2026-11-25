# M0-actiongen handoff

- `ActionData.JSONSchema` is generated as `func() *jsonschema.Schema` (`jsonschema.NewGenerator().Reflect(reflect.TypeFor[T]())`), not DESIGN 6.4's `func() []byte`. M0-input-types must declare the field with that type; the schema is marshalled by the caller.
- Generated `Build` is `func(jsontext.Value) (gpui.Action, error)` and decodes with `encoding/json/v2`; unit structs return `T{}` without reading the value; `no_json` returns `errors.New("<name> cannot be built from JSON")`.
- Generated `ActionData` literal fields: `Name`, `Type`, `Build`, `JSONSchema` (omitted for unit and no_json), `DeprecatedAliases`, `DeprecationMessage`, `Documentation`. Zero values are omitted.
- Generated `PartialEq` is `ok && a == o`, so an action struct must be comparable (a compile error otherwise), matching upstream's derived `PartialEq`.
- Inside package `gpui` (the root), run `go run ./internal/cmd/actiongen -self <files>`: identifiers are unqualified and the root does not import itself.
- Generated files need the root (`gpui.Action`, `gpui.ActionData`, `gpui.RegisterAction`) and `internal/jsonschema`; they compile once M0-input-types lands.
