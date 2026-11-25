package main

import (
	"bufio"
	"bytes"
	"flag"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

var goldenSets = map[string][]string{
	"visibility_overflow_cursor": {"visibility_style_methods", "overflow_style_methods", "cursor_style_methods"},
	"padding":                    {"padding_style_methods"},
	"position":                   {"position_style_methods"},
	"border_box_shadow":          {"border_style_methods", "box_shadow_style_methods"},
}

func TestStylegen_Golden(t *testing.T) {
	for name, set := range goldenSets {
		got, err := Generate("gpui", set)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		golden := filepath.Join("testdata", name+"_gen.go.golden")
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
			t.Errorf("%s: output differs from %s; run go test -update", name, golden)
		}
	}
}

func readTSV(t *testing.T) (rust, gonames []string) {
	f, err := os.Open("../../../docs/port/parity/styled-methods.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		rust = append(rust, cols[0])
		gonames = append(gonames, cols[1])
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rust, gonames
}

func TestStylegen_MethodsMatchTSV(t *testing.T) {
	rust, gonames := readTSV(t)
	if len(rust) != 3099 {
		t.Fatalf("styled-methods.tsv has %d rows, want 3099", len(rust))
	}
	got := MethodNames(nil)
	if len(got) != len(rust) {
		t.Fatalf("generated %d methods, want %d", len(got), len(rust))
	}
	for i := range got {
		if got[i] != rust[i] {
			t.Fatalf("method %d: got %q, want %q", i, got[i], rust[i])
		}
		if goName(got[i]) != gonames[i] {
			t.Fatalf("method %q: Go name %q, want %q", got[i], goName(got[i]), gonames[i])
		}
	}
	src, err := Generate("gpui", nil)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "styled_gen.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var decls []string
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			decls = append(decls, fd.Name.Name)
			if fd.Doc == nil || !strings.HasPrefix(fd.Doc.Text(), fd.Name.Name+" ") {
				t.Errorf("%s: godoc must start with the method name", fd.Name.Name)
			}
		}
	}
	if strings.Join(decls, ",") != strings.Join(gonames, ",") {
		t.Fatalf("declared methods differ from styled-methods.tsv")
	}
	if !bytes.HasPrefix(src, []byte(header+"\n")) {
		t.Fatalf("missing generated-file header")
	}
}

func TestStylegen_TypeChecksAgainstContract(t *testing.T) {
	src, err := Generate("gpui", nil)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile("testdata/contract/contract.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for name, b := range map[string][]byte{"contract.go": contract, "styled_gen.go": src} {
		f, err := parser.ParseFile(fset, name, b, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check("github.com/megascan/gpui-go", fset, files, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStylegen_GoName(t *testing.T) {
	for in, want := range map[string]string{
		"w_0":               "W0",
		"w_neg_1_2":         "WNeg1_2",
		"min_size_1_12":     "MinSize1_12",
		"rounded_2xl":       "Rounded2xl",
		"border_t_0":        "BorderT0",
		"overflow_x_hidden": "OverflowXHidden",
	} {
		if got := goName(in); got != want {
			t.Errorf("goName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStylegen_Bodies(t *testing.T) {
	src, err := Generate("gpui", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"style.Margin.Top = gpuiutil.Some((-Px(0.)).ToLength())",
		"style.Size.Width = gpuiutil.Some(Relative(1. / 3.).Neg().ToLength())",
		"style.Padding.Left = gpuiutil.Some(Rems(0.125).ToDefiniteLength())",
		"style.CornerRadii.TopLeft = gpuiutil.Some(Px(9999.).ToAbsoluteLength())",
		"style.Inset.Top = gpuiutil.Some(Auto().ToLength())",
		"func (s *Styled[S]) MtAuto() S {",
	} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("generated source lacks %q", want)
		}
	}
	for _, absent := range []string{"PAuto", "MNegAuto", "GapAuto", "RoundedNeg", "BorderNeg"} {
		if bytes.Contains(src, []byte(") "+absent+"() S")) {
			t.Errorf("generated source must not declare %s", absent)
		}
	}
}
