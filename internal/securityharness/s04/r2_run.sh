#!/bin/bash
# R2 S04 expect: OpenURL allowlist holds against case/whitespace; mailto attach params and OpenWithSystem URLs do not reach the launcher
. "$(dirname "$(readlink -f "$0")")/../lib/sandbox.sh"
sbx_selftest || exit 99
o=$HARNESS/s04/testdata/overlay
jq -n --arg p "$PROJ" --arg o "$o" '{Replace: {
 ($p+"/platform/linux/internal/common/s04_security_test.go"): ($o+"/platform/linux/internal/common/s04_security_test.go"),
 ($p+"/platform/linux/internal/common/r2_s04_openurl_test.go"): ($o+"/platform/linux/internal/common/r2_s04_openurl_test.go"),
 ($p+"/platform/linux/s04_security_test.go"): ($o+"/platform/linux/s04_security_test.go"),
 ($p+"/platform/linux/r2_s04_openwithsystem_test.go"): ($o+"/platform/linux/r2_s04_openwithsystem_test.go")}}' >"$T/ov.json"
mkdir -p "$T/gotmp"
sbx env GOTMPDIR="$T/gotmp" nice -n 19 go test -overlay="$T/ov.json" -tags securityharness -count=1 -timeout 120s -v -run '^TestR2_S04_' ./platform/linux/internal/common ./platform/linux
