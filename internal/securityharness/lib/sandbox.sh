PROJ=${PROJ:-/home/strange/Projects/gpui-go-wt/sec}
HARNESS=${HARNESS:-$PROJ/internal/securityharness}
SWEEP_GOMODCACHE=${SWEEP_GOMODCACHE:-/home/strange/go/pkg/mod}
SWEEP_GOCACHE=${SWEEP_GOCACHE:-/home/strange/.cache/security-sweep/gpui-go/gocache}
mkdir -p "$SWEEP_GOCACHE"
T=$(mktemp -d "${SWEEP_TMPROOT:-/tmp}/sweep.XXXXXX"); S="$T/stubs"; mkdir -p "$S" "$T/home" "$T/run" "$T/tmp"; : >"$T/canary.log"; : >"$T/calls.log"
for c in xdg-open gio gnome-open kde-open kde-open5 kde-open6 hyprctl grim notify-send systemctl systemd-run busctl gdbus dbus-send dbus-launch loginctl avahi-browse avahi-resolve; do
  printf '#!/bin/sh\necho "%s $*" >>"%s/calls.log"\n' "$c" "$T" >"$S/$c"; chmod +x "$S/$c"; done
sbx() { bwrap --ro-bind /usr /usr --symlink usr/bin /bin --symlink usr/lib /lib --symlink usr/lib64 /lib64 \
  --ro-bind /etc /etc --proc /proc --dev /dev --tmpfs /tmp --tmpfs /run --tmpfs /home \
  --ro-bind "$SWEEP_GOMODCACHE" "$SWEEP_GOMODCACHE" --bind "$SWEEP_GOCACHE" "$SWEEP_GOCACHE" \
  --ro-bind "$PROJ" "$PROJ" --bind "$T" "$T" --chdir "$PROJ" \
  --unshare-all --die-with-parent --new-session --clearenv \
  --setenv HOME "$T/home" --setenv XDG_RUNTIME_DIR "$T/run" --setenv TMPDIR "$T/tmp" --setenv PATH "$S:/usr/bin" --setenv T "$T" \
  --setenv LANG C.UTF-8 --setenv GOENV off --setenv GOTOOLCHAIN local --setenv GOPROXY off --setenv GOFLAGS -mod=readonly \
  --setenv GOMODCACHE "$SWEEP_GOMODCACHE" --setenv GOCACHE "$SWEEP_GOCACHE" --setenv GOPATH "$T/gopath" \
  --setenv GOTELEMETRY off --setenv SWEEP_LOOPBACK_ONLY 1 --unsetenv XDG_RUNTIME_DIR --setenv XDG_RUNTIME_DIR "$T/run" "$@"; }
sbx_go_test() { sbx nice -n 19 go test "$@"; }
canary() {
  sbx sh -c 'ip -o addr 2>/dev/null | grep -v " lo " && exit 1; [ -e /home/strange/.config ] && exit 2; [ -e /run/user ] && exit 3; [ -e /tmp/.X11-unix ] && exit 4; exit 0' || echo "CANARY: sandbox isolation check failed ($?)" >>"$T/canary.log"
  grep -E '[a-z]+://' "$T/calls.log" 2>/dev/null | grep -vE '://(127\.|localhost|\[::1\])' | sed 's/^/CANARY: stub got non-loopback URL: /' >>"$T/canary.log"
  [ ! -s "$T/canary.log" ] || { cat "$T/canary.log"; return 99; }
}
sbx_selftest() { if sbx curl -sS -m3 http://1.1.1.1/ >/dev/null 2>&1; then echo "CANARY: non-loopback connect succeeded" >>"$T/canary.log"; return 99; fi; return 0; }
sweep_exit() { rc=$?; canary || rc=99; rm -rf "$T"; exit $rc; }
trap sweep_exit EXIT
