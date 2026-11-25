#!/bin/bash
# B3 s08a expect: every exported AT-SPI method survives hostile args (no panic, no hang) and no data race while the UI thread updates the tree
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s08a 01-bus-hostile-race ./a11y/atspi/bus '^TestSecB3_S08a' -race
exit $?
