#!/bin/bash
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/overlay-subset.sh"
sbx_selftest || exit 99
jq -n --arg p "$PROJ/platform/linux/x11/v1_s06b_incr_test.go" --arg h "$HARNESS/s06b/testdata/overlay/platform/linux/x11/v1_s06b_incr_test.go" '{Replace: {($p): $h}}' >"$T/overlay.json"
mkdir -p "$HARNESS/s06b/logs"
sbx bash -c 'mkdir -p /tmp/.X11-unix; Xvfb :99 -nolisten tcp -extension GLX >"$T/xvfb.log" 2>&1 & x=$!; for i in $(seq 50); do [ -e /tmp/.X11-unix/X99 ] && break; sleep 0.1; done; DISPLAY=:99 nice -n 19 go test -overlay="$T/overlay.json" -tags securityharness -count=1 -timeout 300s -v -run "^TestSecV1_S06b" ./platform/linux/x11; rc=$?; kill $x; exit $rc' >"$HARNESS/s06b/logs/01-incr-mindatalen.log" 2>&1
rc=$?
grep -E '^(=== RUN|\s*--- |FAIL|ok |PASS|panic:)|UNSAFE|child ' "$HARNESS/s06b/logs/01-incr-mindatalen.log" | tail -30
exit $rc
