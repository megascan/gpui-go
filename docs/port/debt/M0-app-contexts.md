# M0-app-contexts debt

- RunTest retries only panics: t.Fatal / t.Error cannot be undone, so such a failure ends the run after OnFailure and the failing-seed report (testing.go).
- Context.Processor captures a strong entity handle for the closure's lifetime, as upstream; under strict leak detection a dropped processor is reported as a collected handle.
- Hash: values that are neither Hashable nor a basic Rust-equivalent type hash through hash/maphash (per-process seed), not FxHasher; Resource hashes a path as a str, not Rust's component-wise Path hash.
- VisualContextUpdateWindowEntity (gpui.go, not owned) uses the context window for AsyncWindowContext; upstream's AsyncWindowContext::update_window_entity uses with_window(view) ("entity has no current window").
- asset_cache.rs tests are blocked:M0-test-contexts (need TestAppContext); asset_cache_test.go has FxHasher vector tests and an inference compile check.
- App.FetchAsset / RemoveAsset / HasAsset (app.rs rows) live in asset_cache.go because app.go is not owned by this stage.
