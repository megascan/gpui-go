#!/bin/bash
set -u
H=$(cd "$(dirname "$0")" && pwd)
OUT=${OUT:-$(mktemp -d "${SWEEP_TMPROOT:-/tmp}/sweep-r2.XXXXXX")}
fail=0
for s in s01/r2_run.sh s02/r2_run.sh s04/r2_run.sh s07/r2_screencap.sh s09b/r2_run.sh s15/r2_run.sh; do
  log="$OUT/$(printf '%s' "$s" | tr / _).log"
  bash "$H/$s" >"$log" 2>&1
  rc=$?
  case $rc in 0) r=PASS ;; 99) r=CANARY ;; *) r=FAIL ;; esac
  [ "$rc" = 0 ] || fail=1
  [ "$rc" = 99 ] && { echo "$s CANARY"; echo "logs: $OUT"; exit 99; }
  echo "$s $r"
done
echo "logs: $OUT"
exit $fail
