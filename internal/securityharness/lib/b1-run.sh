#!/usr/bin/env bash
set -u
here=$(cd "$(dirname "$0")" && pwd)
. "$here/sandbox.sh"
b1_netcheck() {
  out=$(sbx bash -c 'cat /proc/net/dev | tail -n +3 | cut -d: -f1 | tr -d " "; exec 3<>/dev/tcp/1.1.1.1/80' 2>&1)
  ifaces=$(printf '%s\n' "$out" | grep -vE 'unreachable|/dev/tcp|connect|^$' )
  if [ "$ifaces" != "lo" ]; then echo "CANARY: interfaces in sandbox: $ifaces" >>"$T/canary.log"; return 99; fi
  printf '%s\n' "$out" | grep -qi 'unreachable' || { echo "CANARY: non-loopback connect not refused as unreachable: $out" >>"$T/canary.log"; return 99; }
  sbx_selftest || return 99
  return 0
}
b1_netcheck || exit 99
mkdir -p "$T/gotmp" "$T/evidence"
sbx env GOTMPDIR="$T/gotmp" SECHARNESS_EVIDENCE="$T/evidence" nice -n 19 go test -tags securityharness -count=1 -timeout 900s "$@"
rc=$?
if [ -n "${B1_EVIDENCE:-}" ]; then mkdir -p "$B1_EVIDENCE"; cp -r "$T/evidence/." "$B1_EVIDENCE/"; cp "$T/calls.log" "$B1_EVIDENCE/calls.log"; fi
exit $rc
