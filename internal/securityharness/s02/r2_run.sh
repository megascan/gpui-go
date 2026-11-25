#!/usr/bin/env bash
here=$(cd "$(dirname "$0")" && pwd)
exec "$here/../lib/b1-run.sh" -v -run "${R2_RUN:-TestR[23]_S02_}" ./internal/securityharness/s02/
