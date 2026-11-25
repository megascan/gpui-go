#!/usr/bin/env bash
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
sbx_selftest || exit 99
mkdir -p "$T/gotmp"
sbx env GOTMPDIR="$T/gotmp" nice -n 19 go test -count=1 -v -timeout 600s -tags securityharness -run "${R2_RUN:-TestR2_S15_}" ./internal/securityharness/s15/ "$@"
