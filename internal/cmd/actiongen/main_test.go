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

func TestActiongen_Golden(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*.go")
	if err != nil {
		t.Fatal(err)
	}
	selfInputs, err := filepath.Glob("testdata/self/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		files []string
		self  bool
	}{{inputs, false}, {selfInputs, true}} {
		for _, in := range c.files {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Generate(in, src, c.self)
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
}

func TestActiongen_NoDirective(t *testing.T) {
	got, err := Generate("x.go", []byte("package x\n\ntype A struct{ B int }\n"), false)
	if err != nil || got != nil {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestActiongen_Errors(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want string
	}{
		{"non-struct", "package x\n\n//gpui:action\ntype A int\n", "only struct types can be actions"},
		{"generic", "package x\n\n//gpui:action\ntype A[T any] struct{ V T }\n", "generic types cannot be actions"},
		{"unknown key", "package x\n\n//gpui:action bogus\ntype A struct{}\n", "'bogus' argument not recognized"},
		{"duplicate key", "package x\n\n//gpui:action no_json no_json\ntype A struct{}\n", "'no_json' argument specified multiple times"},
		{"name with ::", "package x\n\n//gpui:action name=ed::Move\ntype A struct{}\n", "must not contain '::'"},
		{"bad namespace", "package x\n\n//gpui:action namespace=ed.x\ntype A struct{}\n", "namespace must be an identifier"},
		{"value on flag", "package x\n\n//gpui:action no_json=yes\ntype A struct{}\n", "'no_json' takes no value"},
		{"unterminated quote", "package x\n\n//gpui:action deprecated=\"oops\ntype A struct{}\n", "unterminated quoted value"},
		{"misplaced", "package x\n\n//gpui:action\n\ntype A struct{}\n", "misplaced //gpui:action directive"},
		{"collides with import", "package x\n\n//gpui:action\ntype json struct{}\n", "collides with a generated import"},
		{"two directives", "package x\n\n//gpui:action\n//gpui:action no_json\ntype A struct{}\n", "multiple //gpui:action directives"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Generate("x.go", []byte(c.src), false)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want containing %q", err, c.want)
			}
		})
	}
}

func TestActiongen_ParseArguments(t *testing.T) {
	args, err := parseArguments(` namespace=editor name="Select Next" no_json deprecated="a b"`)
	if err != nil {
		t.Fatal(err)
	}
	want := []argument{
		{key: "namespace", value: "editor", hasValue: true},
		{key: "name", value: "Select Next", hasValue: true},
		{key: "no_json"},
		{key: "deprecated", value: "a b", hasValue: true},
	}
	if len(args) != len(want) {
		t.Fatalf("args = %+v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg %d = %+v, want %+v", i, args[i], want[i])
		}
	}
}
