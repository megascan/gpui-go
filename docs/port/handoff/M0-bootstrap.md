# M0-bootstrap handoff

## Missing module: github.com/oliverbestmann/webgpu/libs-linux

- For: orchestrator (needs network), before M1-wgpu-context.
- github.com/oliverbestmann/webgpu/wgpu/libs_linux.go (`linux && !android && cgo`) imports github.com/oliverbestmann/webgpu/libs-linux. The install step left only a `.lock` file for it in the module cache and no entry in go.mod or go.sum.
- Repro: `CGO_ENABLED=1 GOFLAGS=-mod=readonly go build -tags deps -o /dev/null ./internal/deps` fails with "missing go.sum entry for module providing package github.com/oliverbestmann/webgpu/libs-linux".
- Fix: `cd /home/strange/Projects/gpui-go-wt/port && go get github.com/oliverbestmann/webgpu/libs-linux@v0.0.0-20261112101129-823eab7d2391` (the version webgpu v1.38.1's go.mod requires). Not `go mod tidy`: it would add the darwin/windows/ios/android library modules too.
- After the fix the same build command must exit 0; then drop WEBGPU-LIBS from TECH_DEBT.md.

## go directive is 1.27.3

github.com/xo/okage v0.20.1 requires `go 1.27.3`; go.mod therefore says `go 1.27.3`. No action needed (TECH_DEBT GO-DIRECTIVE).
