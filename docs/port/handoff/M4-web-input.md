
## verify.sh: wasm test fails on env size, not on code

`go test -exec go_js_wasm_exec` fails before any Go code runs with "total length of command line and environment variables exceeds limit": wasm_exec.js caps argv+env at 12288 bytes, and the harness shell env is ~9 KB plus the GO* vars go test adds. With a trimmed env the package passes:

    env -i PATH="$PATH" HOME="$HOME" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/...   # ok

Fix (exit stage, STAGES.md + verify.sh together): run every js/wasm `go test -exec` line under `env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)"` (affects M4-web-input, M4-web-text and any other wasm test lines, and the ci.yml wasm step if CI env grows).
