#!/bin/bash
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
sbx_selftest || exit 99
jq -n --arg p "$PROJ/text/gotext/v1_s03_strike_test.go" --arg h "$HARNESS/s03/testdata/overlay/text/gotext/v1_s03_strike_test.go" '{Replace: {($p): $h}}' >"$T/overlay.json"
sbx_go_test -overlay="$T/overlay.json" -tags securityharness -count=1 -timeout 120s -v -run '^TestSecV1_S03' ./text/gotext 2>&1 | grep -E '^(=== RUN|\s*--- |\s+v1_|ok |FAIL|PASS|panic:)'
exit ${PIPESTATUS[0]}
