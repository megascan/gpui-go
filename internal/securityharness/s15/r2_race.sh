#!/bin/bash
. "$(dirname "$(readlink -f "$0")")/../lib/sandbox.sh"
sbx_selftest || exit 99
mkdir -p "$T/gotmp"
sbx env GOTMPDIR="$T/gotmp" nice -n 19 go test -race -count=1 -timeout 900s ./scheduler ./platform/linux/internal/common ./platform/linux/internal/dbus ./text/gotext ./a11y/consumer ./internal/linux/xkb ./platform/linux/x11/internal/xrm ./platform/linux/x11/internal/xcursor . 2>&1 | tee "$T/race.log" | grep -E '^(ok|FAIL|---|panic|WARNING: DATA RACE)' 
grep -q -E '^FAIL|DATA RACE' "$T/race.log" && { grep -B2 -A40 'DATA RACE' "$T/race.log" | head -120; exit 1; }
exit 0
