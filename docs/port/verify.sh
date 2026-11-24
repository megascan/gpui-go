#!/bin/sh
# Usage: sh docs/port/verify.sh <stage name>|final. Generated from docs/port/STAGES.md; keep the two in step.
cd "$(dirname "$0")/../.." || exit 2
set -x
case "$1" in
M0-bootstrap)
  ( go build ./... && go vet ./... && go test ./internal/... && test -z "$(gofmt -l .)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -lint ) || exit 1
  ;;
M0-foundation)
  ( go build ./gpuiutil/... && go vet ./gpuiutil/... && go test ./gpuiutil/... && test -z "$(gofmt -l gpuiutil)" ) || exit 1
  ( go build ./refineable/... && go vet ./refineable/... && go test ./refineable/... && test -z "$(gofmt -l refineable)" ) || exit 1
  ( go build ./internal/cmd/refinegen/... && go vet ./internal/cmd/refinegen/... && go test ./internal/cmd/refinegen/... && test -z "$(gofmt -l internal/cmd/refinegen)" ) || exit 1
  ( go build ./collections/... && go vet ./collections/... && go test ./collections/... && test -z "$(gofmt -l collections)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area foundation -done gpui_util/,refineable/,collections/,gpui_tokio/ ) || exit 1
  ;;
M0-scheduler-core)
  ( go build ./scheduler/... && go vet ./scheduler/... && go test ./scheduler/... && test -z "$(gofmt -l scheduler)" && go test -race ./scheduler/... ) || exit 1
  ;;
M0-scheduler-test)
  ( go build ./scheduler/... && go vet ./scheduler/... && go test ./scheduler/... && test -z "$(gofmt -l scheduler)" && go test -count=3 ./scheduler/... && go test -race ./scheduler/... ) || exit 1
  ( go run ./internal/cmd/paritycheck -area scheduler -done scheduler/ ) || exit 1
  ;;
M0-sumtree)
  ( go build ./sumtree/... && go vet ./sumtree/... && go test ./sumtree/... && test -z "$(gofmt -l sumtree)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area sumtree -done sum_tree/ ) || exit 1
  ;;
M0-a11y-geometry)
  ( go build ./a11y && go vet ./a11y && go test ./a11y && test -z "$(gofmt -l a11y)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y ) || exit 1
  ;;
M0-a11y-schema)
  ( go build ./a11y && go vet ./a11y && go test ./a11y && test -z "$(gofmt -l a11y)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y ) || exit 1
  ;;
M0-jsonschema)
  ( go build ./internal/jsonschema/... && go vet ./internal/jsonschema/... && go test ./internal/jsonschema/... && test -z "$(gofmt -l internal/jsonschema)" ) || exit 1
  ;;
M0-etagere)
  ( go build ./internal/etagere/... && go vet ./internal/etagere/... && go test ./internal/etagere/... && test -z "$(gofmt -l internal/etagere)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-etagere ) || exit 1
  ;;
M0-lyon)
  ( go build ./internal/lyon/... && go vet ./internal/lyon/... && go test ./internal/lyon/... && test -z "$(gofmt -l internal/lyon)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-lyon ) || exit 1
  ;;
M0-parity-rescan)
  ( go build ./internal/cmd/paritycheck/... && go vet ./internal/cmd/paritycheck/... && go test ./internal/cmd/paritycheck/... && test -z "$(gofmt -l internal/cmd/paritycheck)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -rescan -area foundation,sumtree,scheduler ) || exit 1
  ;;
M0-stylegen)
  ( go build ./internal/cmd/stylegen/... && go vet ./internal/cmd/stylegen/... && go test ./internal/cmd/stylegen/... && test -z "$(gofmt -l internal/cmd/stylegen)" ) || exit 1
  ;;
M0-actiongen)
  ( go build ./internal/cmd/actiongen/... && go vet ./internal/cmd/actiongen/... && go test ./internal/cmd/actiongen/... && test -z "$(gofmt -l internal/cmd/actiongen)" ) || exit 1
  ;;
M0-taffy-style)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-style-mod)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-style-grid)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-tree)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-compute)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-fixtures)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ( s=$(ls internal/taffy/testdata/xml.zip | sort | xargs sha256sum) && go generate ./internal/taffy/... && [ "$s" = "$(ls internal/taffy/testdata/xml.zip | sort | xargs sha256sum)" ] ) || exit 1
  ;;
M0-taffy-flexbox)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-block)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-grid-types)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-grid-placement)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-grid-sizing)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-taffy ) || exit 1
  ;;
M0-taffy-handwritten-a)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ;;
M0-taffy-handwritten-b)
  ( go build ./internal/taffy/... && go vet ./internal/taffy/... && go test ./internal/taffy/... && test -z "$(gofmt -l internal/taffy)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-taffy -all-done ) || exit 1
  ;;
M0-geometry)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-geometry,foundation -done gpui_shared_string/,gpui/src/shared_uri.rs ) || exit 1
  ;;
M0-geometry-units)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-geometry -done gpui/src/geometry.rs,gpui/src/color.rs ) || exit 1
  ;;
M0-contracts)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-geometry -done gpui/src/gpui.rs ) || exit 1
  ;;
M0-platform-types)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-platform,core-input -done gpui/src/platform/layer_shell.rs,gpui/src/platform/platform_info.rs,gpui/src/platform/keyboard.rs ) || exit 1
  ;;
M0-platform-data)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-platform -done gpui/src/platform.rs,gpui/src/platform/app_menu.rs,gpui/src/platform/popup.rs ) || exit 1
  ;;
M0-input-types)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-input,core-app -done gpui/src/platform/keystroke.rs,gpui/src/action.rs,gpui/src/global.rs,gpui/src/assets.rs ) || exit 1
  ;;
M0-scene)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go build ./render && go vet ./render && test -z "$(gofmt -l render/render.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-style,core-geometry -done gpui/src/scene.rs,gpui/src/bounds_tree.rs,gpui/src/path_builder.rs,gpui/src/path_winding.rs ) || exit 1
  ;;
M0-text-system)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-text -done gpui/src/text_system/font_fallbacks.rs,gpui/src/text_system/font_features.rs,gpui/src/text_system/line_layout.rs ) || exit 1
  ;;
M0-line-wrapper)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ;;
M0-profiler-core)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler.rs,gpui/src/profiler/actions.rs,gpui/src/profiler/hang/monitor.rs ) || exit 1
  ;;
M0-entities)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go test -race -run 'Entity|Subscription' . ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app/entity_map.rs,gpui/src/subscription.rs ) || exit 1
  ;;
M0-executors)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go test -race -run 'Executor|Scheduler|Dispatcher|Queue' . ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-app,core-platform -done gpui/src/executor.rs,gpui/src/platform_scheduler.rs,gpui/src/platform/threaded_dispatcher.rs,gpui/src/queue.rs ) || exit 1
  ;;
M0-app-core)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ;;
M0-app-effects)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app.rs ) || exit 1
  ;;
M0-app-contexts)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app/context.rs,gpui/src/app/async_context.rs ) || exit 1
  ;;
M0-test-platform)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-platform -done gpui/src/platform/test/ ) || exit 1
  ;;
M0-test-contexts)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-app -done gpui/src/app/headless_app_context.rs,gpui/src/asset_cache.rs ) || exit 1
  ;;
M0-keymap)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-input -done gpui/src/keymap.rs,gpui/src/keymap/,gpui/src/tab_stop.rs ) || exit 1
  ;;
M0-gestures-spring)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-input,core-style -done gpui/src/gestures.rs,gpui/src/spring.rs ) || exit 1
  ;;
M0-style)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( s=$(ls styled_gen.go style_gen.go geometry_gen.go | sort | xargs sha256sum) && go generate . && [ "$s" = "$(ls styled_gen.go style_gen.go geometry_gen.go | sort | xargs sha256sum)" ] ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-style -done gpui/src/style.rs,gpui/src/styled.rs ) || exit 1
  ;;
M0-element)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-window,core-style -done gpui/src/view.rs,gpui/src/taffy.rs,gpui/src/inspector.rs ) || exit 1
  ;;
M0-window-core)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ;;
M0-window-paint)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ;;
M0-window-dispatch)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-window -done gpui/src/window/prompts.rs ) || exit 1
  ;;
M0-div-api)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ;;
M0-div-paint)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-elements,core-window -done gpui/src/elements/div.rs,gpui/src/element.rs ) || exit 1
  ;;
M0-text-elements)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-text,core-elements -done gpui/src/text_system/line.rs,gpui/src/elements/text.rs,gpui/src/text_system.rs,gpui/src/text_system/line_wrapper.rs ) || exit 1
  ;;
M0-window-tests)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-window -done gpui/src/window.rs ) || exit 1
  ;;
M0-blocked-tests)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-input,core-app -done gpui/src/key_dispatch.rs,gpui/src/interactive.rs,gpui/src/input.rs,gpui/src/app.rs,gpui/src/app/test_context.rs,gpui/src/app/test_app.rs ) || exit 1
  ;;
M0-elements-basic)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-elements -done gpui/src/elements/anchored.rs,gpui/src/elements/animation.rs,gpui/src/elements/canvas.rs,gpui/src/elements/container_query.rs,gpui/src/elements/deferred.rs,gpui/src/elements/surface.rs,gpui/src/elements/uniform_list.rs ) || exit 1
  ;;
M0-list)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ;;
M0-elements-media)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-elements,core-text -done gpui/src/elements/img.rs,gpui/src/elements/image_cache.rs,gpui/src/elements/svg.rs,gpui/src/svg_renderer.rs ) || exit 1
  ;;
M0-window-a11y)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-window,core-elements -done gpui/src/window/,gpui/src/elements/list.rs ) || exit 1
  ;;
M0-exit)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( PK=". ./scheduler/... ./sumtree/... ./collections/... ./gpuiutil/... ./refineable/... ./a11y ./internal/taffy/... ./internal/lyon/... ./internal/etagere/... ./internal/jsonschema/... ./internal/cmd/paritycheck/... ./internal/cmd/refinegen/... ./internal/cmd/actiongen/... ./internal/cmd/stylegen/..."; go vet $PK && go test $PK && go test -race . ./scheduler/... && for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet $PK && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null $PK || exit 1; done ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,macros,core-tests -rescan ) || exit 1
  ( gh pr checks feat/port || echo 'CI-PENDING: Actions billing blocked, see TECH_DEBT CI-BILLING' )
  ;;
M1-wayland-ffi)
  ( go build ./internal/cmd/wlgen/... && go vet ./internal/cmd/wlgen/... && go test ./internal/cmd/wlgen/... && test -z "$(gofmt -l internal/cmd/wlgen)" ) || exit 1
  ( go build ./internal/linux/wl/... && go vet ./internal/linux/wl/... && go test ./internal/linux/wl/... && test -z "$(gofmt -l internal/linux/wl)" ) || exit 1
  ( go build ./internal/linux/wlproto/... && go vet ./internal/linux/wlproto/... && go test ./internal/linux/wlproto/... && test -z "$(gofmt -l internal/linux/wlproto)" ) || exit 1
  ( s=$(find internal/linux/wlproto -name '*_gen.go' | sort | xargs sha256sum) && go generate ./internal/linux/wlproto/... && [ "$s" = "$(find internal/linux/wlproto -name '*_gen.go' | sort | xargs sha256sum)" ] ) || exit 1
  ( go vet ./internal/cmd/wlspike && { [ -z "$WAYLAND_DISPLAY" ] || timeout 60 go run ./internal/cmd/wlspike -frames 90; } ) || exit 1
  ;;
M1-linux-ffi)
  ( go build ./internal/linux/xkb/... && go vet ./internal/linux/xkb/... && go test ./internal/linux/xkb/... && test -z "$(gofmt -l internal/linux/xkb)" ) || exit 1
  ( go build ./internal/linux/xcbshim/... && go vet ./internal/linux/xcbshim/... && go test ./internal/linux/xcbshim/... && test -z "$(gofmt -l internal/linux/xcbshim)" ) || exit 1
  ;;
M1-x11-proto)
  ( go build ./platform/linux/x11/internal/xext/... && go vet ./platform/linux/x11/internal/xext/... && go test ./platform/linux/x11/internal/xext/... && test -z "$(gofmt -l platform/linux/x11/internal/xext)" ) || exit 1
  ( go build ./platform/linux/x11/internal/xim/... && go vet ./platform/linux/x11/internal/xim/... && go test ./platform/linux/x11/internal/xim/... && test -z "$(gofmt -l platform/linux/x11/internal/xim)" ) || exit 1
  ;;
M1-linux-dbus)
  ( go build ./platform/linux/internal/dbus/... && go vet ./platform/linux/internal/dbus/... && go test ./platform/linux/internal/dbus/... && test -z "$(gofmt -l platform/linux/internal/dbus)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-dbus -all-done ) || exit 1
  ;;
M1-wgpu-context)
  ( go build ./render/wgpu/... && go vet ./render/wgpu/... && go test ./render/wgpu/... && test -z "$(gofmt -l render/wgpu)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/wgpu_context.rs,gpui_wgpu/src/wgpu_atlas.rs ) || exit 1
  ;;
M1-wgpu-renderer)
  ( go build ./render/wgpu/... && go vet ./render/wgpu/... && go test ./render/wgpu/... && test -z "$(gofmt -l render/wgpu)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/ ) || exit 1
  ;;
M1-linux-common)
  ( go build ./platform/linux/internal/common/... && go vet ./platform/linux/internal/common/... && go test ./platform/linux/internal/common/... && test -z "$(gofmt -l platform/linux/internal/common)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-common ) || exit 1
  ;;
M1-wayland-client)
  ( go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-wayland -done gpui_linux/src/linux/wayland/serial.rs,gpui_linux/src/linux/wayland/display.rs,gpui_linux/src/linux/wayland/cursor.rs ) || exit 1
  ;;
M1-wayland-input)
  ( go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)" ) || exit 1
  ;;
M1-wayland-window)
  ( go build ./platform/linux/wayland/... && go vet ./platform/linux/wayland/... && go test ./platform/linux/wayland/... && test -z "$(gofmt -l platform/linux/wayland)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-wayland -done gpui_linux/src/linux/wayland ) || exit 1
  ;;
M1-x11-client)
  ( go build ./platform/linux/x11 && go vet ./platform/linux/x11 && go test ./platform/linux/x11 && test -z "$(gofmt -l platform/linux/x11)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-x11 -done gpui_linux/src/linux/x11/event.rs,gpui_linux/src/linux/x11/xim_handler.rs ) || exit 1
  ;;
M1-x11-window)
  ( go build ./platform/linux/x11/... && go vet ./platform/linux/x11/... && go test ./platform/linux/x11/... && test -z "$(gofmt -l platform/linux/x11)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-x11 -done gpui_linux/src/linux/x11 ) || exit 1
  ;;
M1-linux-platform)
  ( go build ./platform/... && go vet ./platform/... && go test ./platform ./platform/linux && test -z "$(gofmt -l platform/*.go platform/linux/*.go)" ) || exit 1
  ( for t in darwin/arm64 windows/amd64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./platform && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null ./platform || exit 1; done ) || exit 1
  ( go run ./internal/cmd/paritycheck -area linux-platform,linux-common,platform-select -done gpui_linux/src/,gpui_platform/src/ ) || exit 1
  ;;
M1-linux-screencap)
  ( go build ./internal/linux/pw/... && go vet ./internal/linux/pw/... && go test ./internal/linux/pw/... && test -z "$(gofmt -l internal/linux/pw)" ) || exit 1
  ( go build ./platform/linux/internal/screencap/... && go vet ./platform/linux/internal/screencap/... && go test ./platform/linux/internal/screencap/... && test -z "$(gofmt -l platform/linux/internal/screencap)" ) || exit 1
  ( go vet ./platform/linux && test -z "$(gofmt -l platform/linux/screencap.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-scap-linux -all-done ) || exit 1
  ;;
M1-exit)
  ( go build -o /dev/null ./examples/hello_world && go vet ./examples/hello_world ./internal/cmd/shoot && go test . ./platform/... ./render/... ./internal/linux/... ) || exit 1
  ( go test -race . ./scheduler/... ./sumtree/... ./collections/... ./gpuiutil/... ./refineable/... ./internal/taffy/... ./internal/linux/... ./platform/linux/internal/... ) || exit 1
  ( go run ./internal/cmd/shoot -pkg ./examples/hello_world -out docs/port/screenshots/m1/hello_world_wayland.png && go run ./internal/cmd/shoot -x11 -pkg ./examples/hello_world -out docs/port/screenshots/m1/hello_world_x11.png ) || exit 1
  ( gh pr checks feat/port || echo 'CI-PENDING: Actions billing blocked, see TECH_DEBT CI-BILLING' )
  ;;
M2-gotext-layout)
  ( go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)" ) || exit 1
  ;;
M2-gotext-raster)
  ( go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)" ) || exit 1
  ;;
M2-gotext-color)
  ( go build ./text/gotext/... && go vet ./text/gotext/... && go test ./text/gotext/... && test -z "$(gofmt -l text/gotext)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area gotext -done gpui_wgpu/ ) || exit 1
  ;;
M2-exit)
  ( go build -o /dev/null ./examples/... && go vet ./examples/... ./text/... ./platform/linux/internal/common && go test . ./text/... ./platform/... ./render/... ) || exit 1
  ( for e in hello_world text svg_example image_example; do go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/m2/$e.png || exit 1; done ) || exit 1
  ( go run ./internal/cmd/paritycheck -area examples-m2 -all-done ) || exit 1
  ( gh pr checks feat/port || echo 'CI-PENDING: Actions billing blocked, see TECH_DEBT CI-BILLING' )
  ;;
M3-profiler-journal)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler/journal.rs ) || exit 1
  ;;
M3-bench-context)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-profiler,core-tests -done gpui/src/bench.rs,gpui/src/app/bench_context.rs,gpui/benches/ ) || exit 1
  ;;
M3-randomized-tree)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go test -run xxx -tags gpui_release -bench FocusCleanup -benchtime 1x . ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/randomized_element_tree.rs ) || exit 1
  ;;
M3-profiler-hang)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-profiler -done gpui/src/profiler/hang.rs,gpui/src/debug_overlay.rs ) || exit 1
  ;;
M3-root-capture)
  ( go build . && go vet . && go test . && test -z "$(gofmt -l *.go)" ) || exit 1
  ( test ! -e stubs.go && test ! -e DOCS && test ! -e docs/DOCS ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-platform,macros -done gpui/src/platform/scap_screen_capture.rs,gpui/src/platform/visual_test.rs ) || exit 1
  ( go run ./internal/cmd/paritycheck -area core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,core-profiler,macros ) || exit 1
  ;;
M3-a11y-consumer-tree)
  ( go build ./a11y/consumer/... && go vet ./a11y/consumer/... && go test ./a11y/consumer/... && test -z "$(gofmt -l a11y/consumer)" ) || exit 1
  ;;
M3-a11y-consumer-node)
  ( go build ./a11y/consumer/... && go vet ./a11y/consumer/... && go test ./a11y/consumer/... && test -z "$(gofmt -l a11y/consumer)" ) || exit 1
  ;;
M3-a11y-consumer-text)
  ( go build ./a11y/consumer/... && go vet ./a11y/consumer/... && go test ./a11y/consumer/... && test -z "$(gofmt -l a11y/consumer)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y-consumer -all-done ) || exit 1
  ;;
M3-a11y-atspi-node)
  ( go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)" ) || exit 1
  ;;
M3-a11y-atspi-adapter)
  ( go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y-atspi -done accesskit_atspi_common-0.19.1/ ) || exit 1
  ;;
M3-a11y-atspi-bus)
  ( go build ./a11y/atspi/... && go vet ./a11y/atspi/... && go test ./a11y/atspi/... && test -z "$(gofmt -l a11y/atspi)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y-atspi -all-done ) || exit 1
  ;;
M3-platform-benches)
  ( go vet ./platform && go test -run xxx -bench Benchmark -benchtime 1x ./platform && test -z "$(gofmt -l platform/bench_atlas_upload_test.go platform/bench_inspector_render_test.go platform/bench_primitives_test.go platform/bench_randomized_element_tree_test.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area platform-select -all-done ) || exit 1
  ;;
M3-examples-a)
  ( E="active_state_bug anchor grid_layout opacity gradient pattern scrollable tree uniform_list list_example data_table paths_bench popover"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-a/$e.png || exit 1; done ) || exit 1
  ( go run ./internal/cmd/paritycheck -area examples-a -all-done ) || exit 1
  ;;
M3-examples-b)
  ( E="text_layout text_wrapper focus_visible tab_stop drag_drop mouse_pressure a11y svg_animation shadow painting"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-b/$e.png || exit 1; done ) || exit 1
  ( go build -o /dev/null ./examples/ownership_post && go vet ./examples/ownership_post && test -z "$(gofmt -l examples/ownership_post)" && go run ./internal/cmd/shoot -no-window 5s -pkg ./examples/ownership_post ) || exit 1
  ( go run ./internal/cmd/paritycheck -area examples-b -all-done ) || exit 1
  ;;
M3-examples-c)
  ( E="window window_movable window_positioning window_shadow on_window_close_quit move_entity_between_windows set_menus gif_viewer image_gallery image_loading native_view native_view_appkit_field"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -may-exit -pkg ./examples/$e -out docs/port/screenshots/examples-c/$e.png || exit 1; done ) || exit 1
  ( for e in layer_shell toggle_headless; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -no-window 5s -pkg ./examples/$e || exit 1; done ) || exit 1
  ( go build -o /dev/null ./examples/system_notifications && go vet ./examples/system_notifications && test -z "$(gofmt -l examples/system_notifications)" && dbus-run-session -- go run ./internal/cmd/shoot -may-exit -pkg ./examples/system_notifications -out docs/port/screenshots/examples-c/system_notifications.png ) || exit 1
  ( go run ./internal/cmd/paritycheck -area examples-c -all-done ) || exit 1
  ;;
M3-examples-d)
  ( E="view_example testing input animation"; for e in $E; do go build -o /dev/null ./examples/$e && go vet ./examples/$e && go test ./examples/$e && test -z "$(gofmt -l examples/$e)" && go run ./internal/cmd/shoot -pkg ./examples/$e -out docs/port/screenshots/examples-d/$e.png || exit 1; done ) || exit 1
  ( go run ./internal/cmd/paritycheck -area examples-d -all-done ) || exit 1
  ;;
M3-exit)
  ( go vet ./... && go test ./... && test -z "$(gofmt -l .)" ) || exit 1
  ( go test -race $(go list ./... | grep -vE '/(examples|render/wgpu|internal/cmd/wlspike)(/|$)|/platform/linux/(wayland|x11)(/|$)') ) || exit 1
  ( s=$(find . -name '*_gen.go' -not -path './internal/apple/*' -not -path './internal/win/*' -not -path './platform/apple/*' -not -path './platform/macos/*' -not -path './platform/windows/*' | sort | xargs sha256sum) && go generate . ./a11y/... ./internal/taffy/... ./internal/linux/... ./platform/linux/... && [ "$s" = "$(find . -name '*_gen.go' -not -path './internal/apple/*' -not -path './internal/win/*' -not -path './platform/apple/*' -not -path './platform/macos/*' -not -path './platform/windows/*' | sort | xargs sha256sum)" ] ) || exit 1
  ( X='internal/(apple|win)|platform/(apple|macos|ios|windows)|a11y/(macos|windows)'; for t in darwin/arm64 windows/amd64 js/wasm; do P=$(GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go list ./... | grep -vE "$X"); GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet $P && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null $P || exit 1; done ) || exit 1
  ( go run ./internal/cmd/paritycheck -area foundation,sumtree,scheduler,macros,core-geometry,core-app,core-input,core-platform,core-window,core-style,core-text,core-elements,core-profiler,core-tests,examples-m2,examples-a,examples-b,examples-c,examples-d,platform-select,linux-common,linux-dbus,linux-platform,linux-wayland,linux-x11,wgpu,gotext,ext-scap-linux,web -all-done -allow-blocked -lint -rescan ) || exit 1
  ( gh pr checks feat/port || echo 'CI-PENDING: Actions billing blocked, see TECH_DEBT CI-BILLING' )
  ;;
M4-apple-ffi)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/objc/... ./internal/apple/cf/... ./internal/apple/gcd/... ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/objc/... ./internal/apple/cf/... ./internal/apple/gcd/... ./platform/apple/... || exit 1; done && test -z "$(gofmt -l internal/apple/objc internal/apple/cf internal/apple/gcd platform/apple)" ) || exit 1
  ;;
M4-apple-text)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/ct/... ./internal/apple/cg/... ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/ct/... ./internal/apple/cg/... ./platform/apple/... || exit 1; done && test -z "$(gofmt -l internal/apple/ct internal/apple/cg platform/apple)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area apple -done gpui_apple/src/text_system.rs,gpui_apple/src/open_type.rs,gpui_apple/src/dispatcher.rs,gpui_apple/src/keychain.rs ) || exit 1
  ;;
M4-apple-metal-atlas)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/mtl/... ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/mtl/... ./platform/apple/... || exit 1; done && test -z "$(gofmt -l internal/apple/mtl platform/apple internal/cmd/metalhdr)" ) || exit 1
  ( go build ./internal/cmd/metalhdr/... && go vet ./internal/cmd/metalhdr/... && go test ./internal/cmd/metalhdr/... && test -z "$(gofmt -l internal/cmd/metalhdr)" ) || exit 1
  ;;
M4-apple-metal-renderer)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/apple/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/apple/... || exit 1; done && test -z "$(gofmt -l platform/apple)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area apple -all-done ) || exit 1
  ;;
M4-macos-appkit)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./internal/apple/appkit/... ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/apple/appkit/... ./platform/macos/... || exit 1; done && test -z "$(gofmt -l internal/apple/appkit platform/macos)" ) || exit 1
  ;;
M4-macos-display)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)" ) || exit 1
  ;;
M4-macos-platform)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)" ) || exit 1
  ;;
M4-macos-window)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)" ) || exit 1
  ;;
M4-macos-window-objc)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)" ) || exit 1
  ;;
M4-macos-native-views)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/macos/... || exit 1; done && test -z "$(gofmt -l platform/macos)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area macos -all-done ) || exit 1
  ;;
M4-macos-a11y)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./a11y/macos/... ./platform/macos/... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./a11y/macos/... ./platform/macos/... || exit 1; done && test -z "$(gofmt -l a11y/macos platform/macos)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y-macos -all-done ) || exit 1
  ;;
M4-ios-app)
  ( sh docs/port/ios-vet.sh ./internal/apple/... ./platform/ios/... ./platform && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go test -c -o /dev/null ./platform/ios/... && test -z "$(gofmt -l platform/ios internal/apple/uikit)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ios -done gpui_ios/src/ios/application.rs,gpui_ios/src/ios/events.rs,gpui_ios/src/ios/platform.rs ) || exit 1
  ;;
M4-ios-window)
  ( sh docs/port/ios-vet.sh ./internal/apple/... ./platform/ios/... ./platform && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go test -c -o /dev/null ./platform/ios/... && test -z "$(gofmt -l platform/ios examples/ios_ios)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ios -all-done -allow-blocked ) || exit 1
  ;;
M4-windows-com)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/com/... ./internal/win/win32/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/com/... ./internal/win/win32/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l internal/win/com internal/win/win32 platform/windows)" ) || exit 1
  ;;
M4-windows-directwrite)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/dwrite/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/dwrite/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l internal/win/dwrite platform/windows)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area windows -done gpui_windows/src/direct_write.rs,gpui_windows/src/vsync.rs ) || exit 1
  ;;
M4-windows-d3d-atlas)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/d3d/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/d3d/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l internal/win/d3d platform/windows)" ) || exit 1
  ;;
M4-windows-directx-renderer)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)" ) || exit 1
  ;;
M4-windows-events)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)" ) || exit 1
  ;;
M4-windows-shell)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)" ) || exit 1
  ;;
M4-windows-window)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)" ) || exit 1
  ;;
M4-windows-platform)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./platform/windows/... || exit 1; done && test -z "$(gofmt -l platform/windows)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area windows -all-done ) || exit 1
  ;;
M4-windows-a11y-node)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./a11y/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./a11y/windows/... || exit 1; done && test -z "$(gofmt -l a11y/windows)" ) || exit 1
  ;;
M4-windows-a11y)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./a11y/windows/... ./platform/windows/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./a11y/windows/... ./platform/windows/... || exit 1; done && test -z "$(gofmt -l a11y/windows platform/windows)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-a11y-windows -all-done ) || exit 1
  ;;
M4-windows-screencap)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./internal/win/wgc/... ./internal/win/screencap/... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./internal/win/wgc/... ./internal/win/screencap/... || exit 1; done && test -z "$(gofmt -l internal/win/wgc internal/win/screencap)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area ext-scap-windows -all-done ) || exit 1
  ;;
M4-web-wgpu-js)
  ( GOOS=js GOARCH=wasm go vet ./render/... && GOOS=js GOARCH=wasm go build ./render/... && GOOS=js GOARCH=wasm go test -c -o /dev/null ./render/wgpu/... && go build ./render/... && go test ./render/... && test -z "$(gofmt -l render)" ) || exit 1
  ;;
M4-web-webgl-shaders)
  ( go build ./internal/cmd/glslgen/... && go vet ./internal/cmd/glslgen/... && go test ./internal/cmd/glslgen/... && test -z "$(gofmt -l internal/cmd/glslgen)" ) || exit 1
  ( go build ./internal/webglsl/... && go vet ./internal/webglsl/... && go test ./internal/webglsl/... && test -z "$(gofmt -l internal/webglsl)" ) || exit 1
  ( command -v naga >/dev/null || exit 0; s=$(find internal/webglsl -name '*.glsl' -o -name '*_gen.go' | sort | xargs sha256sum) && go generate ./internal/webglsl && [ "$s" = "$(find internal/webglsl -name '*.glsl' -o -name '*_gen.go' | sort | xargs sha256sum)" ] ) || exit 1
  ;;
M4-web-webgl)
  ( GOOS=js GOARCH=wasm go vet ./render/... && GOOS=js GOARCH=wasm go build ./render/... && GOOS=js GOARCH=wasm go test -c -o /dev/null ./render/wgpu/... && go build ./render/... && go test ./render/... && test -z "$(gofmt -l render)" ) || exit 1
  ( env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./render/wgpu/internal/gpu/webgl2/... ) || exit 1
  ( go run ./internal/cmd/paritycheck -area wgpu -done gpui_wgpu/src/ ) || exit 1
  ;;
M4-web-platform)
  ( GOOS=js GOARCH=wasm go vet ./platform/web/... && GOOS=js GOARCH=wasm go test -c -o /dev/null ./platform/web/... && test -z "$(gofmt -l platform/web)" ) || exit 1
  ;;
M4-web-input)
  ( GOOS=js GOARCH=wasm go vet ./platform/web/... && env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... && test -z "$(gofmt -l platform/web)" ) || exit 1
  ;;
M4-web-text)
  ( GOOS=js GOARCH=wasm go vet ./platform/... && env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... && test -z "$(gofmt -l platform/web platform/platform_js.go)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area web -done gpui_web/src/ ) || exit 1
  ;;
M4-web-example)
  ( GOOS=js GOARCH=wasm go vet ./examples/web_hello_web && GOOS=js GOARCH=wasm go build -o /dev/null ./examples/web_hello_web && test -z "$(gofmt -l examples/web_hello_web)" ) || exit 1
  ( go run ./internal/cmd/paritycheck -area web -all-done -allow-blocked ) || exit 1
  ;;
M4-ci-macos)
  ( for a in arm64 amd64; do GOOS=darwin GOARCH=$a CGO_ENABLED=0 go vet ./... && GOOS=darwin GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./... || exit 1; done && test -z "$(gofmt -l platform/apple platform/macos platform/ios internal/apple a11y/macos platform/platform_darwin.go)" ) || exit 1
  ( gh pr checks feat/port ) || exit 1
  ;;
M4-ci-windows)
  ( for a in amd64 arm64; do GOOS=windows GOARCH=$a CGO_ENABLED=0 go vet ./... && GOOS=windows GOARCH=$a CGO_ENABLED=0 go test -c -o /dev/null ./... || exit 1; done && test -z "$(gofmt -l platform/windows internal/win a11y/windows platform/platform_windows.go)" ) || exit 1
  ( gh pr checks feat/port ) || exit 1
  ;;
M4-exit)
  ( test -z "$(gofmt -l .)" && go vet ./... && go test ./... ) || exit 1
  ( s=$(find . -name '*_gen.go' | sort | xargs sha256sum) && go generate ./... && [ "$s" = "$(find . -name '*_gen.go' | sort | xargs sha256sum)" ] ) || exit 1
  ( go run ./internal/cmd/paritycheck -all-done -rescan -lint ) || exit 1
  ( gh pr checks feat/port ) || exit 1
  ;;
final)
  ( test -z "$(gofmt -l .)" ) || exit 1
  ( go vet ./... ) || exit 1
  ( go test ./... ) || exit 1
  ( go test -race $(go list ./... | grep -vE '/(examples|render/wgpu|internal/cmd/wlspike)(/|$)|/platform/linux/(wayland|x11)(/|$)') ) || exit 1
  ( go test -tags gpui_profiler . ) || exit 1
  ( s=$(find . -name '*_gen.go' | sort | xargs sha256sum) && go generate ./... && [ "$s" = "$(find . -name '*_gen.go' | sort | xargs sha256sum)" ] ) || exit 1
  ( for t in darwin/arm64 darwin/amd64 windows/amd64 windows/arm64 js/wasm; do GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./... && GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go test -c -o /dev/null ./... || exit 1; done ) || exit 1
  ( GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet ./internal/apple/... ./platform/apple/... ./platform/ios/... ./platform ) || exit 1
  ( GOOS=js GOARCH=wasm go build -o /dev/null ./examples/web_hello_web && GOOS=js GOARCH=wasm go build ./render/... ./platform/web/... ) || exit 1
  ( env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOMODCACHE="$(go env GOMODCACHE)" GOFLAGS="$GOFLAGS" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./platform/web/... ./render/wgpu/internal/gpu/webgl2/... ) || exit 1
  ( go build -o /dev/null ./examples/... ) || exit 1
  ( go run ./internal/cmd/paritycheck -all-done -rescan -lint ) || exit 1
  ( test ! -e stubs.go && test ! -e DOCS ) || exit 1
  ( gh pr checks feat/port ) || exit 1
  ;;
*)
  echo "unknown stage: $1" >&2
  exit 2
  ;;
esac
