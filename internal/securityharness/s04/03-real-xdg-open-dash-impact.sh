#!/bin/bash
. "$(dirname "$(readlink -f "$0")")/../lib/sandbox.sh"
sbx_selftest || exit 99
for a in --help --version -x ./--help; do
  : >"$T/f"; out=$(sbx sh -c 'cd "$T" && : >./--help && /usr/bin/xdg-open "$1" 2>&1; echo "rc=$?"' sh "$a" | tr '\n' ' ' | cut -c1-160)
  echo "real xdg-open argv[1]=$a -> $out"
done
