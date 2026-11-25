#!/usr/bin/env bash
here=$(cd "$(dirname "$0")" && pwd)
exec "$here/../lib/b1-run.sh" -v -run "${R2_RUN:-TestR2_S01_}" ./internal/securityharness/s01/
