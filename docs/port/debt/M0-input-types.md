# Debt: M0-input-types

- `KeybindingKeystroke` keeps the Windows display fields on every OS and branches on `runtime.GOOS` instead of per-GOOS files (the stage owned only platform_keystroke.go). Behaviour matches upstream per OS; `NewKeybindingKeystroke` is callable on every OS.
- `ActionData.JSONSchema` takes no shared `jsonschema.Generator`, so schemas do not share `$defs` across actions as schemars' `SchemaGenerator` does. Revisit when keymap schema generation is ported.
