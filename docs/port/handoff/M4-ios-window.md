# M4-ios-window handoff

- Exit stage (paritycheck lint owner): examples/ios_ios/export.go is a cgo file (`ios && cgo`) required by the c-archive build the brief asks for; allow cgo for examples/ios_ios in lint.go or accept the finding. platform/ios/window_test.go is `darwin` (tests callbackSlot, which is darwin) like application_test.go; same lint note as the M4-ios-app handoff.
- Exit stage (CI): platform/ios ios-tagged tests (text_input_test.go, integration_text_input_test.go) compile with the ios-vet.sh go.work workaround and `-tags ios`; running them needs an iOS simulator runner (owner check in docs/port/owner-checks/M4-ios-window.md).
