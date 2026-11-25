#!/bin/bash
# B3 s08b expect: no PlatformNode method panics or leaks the tree lock on hostile offsets/indices/coord types
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s08b 01-platformnode-args ./a11y/atspi \^TestSecB3_S08b
exit $?
