#!/bin/bash
. "$(dirname "$(readlink -f "$0")")/../lib/sandbox.sh"
sbx_selftest || exit 99
sbx_go_test -tags securityharness -count=1 -timeout 180s -v -run 'TestSecurity_S16_' ./internal/securityharness/s16 2>&1 | tee "$T/out.log" | grep -E '^(=== RUN|--- |    |ok|FAIL|PASS|panic)'
grep -q "^ok" "$T/out.log" || { tail -40 "$T/out.log"; exit 1; }
