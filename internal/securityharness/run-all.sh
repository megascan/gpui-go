#!/bin/bash
set -u
H=$(cd "$(dirname "$0")" && pwd)
OUT=${OUT:-$(mktemp -d /tmp/sweep-runall.XXXXXX)}
fail=0
for s in s01/run.sh s02/run.sh s03/run.sh s03/01-strike-overlay.sh s04/01-open-launch-common.sh s04/02-open-with-system.sh s04/03-real-xdg-open-dash-impact.sh s05/01-readfd-timeout.sh s05/02-fileurl-wayland.sh s05/03-fileurl-x11.sh s05/04-fileuri-portal.sh s05/05-mime-quadratic.sh s06a/01-xrm-include.sh s06a/02-xcursor.sh s06a/03-xim-decode.sh s06b/01-incr-mindatalen.sh s07/01-dbus-session-services.sh s07/02-filechooser-portal.sh s08a/01-bus-hostile-race.sh s08b/01-platformnode-args.sh s08b/02-paragraph-livelock.sh s09b/01-keymap-fd-size.sh s13/run.sh s14/run.sh s15/run.sh s16/01-image-loader.sh s16/02-support-client.sh s18/run.sh s19/run.sh; do
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
