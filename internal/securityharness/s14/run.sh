#!/usr/bin/env bash
here=$(cd "$(dirname "$0")" && pwd)
exec "$here/../lib/b1-run.sh" -v -run 'TestS14_' ./internal/securityharness/s14/
