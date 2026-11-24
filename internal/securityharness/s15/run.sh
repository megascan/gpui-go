#!/usr/bin/env bash
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
sbx_selftest || exit 99
sbx_go_test -count=1 -v -timeout 600s -tags securityharness -run '^(TestS15_|TestControl)' ./internal/securityharness/s15/ "$@"
