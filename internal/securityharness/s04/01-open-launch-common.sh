#!/bin/bash
. "$(dirname "$(readlink -f "$0")")/../lib/sandbox.sh"
. "$HARNESS/lib/overlay-subset.sh"
sbx_selftest || exit 99
overlay_subset "$HARNESS/s04/"
sbx_go_test -overlay="$T/overlay.json" -tags securityharness -count=1 -timeout 120s -v -run 'TestSecurity_S04_' ./platform/linux/internal/common 2>&1 | tee "$T/out.log" | grep -E '^(=== RUN|--- |    |ok|FAIL|PASS|panic)' 
grep -q "^ok" "$T/out.log" || { tail -40 "$T/out.log"; exit 1; }
