#!/bin/bash
# B3 s06a expect: RESOURCE_MANAGER #include of /dev/zero, a FIFO, or fan-out files never hangs or exhausts memory; parser never panics
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s06a 01-xrm-include ./platform/linux/x11/internal/xrm \^TestSecB3_S06a
exit $?
