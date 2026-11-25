b3_run() {
  sphere=$1; name=$2; pkg=$3; run=$4; shift 4
  sbx_selftest || exit 99
  LOG=$(b3_log "$sphere" "$name")
  b3_gotest "$pkg" "$run" -timeout 300s "$@" >"$LOG" 2>&1
  rc=$?
  grep -E '^(=== RUN|\s*--- |FAIL|ok |PASS|panic:)|FAIL:|fatal error|signal SIG' "$LOG" | tail -60
  return $rc
}
