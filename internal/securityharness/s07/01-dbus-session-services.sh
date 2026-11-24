#!/bin/bash
# B3 S07 expect: notification/secret-service signals are accepted only from the service owner; no plaintext secret on the bus; malformed encrypted replies error without panic
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"


b3_run s07 01-dbus-session-services ./platform/linux/internal/dbus "^TestSecB3_S07"


exit $?
