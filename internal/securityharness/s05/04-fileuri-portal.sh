#!/bin/bash
# B3 s05 expect: portal fileURIToPath differential recorded
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s05 04-fileuri-portal ./platform/linux \^TestSecB3_S05
exit $?
