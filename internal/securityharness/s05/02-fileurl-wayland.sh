#!/bin/bash
# B3 s05 expect: fileURLToPath never panics; NUL/CRLF/.. acceptance recorded (INFO)
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s05 02-fileurl-wayland ./platform/linux/wayland \^TestSecB3_S05
exit $?
