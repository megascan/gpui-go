#!/bin/bash
# B3 s07 expect: FileChooser Response accepted only from the portal owner
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s07 02-filechooser-portal ./platform/linux \^TestSecB3_S07
exit $?
