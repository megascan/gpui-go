B3_SPHERES="s05 s06a s06b s07 s08a s08b s09b"
B3_LOCK="$HARNESS/testdata/overlay/.overlay.lock"
b3_overlay() {
  out="$T/overlay-b3.json"
  (cd "$HARNESS" && for s in $B3_SPHERES; do [ -d "$s/testdata/overlay" ] && find "$s/testdata/overlay" -name '*_test.go'; done) |
    jq -Rn --arg proj "$PROJ" --arg h "$HARNESS" '{Replace: ([inputs | {key: ($proj + "/" + sub("^[^/]+/testdata/overlay/"; "")), value: ($h + "/" + .)}] | from_entries)}' >"$out"
  printf '%s' "$out"
}
b3_merge_shared() {
  mine=$(b3_overlay)
  mkdir -p "$HARNESS/testdata/overlay"
  (
    flock 9
    shared="$HARNESS/testdata/overlay/overlay.json"
    [ -s "$shared" ] || echo '{"Replace":{}}' >"$shared"
    jq -s '{Replace: (.[0].Replace + .[1].Replace)}' "$shared" "$mine" >"$shared.b3tmp" && mv "$shared.b3tmp" "$shared"
  ) 9>"$B3_LOCK"
}
b3_gotest() {
  pkg=$1; run=$2; shift 2
  ov=$(b3_overlay)
  sbx_go_test -overlay="$ov" -tags securityharness -count=1 -run "$run" -v "$@" "$pkg"
}
b3_log() { mkdir -p "$HARNESS/$1/logs"; printf '%s' "$HARNESS/$1/logs/$2.log"; }
