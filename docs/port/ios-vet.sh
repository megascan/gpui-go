#!/bin/sh
set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
tmp=$(mktemp -d)
trap 'chmod -R u+w "$tmp"; rm -rf "$tmp"' EXIT
src=$(go env GOMODCACHE)/github.com/ebitengine/purego@v0.11.3
cp -r "$src" "$tmp/purego" && chmod -R u+w "$tmp/purego" && rm -f "$tmp/purego/is_ios.go"
printf 'go 1.27.3\nuse %s\nreplace github.com/ebitengine/purego v0.11.3 => %s/purego\n' "$root" "$tmp" > "$tmp/go.work"
cd "$root"
GOWORK="$tmp/go.work" GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go vet -tags ios "$@"
