#!/bin/bash
. "$(dirname "$(readlink -f "$0")")/../lib/sandbox.sh"
. "$HARNESS/lib/overlay-subset.sh"
sbx_selftest || exit 99
overlay_subset "$HARNESS/s16/"
sbx_go_test -overlay="$T/overlay.json" -tags securityharness -count=1 -timeout 180s -v -run 'TestSecurity_S16_' ./examples/internal/support 2>&1 | tee "$T/out.log" | grep -E '^(=== RUN|--- |    |ok|FAIL|PASS|panic)'
grep -q "^ok" "$T/out.log" || { tail -40 "$T/out.log"; exit 1; }
