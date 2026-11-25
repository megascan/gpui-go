# M0-app-effects debt

- `App.Notify` visits a window's invalidators in ascending WindowID order (upstream: FxHashMap order).
- cfg-gated methods (`ReadFromPrimary`, `WriteToPrimary`, `ReadFromFindPasteboard`, `WriteToFindPasteboard`, `WriteCredentialsIfNotExists`, `ClearGlobals`, `HasAsset`-to-be, `GetName`, `SetInspectorRenderer`, `RegisterInspectorElement`, `ForegroundJournal`, `StartHangMonitor`) exist on every GOOS / without feature tags, as the Platform interface does (D-FEATURE).
- `ActionSchemas` / `ActionSchemaByName` take no schema generator (D-SCHEMA: the registry calls each action's reflector); `ActionSchemas` returns exported `ActionSchema{Name, Schema}`.
- `HTTPClient` (D-HTTP) is `Send(*http.Request) *Task[*http.Response]`, `UserAgent`, `Proxy`; none of http_client's provided helpers.
- `flushEffects`' test-mode redraw runs whenever `mode.test` (upstream additionally cfg(test/test-support)); `event_arena.clear()` is dropped (D-ARENA).
- Panic texts for anyhow unwraps are rendered as `called \`Result::unwrap()\` on an \`Err\` value: <msg>`; `ObserveNew` downcast failure omits upstream's Debug of the entity.
