//go:build securityharness

package s18

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

const hyprctlStub = `#!/bin/sh
D=$(dirname "$0")
case "$1" in
eval)
  n=$(ls "$D"/lua-*.txt 2>/dev/null | wc -l)
  printf '%s' "$2" >"$D/lua-$n.txt"
  echo ok
  ;;
clients)
  pid=$(sed -n 's/^local P, X11, KEY, SPECIAL, W, H = \([0-9]*\),.*/\1/p' "$D/lua-0.txt")
  sed "s/PIDX/$pid/g" "$D/clients.json"
  ;;
esac
`

const goStub = `#!/bin/sh
D=$(dirname "$0")
printf '%s\n' "$@" >"$D/go-argv.txt"
printf '#!/bin/sh\nexec sleep 30\n' >"$3"
chmod +x "$3"
`

const grimStub = `#!/bin/sh
D=$(dirname "$0")
printf '%s\n' "$@" >"$D/grim-argv.txt"
`

func TestS18_ShootHostileHyprctlJSON(t *testing.T) {
	work := t.TempDir()
	stubs := filepath.Join(work, "stubs")
	if err := os.MkdirAll(stubs, 0o755); err != nil {
		t.Fatal(err)
	}
	shoot := filepath.Join(work, "shoot")
	build := exec.Command("go", "build", "-o", shoot, "../../cmd/shoot")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("harness-error: building shoot: %v\n%s", err, out)
	}
	pwn := filepath.Join(work, "pwn")
	stableID := `-o` + pwn + `";$(touch ` + pwn + `)` + "`touch " + pwn + "`'é\x01"
	clients := `[{"address":"0xdead","mapped":true,"at":[0,0],"size":[800,600],"pid":PIDX,"xwayland":false,"floating":true,` +
		`"workspace":{"id":-99,"name":"special:gpui-shoot-PIDX"},"stableId":` + jsonString(stableID) + `}]`
	for name, body := range map[string]string{"hyprctl": hyprctlStub, "go": goStub, "grim": grimStub, "clients.json": clients} {
		mode := os.FileMode(0o755)
		if strings.HasSuffix(name, ".json") {
			mode = 0o644
		}
		if err := os.WriteFile(filepath.Join(stubs, name), []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(work, "out", "shot.png")
	cmd := exec.Command(shoot, "-pkg", "./examples/hello_world", "-out", out, "-settle", "100ms", "-timeout", "15s")
	cmd.Env = append(os.Environ(), "PATH="+stubs+":/usr/bin")
	cmd.Dir = work
	output, err := cmd.CombinedOutput()
	t.Logf("shoot exit=%v output=%s", err, output)
	if err != nil {
		t.Fatalf("harness-error: shoot failed against stubs")
	}
	grimArgv, _ := os.ReadFile(filepath.Join(stubs, "grim-argv.txt"))
	got := strings.Split(strings.TrimSuffix(string(grimArgv), "\n"), "\n")
	want := []string{"-T", stableID, out}
	t.Logf("grim argv=%q", got)
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("ARGV: grim received %q, want %q", got, want)
	}
	if _, err := os.Stat(pwn); err == nil {
		t.Errorf("INJECTION: hostile stableId executed a command")
	}
	luas, _ := filepath.Glob(filepath.Join(stubs, "lua-*.txt"))
	for _, p := range luas {
		b, _ := os.ReadFile(p)
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "KEY") && strings.Contains(line, "local P") || strings.Contains(line, "_G.__gpui_shoot") && strings.Contains(line, `"`) && i < 4 {
				t.Logf("%s: %s", filepath.Base(p), line)
			}
		}
		for _, r := range string(b) {
			if r >= utf8.RuneSelf {
				t.Errorf("LUA: non-ASCII rune %q in generated Lua; Go %%q escapes may differ from Lua", r)
				break
			}
		}
		if strings.Contains(string(b), pwn) || strings.Contains(string(b), "\\u") || strings.Contains(string(b), "\\U") {
			t.Errorf("LUA: compositor-derived or Go-only escape in generated Lua")
		}
	}
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte("0123456789abcdef"[r>>4])
			b.WriteByte("0123456789abcdef"[r&0xf])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
