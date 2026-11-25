#!/bin/bash
# B3 s06a expect: xcursor header dims/TOC never allocate beyond the file content; parser never panics
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s06a 02-xcursor ./platform/linux/x11/internal/xcursor \^TestSecB3_S06a
exit $?
