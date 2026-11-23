package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestRefinegen_Golden(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range inputs {
		src, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Generate(in, src)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		golden := strings.TrimSuffix(in, ".go") + "_gen.go.golden"
		if *update {
			if err := os.WriteFile(golden, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: output differs from %s:\n%s", in, golden, got)
		}
	}
}

func TestRefinegen_SampleUpToDate(t *testing.T) {
	src, err := os.ReadFile("internal/sample/sample.go")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Generate("sample.go", src)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("internal/sample/sample_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("internal/sample/sample_gen.go is stale; run go generate")
	}
}

func TestRefinegen_NoDirective(t *testing.T) {
	got, err := Generate("x.go", []byte("package x\n\ntype A struct{ B int }\n"))
	if err != nil || got != nil {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestRefinegen_RejectsNonStruct(t *testing.T) {
	_, err := Generate("x.go", []byte("package x\n\n//gpui:refineable\ntype A int\n"))
	if err == nil || err.Error() != "This derive macro only supports structs with named fields" {
		t.Fatalf("err = %v", err)
	}
}

func TestRefinegen_SnakeCase(t *testing.T) {
	for in, want := range map[string]string{"BorderWidth": "border_width", "ID": "id", "TextID": "text_id", "HTTPServer": "http_server", "W1": "w1"} {
		if got := snakeCase(in); got != want {
			t.Errorf("snakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}
