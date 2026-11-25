#!/bin/bash
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s08b 02-paragraph-livelock ./a11y/atspi \^TestSecV1_S08b
exit $?
