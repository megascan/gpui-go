# Owner checks: M4-web-example

YOU: serve examples/web_hello_web as the gallery root (index.html, gallery.js, wasm_exec.js copied from `$(go env GOROOT)/lib/wasm/`, gpui-web-examples.wasm built with `GOOS=js GOARCH=wasm go build -o <dir>/gpui-web-examples.wasm ./examples/web_hello_web`), open `/prime-sieve` in Chrome with WebGPU, check the window renders the prime sieve UI and "Count Primes" runs to completion with the history line.
YOU: same page with `?backend=webgl` (WebGL2 fallback), check it renders and runs.
YOU: open `/dynamic-fonts` with the network throttled or blocked, check the Devanagari sample falls back and the status line changes to "Font loaded" after the download, and "Font loading failed" when blocked.
YOU: run `node test_canvas_fallback.mjs http://127.0.0.1:8080` (needs Chrome, not run here), check it passes.
YOU: run `node test_http_client.mjs http://127.0.0.1:8080` (needs Chrome, not run here), check it passes.
