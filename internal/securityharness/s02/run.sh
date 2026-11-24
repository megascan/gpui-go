#!/usr/bin/env bash
here=$(cd "$(dirname "$0")" && pwd)
exec "$here/../lib/b1-run.sh" -v -run 'TestS02_' ./internal/securityharness/s02/
