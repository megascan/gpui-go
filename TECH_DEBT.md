# Tech debt

Hacks, workarounds and known issues. Remove entries when resolved. Stages write new items to `docs/port/debt/<stage>.md`; exit stages fold them in here.

## Index

| Id | Area | Summary |
|---|---|---|
| L-WGPUVER | render/wgpu | wgpu-native v29 binding against upstream's wgpu 30 |
| L-WEBTHREADS | platform/web | no multi-threaded web dispatcher |
| GO-DIRECTIVE | go.mod | `go 1.27.3` instead of `go 1.27` |

## L-WGPUVER

Bump github.com/oliverbestmann/webgpu when it ships wgpu-native v30, then re-run the M1-wgpu-context feature probe and drop the fallbacks it no longer needs.

## L-WEBTHREADS

Go wasm is single-threaded; revisit if Go gains wasm threads.

## GO-DIRECTIVE

github.com/xo/okage v0.20.1 declares `go 1.27.3`, so the main module's go line cannot be lower than that; go.mod says `go 1.27.3` rather than the planned `go 1.27`. CI's `1.27.x` satisfies it. Drop when the module moves to a newer Go anyway.

