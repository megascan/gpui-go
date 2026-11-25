#!/bin/bash
# B3 s09b expect: declared keymap size larger than the fd never crashes (SIGBUS) the process
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s09b 01-keymap-fd-size ./internal/securityharness/s09b \^TestSecB3_S09b
exit $?
