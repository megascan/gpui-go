#!/bin/bash
# B3 s05 expect: urlToFilePath differential recorded
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s05 03-fileurl-x11 ./platform/linux/x11 \^TestSecB3_S05
exit $?
