#!/bin/bash
# R2 S07 expect: screencap portal Request.Response is accepted only from the portal owner
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
sbx_selftest || exit 99
f=platform/linux/internal/screencap/r2_s07_screencap_test.go
printf '{"Replace":{"%s/%s":"%s/s07/testdata/overlay/%s"}}' "$PROJ" "$f" "$HARNESS" "$f" >"$T/ov.json"
mkdir -p "$T/gotmp"
sbx env GOTMPDIR="$T/gotmp" nice -n 19 go test -overlay="$T/ov.json" -tags securityharness -count=1 -timeout 60s -v -run '^TestR2_S07_' ./platform/linux/internal/screencap
