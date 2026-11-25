#!/bin/bash
# B3 s06a expect: xim.Decode and CompoundTextToUTF8 never panic on mutated frames
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s06a 03-xim-decode ./platform/linux/x11/internal/xim \^TestSecB3_S06a
exit $?
