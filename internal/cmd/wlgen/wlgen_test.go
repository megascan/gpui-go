package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden file")

func TestGenerate_Golden(t *testing.T) {
	outs, err := generateAll([]string{filepath.Join("testdata", "golden.xml")}, "wlproto")
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || outs[0].name != "golden_gen.go" {
		t.Fatalf("outputs = %v", outs)
	}
	path := filepath.Join("testdata", "golden_gen.go.golden")
	if *update {
		if err := os.WriteFile(path, outs[0].data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(outs[0].data, want) {
		t.Fatalf("generated output differs from %s; rerun with -update and review:\n%s", path, outs[0].data)
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	a, err := generateAll([]string{filepath.Join("testdata", "golden.xml")}, "wlproto")
	if err != nil {
		t.Fatal(err)
	}
	b, err := generateAll([]string{filepath.Join("testdata", "golden.xml")}, "wlproto")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a[0].data, b[0].data) {
		t.Fatal("two runs differ")
	}
}

func TestGenerate_Signatures(t *testing.T) {
	p, err := parseProtocol(filepath.Join("testdata", "golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	m := &p.Interfaces[0]
	if got := signature(&m.Requests[1]); got != "usun" {
		t.Fatalf("bind_any signature %q", got)
	}
	if got := signature(&m.Requests[2]); got != "n?os?sfiuah" {
		t.Fatalf("create_item signature %q", got)
	}
	if got := strings.Join(typesList(&m.Requests[1]), ","); got != "nil,nil,nil,nil" {
		t.Fatalf("bind_any types %s", got)
	}
}

func writeXML(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bad.xml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGenerate_Errors(t *testing.T) {
	cases := map[string]string{
		"unknown interface": `<protocol name="x"><interface name="a" version="1"><request name="r"><arg name="o" type="object" interface="missing"/></request></interface></protocol>`,
		"unknown enum":      `<protocol name="x"><interface name="a" version="1"><request name="r"><arg name="v" type="uint" enum="b.c"/></request></interface></protocol>`,
		"collision":         `<protocol name="x"><interface name="a_b" version="1"/><interface name="a" version="1"><enum name="b"/></interface></protocol>`,
		"proxy method":      `<protocol name="x"><interface name="a" version="1"><request name="ptr"/></interface></protocol>`,
	}
	for name, body := range cases {
		if _, err := generateAll([]string{writeXML(t, body)}, "p"); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestRun_Directory(t *testing.T) {
	dir := t.TempDir()
	if err := run([]string{"-o", dir, "-pkg", "wlproto", "testdata"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "golden_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "golden_gen.go.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("run output differs from the golden file")
	}
}
