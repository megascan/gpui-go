# M4-web-input debt

- go_js_wasm_exec aborts when the environment is large (Go wasm env/argv limit); CI and verify.sh need a trimmed environment on heavy developer shells.
- Event handlers are not exercised by tests (no DOM under node); covered by the web example owner check.
