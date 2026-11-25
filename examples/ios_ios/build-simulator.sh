#!/usr/bin/env bash

set -euo pipefail

repository_root="$(git rev-parse --show-toplevel)"
example_directory="$repository_root/examples/ios_ios"
target_directory="${GPUI_TARGET_DIR:-"$repository_root/target"}"
if [[ "$target_directory" != /* ]]; then
    target_directory="$repository_root/$target_directory"
fi
app_directory="$target_directory/gpui-ios-example/GPUIIosExample.app"
build_directory="$target_directory/gpui-ios-build"

sdk_path="$(xcrun --sdk iphonesimulator --show-sdk-path)"
clang="$(xcrun --sdk iphonesimulator --find clang)"
target="arm64-apple-ios15.0-simulator"

cd "$repository_root"
mkdir -p "$build_directory" "$app_directory"

GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
    CC="$clang" \
    CGO_CFLAGS="-isysroot $sdk_path -target $target" \
    CGO_LDFLAGS="-isysroot $sdk_path -target $target" \
    go build -buildmode=c-archive -o "$build_directory/libgpui_ios_example.a" "$@" ./examples/ios_ios

"$clang" -isysroot "$sdk_path" -target "$target" \
    "$example_directory/app/main.c" "$build_directory/libgpui_ios_example.a" \
    -framework UIKit -framework Foundation -framework CoreFoundation -framework QuartzCore \
    -framework Metal -framework CoreText -framework CoreGraphics -framework Security \
    -lresolv \
    -o "$app_directory/gpui_ios_example"
cp "$example_directory/app/Info.plist" "$app_directory/Info.plist"
codesign --force --sign - "$app_directory"
printf 'Built simulator app: %s\n' "$app_directory"
