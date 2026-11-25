#!/bin/bash
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s05 05-mime-quadratic ./platform/linux/wayland \^TestSecV1_S05
exit $?
