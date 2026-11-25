overlay_subset() { jq --arg p "$1" '{Replace: (.Replace | with_entries(select(.value | startswith($p))))}' "$HARNESS/testdata/overlay/overlay.json" >"$T/overlay.json"; }
