#!/bin/bash
# R2 S09b expect: shrinking the keymap memfd after NewKeymapFromFd's fstat never SIGBUSes the client
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
sbx_selftest || exit 99
mkdir -p "$T/gotmp"
sbx env GOTMPDIR="$T/gotmp" nice -n 19 go test -count=1 -v -timeout 300s -tags securityharness -run '^TestR2_S09b_' ./internal/securityharness/s09b/
