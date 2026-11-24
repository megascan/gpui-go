# Debt: M4-web-example

- fonts/IBMPlexSans-Regular.ttf and fonts/Lilex-Regular.ttf are copies of text/gotext/testdata/fonts (go:embed cannot reach outside the package). Replace with a shared font package when one exists.
- test_http_client.mjs and test_canvas_fallback.mjs need headless Chrome and were not run (no browser launched); font_fallback_testing.go and http_client_testing.go need a browser window, so they run only in the browser drivers.
- The Go wasm test drivers serve wasm_exec.js from GOROOT by hand; there is no build script for the gallery yet.
