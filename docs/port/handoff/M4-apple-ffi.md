# M4-apple-ffi handoff

- `GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet ...` fails before vetting any code with go1.27.4: "ios/arm64 requires external (cgo) linking, but cgo is not enabled" (reproduced on ./gpuiutil). The Verify blocks of M4-ios-app, M4-ios-window and the iOS line at verify.sh:525 use that form and will fail as written; an exit stage needs to change them together with STAGES.md (e.g. CGO_ENABLED=1 with an iOS SDK on the macOS runner, or `go vet` via a darwin constraint check).
- TestProcessInfo_NotificationsAreDeferredAndCancelledOnDrop (platform/apple/process_info_test.go) is blocked:M0-test-contexts: replace its t.Skip with gpui.RunTest and pass the TestAppContext's foreground/background executors and RunUntilParked to checkNotificationLifecycle.
