#!/bin/bash
# B3 s05 expect: a trickling pipe peer cannot block ReadFdWithTimeout far beyond its timeout; >256 MiB ends in a clean error
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/../lib/sandbox.sh"
. "$HERE/../lib/b3.sh"
. "$HERE/../lib/b3-run.sh"
b3_run s05 01-readfd-timeout ./platform/linux/internal/common \^TestSecB3_S05
exit $?
